package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mggarofalo/plane-cli/internal/auth"
	"github.com/mggarofalo/plane-cli/internal/skillgen"
	"github.com/spf13/cobra"
)

var flagSkillOutput string

func init() {
	rootCmd.AddCommand(skillsCmd)
	skillsCmd.AddCommand(skillsGenerateCmd)

	skillsGenerateCmd.Flags().StringVar(&flagSkillOutput, "output", "", "Output directory (default: ~/.claude/skills/plane)")
}

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage Claude Code skills",
}

var skillsGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a Claude Code skill from cached API specs",
	Long: `Generate a Claude Code skill consisting of SKILL.md and references/resources.md.

The skill documents all plane-cli subcommands with parameter tables, name
resolution annotations, and agent-oriented usage guidance. It is intended
for use by Claude Code agents, not as a user-facing skill.

Requires cached API specs (run 'plane docs update-specs' first for full detail).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := flagSkillOutput
		if outputDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("determining home directory: %w", err)
			}
			outputDir = filepath.Join(home, ".claude", "skills", "plane")
		}

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}

		profile := ""
		if cfg, err := auth.LoadConfig(); err == nil {
			profile = cfg.ActiveProfile
		}

		if err := skillgen.Generate(profile, outputDir); err != nil {
			return fmt.Errorf("generating skill: %w", err)
		}

		if !flagQuiet {
			fmt.Fprintf(os.Stderr, "Generated skill in %s\n", outputDir)
		}
		return nil
	},
}
