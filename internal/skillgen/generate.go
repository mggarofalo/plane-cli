package skillgen

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/mggarofalo/plane-cli/internal/cmdgen"
	"github.com/mggarofalo/plane-cli/internal/docs"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// SkillData is the top-level template context.
type SkillData struct {
	Topics    []TopicData
	Total     int // total action count across all topics
	Generated string
}

// TopicData describes a single resource topic.
type TopicData struct {
	Name        string // CLI topic name (e.g. "issue")
	DisplayName string // Title-cased (e.g. "Issue")
	Actions     []ActionData
	HasEnsure   bool
}

// ActionData describes a single CLI subcommand / MCP tool.
type ActionData struct {
	CLIName  string // e.g. "create"
	MCPName  string // e.g. "issue_create"
	Method   string // HTTP method
	Path     string // path template
	Params   []ParamData
	HasSpec  bool
}

// ParamData describes an endpoint parameter.
type ParamData struct {
	FlagName   string
	Type       string
	Required   bool
	Resolvable bool
	IssueRef   bool
	Enum       []string
}

// hiddenParams mirrors mcpserver/schema.go — params handled by server context.
var hiddenParams = map[string]bool{
	"workspace_slug": true,
	"project_id":     true,
}

// Generate produces SKILL.md and references/resources.md in outputDir.
func Generate(profile, outputDir string) error {
	data, err := buildSkillData(profile)
	if err != nil {
		return err
	}

	funcMap := template.FuncMap{
		"repeat": strings.Repeat,
		"join":   strings.Join,
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}

	// Write SKILL.md
	skillPath := filepath.Join(outputDir, "SKILL.md")
	if err := executeTemplate(tmpl, "skill.md.tmpl", skillPath, data); err != nil {
		return fmt.Errorf("writing SKILL.md: %w", err)
	}

	// Write references/resources.md
	refDir := filepath.Join(outputDir, "references")
	if err := os.MkdirAll(refDir, 0755); err != nil {
		return fmt.Errorf("creating references dir: %w", err)
	}
	resPath := filepath.Join(refDir, "resources.md")
	if err := executeTemplate(tmpl, "resources.md.tmpl", resPath, data); err != nil {
		return fmt.Errorf("writing resources.md: %w", err)
	}

	return nil
}

func executeTemplate(tmpl *template.Template, name, path string, data *SkillData) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	if err := tmpl.ExecuteTemplate(f, name, data); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}

func buildSkillData(profile string) (*SkillData, error) {
	var topics []TopicData
	total := 0

	for _, topic := range docs.DefaultTopics {
		if !cmdgen.FilteredTopicName(topic.Name) {
			continue
		}
		if !cmdgen.TopicHasExecutableEntries(&topic) {
			continue
		}

		td := buildTopicData(profile, &topic)
		total += len(td.Actions)
		topics = append(topics, td)
	}

	return &SkillData{
		Topics:    topics,
		Total:     total,
		Generated: time.Now().UTC().Format("2006-01-02"),
	}, nil
}

func buildTopicData(profile string, topic *docs.Topic) TopicData {
	// Load cached specs for param detail
	cachedSpecs, _ := docs.LoadTopicSpecs(profile, topic.Name)
	specByTitle := make(map[string]*docs.CachedSpec)
	for i := range cachedSpecs {
		specByTitle[cachedSpecs[i].Spec.EntryTitle] = &cachedSpecs[i]
	}

	td := TopicData{
		Name:        topic.Name,
		DisplayName: strings.ToUpper(topic.Name[:1]) + topic.Name[1:],
		HasEnsure:   cmdgen.TopicSupportsEnsure(topic.Name),
	}

	for _, entry := range topic.Entries {
		if !cmdgen.IsAPIReferenceURL(entry.URL) {
			continue
		}
		cliName := cmdgen.DeriveSubcommandName(entry.Title, topic.Name)
		if cliName == "" {
			continue
		}

		mcpName := topic.Name + "_" + strings.ReplaceAll(cliName, "-", "_")

		ad := ActionData{
			CLIName: cliName,
			MCPName: mcpName,
		}

		if cached, ok := specByTitle[entry.Title]; ok {
			ad.HasSpec = true
			ad.Method = cached.Spec.Method
			ad.Path = cached.Spec.PathTemplate
			ad.Params = filterAndSortParams(cached.Spec.Params)
		}

		td.Actions = append(td.Actions, ad)
	}

	return td
}

// filterAndSortParams excludes hidden params, converts HTML params, and sorts
// required-first then alphabetical.
func filterAndSortParams(params []docs.ParamSpec) []ParamData {
	var result []ParamData
	seen := make(map[string]bool)

	for _, p := range params {
		if hiddenParams[p.Name] {
			continue
		}

		flagName := cmdgen.ParamToFlagName(p.Name)
		if cmdgen.IsHTMLParam(p.Name) {
			flagName = cmdgen.MarkdownFlagName(p.Name)
		}

		// Skip duplicates (can happen with HTML param conversion)
		if seen[flagName] {
			continue
		}
		seen[flagName] = true

		pd := ParamData{
			FlagName:   flagName,
			Type:       normalizeType(p.Type),
			Required:   p.Required,
			Resolvable: cmdgen.IsResolvableParam(p.Name),
			IssueRef:   cmdgen.IsIssueRefParam(p.Name),
			Enum:       p.Enum,
		}
		result = append(result, pd)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Required != result[j].Required {
			return result[i].Required
		}
		return result[i].FlagName < result[j].FlagName
	})

	return result
}

func normalizeType(t string) string {
	switch t {
	case "string[]":
		return "string[]"
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	default:
		return "string"
	}
}
