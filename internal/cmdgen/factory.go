package cmdgen

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/spf13/cobra"
)

// PendingUpdates tracks background goroutines for stale spec refreshes.
var PendingUpdates sync.WaitGroup

// BuildTopicCommand creates a parent command for a topic (e.g., "issue", "cycle").
// It registers subcommands in Mode A (cached spec) or Mode B (lazy).
func BuildTopicCommand(topicName string, topic *docs.Topic, cachedSpecs []docs.CachedSpec, deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   topicName,
		Short: fmt.Sprintf("Manage %s resources", topicName),
	}

	// Index cached specs by entry title for fast lookup
	specByTitle := make(map[string]*docs.CachedSpec)
	for i := range cachedSpecs {
		specByTitle[cachedSpecs[i].Spec.EntryTitle] = &cachedSpecs[i]
	}

	for _, entry := range topic.Entries {
		if !IsAPIReferenceURL(entry.URL) {
			continue
		}

		cmdName := DeriveSubcommandName(entry.Title, topicName)
		if cmdName == "" {
			continue
		}

		if cached, ok := specByTitle[entry.Title]; ok {
			// Mode A: full spec available
			sub := BuildEndpointCommand(topicName, cmdName, &cached.Spec, deps)
			cmd.AddCommand(sub)

			// Background refresh if stale
			if cached.IsStale() {
				refreshSpecInBackground(topicName, entry, deps)
			}
		} else {
			// Mode B: lazy command
			sub := BuildLazyCommand(topicName, cmdName, entry, deps)
			cmd.AddCommand(sub)
		}
	}

	// Register ensure subcommand if the topic has the required specs
	if TopicSupportsEnsure(topicName) {
		if specs := findEnsureSpecs(topicName, cachedSpecs); specs != nil {
			cmd.AddCommand(BuildEnsureCommand(topicName, specs, deps))
		}
	}

	return cmd
}

// globalFlagNames contains flag names used by the root command's persistent flags.
// Spec params with these names are skipped to avoid conflicts.
var globalFlagNames = map[string]bool{
	"workspace": true, "project": true, "output": true,
	"api-url": true, "api-key": true, "verbose": true,
	"quiet": true, "strict": true, "no-resolve": true,
	"per-page": true, "cursor": true, "all": true,
	"dry-run": true, "help": true,
	"field": true, "fields": true,
	"id-only": true,
}

// BuildEndpointCommand creates a fully-flagged command from a cached spec (Mode A).
func BuildEndpointCommand(topicName, cmdName string, spec *docs.EndpointSpec, deps *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdName,
		Short: spec.EntryTitle,
		Long:  fmt.Sprintf("%s\n\nAPI: %s %s\nDocs: %s", spec.EntryTitle, spec.Method, spec.PathTemplate, spec.SourceURL),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExecuteSpec(cmd.Context(), cmd, spec, deps)
		},
	}

	// Register flags from spec params
	for _, p := range spec.Params {
		if p.Name == "workspace_slug" || p.Name == "project_id" {
			continue
		}
		flagName := ParamToFlagName(p.Name)
		if globalFlagNames[flagName] {
			continue
		}
		desc := p.Description
		if desc == "" {
			desc = p.Name
		}

		// For _html params, register both --description (markdown) and
		// --description-html (raw HTML escape hatch) as mutually exclusive.
		if IsHTMLParam(p.Name) {
			mdFlag := MarkdownFlagName(p.Name)
			mdDesc := desc
			if mdDesc == p.Name {
				mdDesc = mdFlag
			}
			cmd.Flags().String(mdFlag, "", mdDesc+" (markdown)")
			cmd.Flags().String(flagName, "", mdDesc+" (raw HTML)")
			cmd.MarkFlagsMutuallyExclusive(mdFlag, flagName)
			continue
		}

		switch p.Type {
		case "string[]":
			cmd.Flags().StringSlice(flagName, nil, desc)
		case "number":
			cmd.Flags().Int(flagName, 0, desc)
		case "boolean":
			cmd.Flags().Bool(flagName, false, desc)
		default:
			cmd.Flags().String(flagName, "", desc)
		}

		if p.Required {
			cmd.MarkFlagRequired(flagName)
		}
	}

	return cmd
}

// BuildLazyCommand creates a command with DisableFlagParsing that fetches its
// spec on first use (Mode B).
func BuildLazyCommand(topicName, cmdName string, entry docs.Entry, deps *Deps) *cobra.Command {
	return &cobra.Command{
		Use:                cmdName,
		Short:              entry.Title,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check for --help / -h
			if IsHelpRequested(args) {
				return lazyHelp(cmd, topicName, cmdName, entry, deps)
			}

			Infof(deps, "Fetching API spec for '%s %s'...\n", topicName, cmdName)

			spec, err := fetchAndCacheSpec(cmd.Context(), topicName, entry, deps)
			if err != nil {
				return fmt.Errorf("fetching spec: %w", err)
			}

			// Parse raw args against the spec
			parsed, err := ParseRawArgs(args, spec.Params)
			if err != nil {
				return err
			}

			// Apply global flags from parsed args to the root command's flags
			applyGlobalFlags(cmd, parsed)

			// Execute
			if err := ExecuteSpecFromArgs(cmd.Context(), spec, parsed, deps); err != nil {
				return err
			}

			Infof(deps, "hint: spec cached for 'plane %s %s'. Run again for full flag support and --help.\n", topicName, cmdName)
			return nil
		},
	}
}

func lazyHelp(cmd *cobra.Command, topicName, cmdName string, entry docs.Entry, deps *Deps) error {
	Infof(deps, "Fetching API spec for '%s %s'...\n", topicName, cmdName)

	spec, err := fetchAndCacheSpec(cmd.Context(), topicName, entry, deps)
	if err != nil {
		return fmt.Errorf("fetching spec for help: %w", err)
	}

	GenerateHelp(os.Stdout, topicName, cmdName, spec)
	return nil
}

func fetchAndCacheSpec(ctx context.Context, topicName string, entry docs.Entry, deps *Deps) (*docs.EndpointSpec, error) {
	markdown, err := docs.Fetch(ctx, entry.URL)
	if err != nil {
		return nil, err
	}

	spec := docs.ParseEndpointPage(markdown, topicName, entry)
	if spec == nil {
		return nil, fmt.Errorf("could not parse spec from %s", entry.URL)
	}

	// Best-effort cache write
	_ = docs.WriteSpec(deps.Profile, deps.BaseURL, spec)

	return spec, nil
}

func refreshSpecInBackground(topicName string, entry docs.Entry, deps *Deps) {
	PendingUpdates.Add(1)
	go func() {
		defer PendingUpdates.Done()
		ctx := context.Background()
		markdown, err := docs.Fetch(ctx, entry.URL)
		if err != nil {
			return
		}
		spec := docs.ParseEndpointPage(markdown, topicName, entry)
		if spec != nil {
			_ = docs.WriteSpec(deps.Profile, deps.BaseURL, spec)
		}
	}()
}

// applyGlobalFlags sets global flags from parsed args on the root command.
func applyGlobalFlags(cmd *cobra.Command, parsed *ParsedArgs) {
	root := cmd.Root()
	pf := root.PersistentFlags()

	if v := parsed.Get("workspace"); v != "" && v != "true" {
		pf.Set("workspace", v)
	}
	if v := parsed.Get("w"); v != "" && v != "true" {
		pf.Set("workspace", v)
	}
	if v := parsed.Get("project"); v != "" && v != "true" {
		pf.Set("project", v)
	}
	if v := parsed.Get("p"); v != "" && v != "true" {
		pf.Set("project", v)
	}
	if v := parsed.Get("output"); v != "" && v != "true" {
		pf.Set("output", v)
	}
	if v := parsed.Get("o"); v != "" && v != "true" {
		pf.Set("output", v)
	}
	if v := parsed.Get("api-url"); v != "" && v != "true" {
		pf.Set("api-url", v)
	}
	if v := parsed.Get("api-key"); v != "" && v != "true" {
		pf.Set("api-key", v)
	}
	if v := parsed.Get("verbose"); v == "true" || v == "1" {
		pf.Set("verbose", "true")
	}
	if v := parsed.Get("quiet"); v == "true" || v == "1" {
		pf.Set("quiet", "true")
	}
	if v := parsed.Get("q"); v == "true" || v == "1" {
		pf.Set("quiet", "true")
	}
	if v := parsed.Get("per-page"); v != "" && v != "true" {
		pf.Set("per-page", v)
	}
	if v := parsed.Get("cursor"); v != "" && v != "true" {
		pf.Set("cursor", v)
	}
	if v := parsed.Get("all"); v == "true" || v == "1" {
		pf.Set("all", "true")
	}
	if v := parsed.Get("dry-run"); v == "true" || v == "1" {
		pf.Set("dry-run", "true")
	}
	if v := parsed.Get("n"); v == "true" || v == "1" {
		pf.Set("dry-run", "true")
	}
	if v := parsed.Get("strict"); v == "true" || v == "1" {
		pf.Set("strict", "true")
	}
	if v := parsed.Get("no-resolve"); v == "true" || v == "1" {
		pf.Set("no-resolve", "true")
	}
	if v := parsed.Get("field"); v != "" && v != "true" {
		pf.Set("field", v)
	}
	if v := parsed.Get("fields"); v != "" && v != "true" {
		pf.Set("fields", v)
	}
	if v := parsed.Get("id-only"); v == "true" || v == "1" {
		pf.Set("id-only", "true")
	}
}

// WaitForPendingUpdates waits up to the given duration for background spec refreshes.
func WaitForPendingUpdates(timeout <-chan struct{}) {
	done := make(chan struct{})
	go func() {
		PendingUpdates.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-timeout:
	}
}

// topicHasExecutableEntries returns true if at least one entry has an API reference URL
// and is not an overview.
func TopicHasExecutableEntries(topic *docs.Topic) bool {
	for _, e := range topic.Entries {
		if !IsAPIReferenceURL(e.URL) {
			continue
		}
		name := DeriveSubcommandName(e.Title, topic.Name)
		if name != "" {
			return true
		}
	}
	return false
}

// OverriddenTopics lists topics that have static commands and should not be
// dynamically generated. Keep project here since we retain its typed API.
var OverriddenTopics = map[string]bool{
	"introduction": true,
	"user":         true,
}

// FilteredTopicName returns true if a topic should get dynamic commands.
func FilteredTopicName(name string) bool {
	return !OverriddenTopics[strings.ToLower(name)]
}
