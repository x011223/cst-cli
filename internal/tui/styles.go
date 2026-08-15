// Package tui contains all bubbletea-based interactive interfaces for cst-cli.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	accent = lipgloss.Color("205")
	green  = lipgloss.Color("46")
	red    = lipgloss.Color("196")
	yellow = lipgloss.Color("214")
	cyan   = lipgloss.Color("51")

	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(accent).Render
	helpStyle   = lipgloss.NewStyle().Faint(true).Render
	cursorStyle = lipgloss.NewStyle().Foreground(accent).Bold(true).Render
	checkStyle  = lipgloss.NewStyle().Foreground(green).Render
	dimStyle    = lipgloss.NewStyle().Faint(true).Render
	projStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render

	successStyle = lipgloss.NewStyle().Foreground(green).Bold(true).Render
	failStyle    = lipgloss.NewStyle().Foreground(red).Bold(true).Render
	runningStyle = lipgloss.NewStyle().Foreground(yellow).Render

	errorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(red).
			Padding(0, 1).
			Foreground(lipgloss.Color("252"))
)
