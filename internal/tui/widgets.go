package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBar renders a fixed-width colored progress bar using block glyphs.
func ProgressBar(done, total, width int) string {
	if total <= 0 {
		return "[" + dimStyle(strings.Repeat("·", width)) + "]"
	}
	if done > total {
		done = total
	}
	filled := width * done / total
	donePart := lipgloss.NewStyle().Foreground(cyan).Render(strings.Repeat("█", filled))
	restPart := lipgloss.NewStyle().Faint(true).Render(strings.Repeat("░", width-filled))
	return "[" + donePart + restPart + "]"
}

// CheckBox renders a checkbox glyph reflecting the selected state.
func CheckBox(selected bool) string {
	if selected {
		return checkStyle("[x]")
	}
	return dimStyle("[ ]")
}
