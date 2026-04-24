package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/freeloio/freelo-cli/skills"
	"github.com/spf13/cobra"
)

// NewSkillCmd creates the 'skill' command for agent skill management.
func NewSkillCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Show or install the agent skill",
	}

	cmd.AddCommand(
		newSkillShowCmd(app),
		newSkillInstallCmd(app),
	)

	return cmd
}

func newSkillShowCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the embedded SKILL.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := skills.SkillContent()
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
}

func newSkillInstallCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [target]",
		Short: "Install skill file for an AI agent",
		Long: `Install the Freelo skill file for AI agent integration.

Targets:
  claude   - Install to ~/.claude/skills/freelo/ (Claude Code)
  codex    - Install to ~/.codex/skills/freelo/ (OpenAI Codex)
  opencode - Install to ~/.config/opencode/skill/freelo/ (OpenCode)
  all      - Install to all known locations`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			target := "claude"
			if len(args) > 0 {
				target = args[0]
			}

			data, err := skills.SkillContent()
			if err != nil {
				return err
			}

			home, _ := os.UserHomeDir()

			targets := map[string]string{
				"claude":   filepath.Join(home, ".claude", "skills", "freelo", "SKILL.md"),
				"codex":    filepath.Join(home, ".codex", "skills", "freelo", "SKILL.md"),
				"opencode": filepath.Join(home, ".config", "opencode", "skill", "freelo", "SKILL.md"),
			}

			install := func(name, path string) error {
				dir := filepath.Dir(path)
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("failed to create %s: %w", dir, err)
				}
				if err := os.WriteFile(path, data, 0644); err != nil {
					return fmt.Errorf("failed to write %s: %w", path, err)
				}
				fmt.Printf("Installed skill to %s\n", path)
				return nil
			}

			if target == "all" {
				for name, path := range targets {
					if err := install(name, path); err != nil {
						return err
					}
				}
				out.OK(map[string]any{"installed": "all"}, "Skill installed to all targets", nil)
				return nil
			}

			path, ok := targets[target]
			if !ok {
				return fmt.Errorf("unknown target '%s' — use claude, codex, opencode, or all", target)
			}

			if err := install(target, path); err != nil {
				return err
			}

			out.OK(map[string]any{"installed": target, "path": path}, fmt.Sprintf("Skill installed for %s", target), nil)
			return nil
		},
	}
	return cmd
}
