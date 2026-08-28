package tui

import (
	"fmt"
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

// ProgressBar64 is ProgressBar for byte counters.
func ProgressBar64(done, total int64, width int) string {
	if total <= 0 {
		return ProgressBar(0, 1, width)
	}
	if done > total {
		done = total
	}
	return ProgressBar(int(done*1000/total), 1000, width)
}

// FormatSize renders a byte count as a short human-readable string.
func FormatSize(n int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// CheckBox renders a checkbox glyph reflecting the selected state.
func CheckBox(selected bool) string {
	if selected {
		return checkStyle("[x]")
	}
	return dimStyle("[ ]")
}

// padRight right-pads s to the given visible width, ignoring ANSI sequences.
func padRight(s string, width int) string {
	n := lipgloss.Width(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

func maxVisible(ss ...string) int {
	m := 0
	for _, s := range ss {
		if n := lipgloss.Width(s); n > m {
			m = n
		}
	}
	return m
}

// tableRow joins cells with two spaces and a trailing newline. Newlines must
// not appear inside styled cells (lipgloss would pad the next line).
func tableRow(cells ...string) string {
	return strings.Join(cells, "  ") + "\n"
}
