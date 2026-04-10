package skills

import "embed"

//go:embed freelo/SKILL.md
var skillFS embed.FS

// SkillContent returns the embedded SKILL.md content.
func SkillContent() ([]byte, error) {
	return skillFS.ReadFile("freelo/SKILL.md")
}
