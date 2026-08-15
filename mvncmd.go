package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateSelect state = iota
	stateRunning
	stateDone
)

// projectStatus tracks the live build status of one project.
type projectStatus struct {
	name        string
	phase       Phase
	completed   int // number of successfully finished phases (0..3)
	running     bool
	done        bool
	failed      bool
	failedPhase Phase
	out         string
}

type model struct {
	state    state
	projects []Project
	cursor   int
	selected map[int]bool
	statuses []projectStatus
	doneCnt  int
	results  []BuildResult
	ch       chan tea.Msg
	err      error
}

func initialModel() model {
	return model{
		state:    stateSelect,
		selected: map[int]bool{},
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateSelect:
			return m.updateSelect(msg)
		case stateDone:
			if msg.String() == "q" || msg.String() == "ctrl+c" || msg.String() == "esc" {
				return m, tea.Quit
			}
		}
	case phaseMsg:
		st := &m.statuses[msg.idx]
		st.phase = msg.phase
		st.running = msg.running
		if !msg.running && msg.out != "" {
			st.out = msg.out
		}
		if !msg.running && msg.failed {
			st.failed = true
			st.failedPhase = msg.phase
		} else if !msg.running && !msg.failed {
			st.completed++
		}
		return m, m.listen()
	case doneMsg:
		m.statuses[msg.idx].done = true
		m.doneCnt++
		if m.doneCnt == len(m.statuses) {
			m.state = stateDone
			return m, nil
		}
		return m, m.listen()
	case errMsg:
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m model) updateSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.projects)-1 {
			m.cursor++
		}
	case " ":
		if len(m.projects) > 0 {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}
	case "a":
		for i := range m.projects {
			m.selected[i] = true
		}
	case "n":
		for i := range m.projects {
			m.selected[i] = false
		}
	case "enter":
		var chosen []Project
		for i, p := range m.projects {
			if m.selected[i] {
				chosen = append(chosen, p)
			}
		}
		if len(chosen) == 0 {
			return m, nil
		}
		return m.startBuilds(chosen)
	}
	return m, nil
}

func (m model) startBuilds(chosen []Project) (tea.Model, tea.Cmd) {
	m.state = stateRunning
	m.statuses = make([]projectStatus, len(chosen))
	for i, p := range chosen {
		m.statuses[i] = projectStatus{name: p.Name}
	}
	m.ch = make(chan tea.Msg, 64)
	go func() {
		results := RunBuilds(chosen,
			func(idx int, phase Phase, running bool, out string, failed bool) {
				m.ch <- phaseMsg{idx: idx, phase: phase, running: running, out: out, failed: failed}
			},
			func(idx int) {
				m.ch <- doneMsg{idx: idx}
			})
		m.results = results
	}()
	return m, m.listen()
}

// listen returns a command that blocks until the next build message arrives.
func (m model) listen() tea.Cmd {
	return func() tea.Msg {
		return <-m.ch
	}
}

func (m model) View() string {
	switch m.state {
	case stateSelect:
		return m.viewSelect()
	case stateRunning:
		return m.viewRunning()
	case stateDone:
		return m.viewDone()
	}
	return ""
}

func (m model) viewSelect() string {
	var b strings.Builder
	b.WriteString(titleStyle("Select Maven projects to build") + "\n")
	b.WriteString(dimStyle("clean → compile → package  ·  parallel across projects, serial per project") + "\n\n")

	if len(m.projects) == 0 {
		b.WriteString(dimStyle("No Maven projects (folders with pom.xml) found in the current directory.") + "\n\n")
		b.WriteString(helpStyle("Press q to quit.") + "\n")
		return b.String()
	}

	for i, p := range m.projects {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle("❯ ")
		}
		if m.selected[i] {
			b.WriteString(cursor + checkStyle("[x] ") + projStyle(p.Name) + "\n")
		} else {
			b.WriteString(cursor + dimStyle("[ ] ") + projStyle(p.Name) + "\n")
		}
	}

	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	b.WriteString("\n" + helpStyle("↑/↓ or k/j move   space toggle   a all   n none   enter run   q quit") + "\n")
	b.WriteString(fmt.Sprintf("\n%s %s\n", cursorStyle("▶"), checkStyle(fmt.Sprintf("%d project(s) selected — press enter to build", count))))
	return b.String()
}

func (m model) viewRunning() string {
	total := len(m.statuses)
	var b strings.Builder
	b.WriteString(titleStyle("Building") + "  " + progressBar(m.doneCnt, total, 24) + "\n")
	b.WriteString(dimStyle(fmt.Sprintf("Overall: %d/%d projects complete", m.doneCnt, total)) + "\n\n")
	for _, st := range m.statuses {
		bar := progressBar(st.completed, len(Phases), 10)
		b.WriteString(fmt.Sprintf(" %s %-26s %s %s\n", iconFor(st), projStyle(st.name), bar, labelFor(st)))
	}
	b.WriteString("\n" + helpStyle("Press q to quit.") + "\n")
	return b.String()
}

// progressBar renders a fixed-width colored progress bar using block glyphs.
func progressBar(done, total, width int) string {
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

func (m model) viewDone() string {
	if m.err != nil {
		return failStyle("Error: ") + m.err.Error() + "\n\n" + helpStyle("Press q to quit.") + "\n"
	}
	var b strings.Builder
	b.WriteString(titleStyle("Build complete") + "\n\n")
	for _, r := range m.results {
		if r.FailedAt == "" {
			b.WriteString(successStyle("✓ "+r.Project.Name) +
				dimStyle("  clean, compile, package all succeeded") + "\n")
			continue
		}
		b.WriteString(failStyle(fmt.Sprintf("✗ %s  failed at %s", r.Project.Name, r.FailedAt)) + "\n")
		for _, pr := range r.Results {
			if pr.Err == nil {
				continue
			}
			body := fmt.Sprintf("mvn %s output:\n%s", pr.Phase, pr.Out)
			b.WriteString(errorBoxStyle.Render(body) + "\n\n")
		}
	}
	b.WriteString(helpStyle("Press q to quit.") + "\n")
	return b.String()
}

type phaseMsg struct {
	idx     int
	phase   Phase
	running bool
	out     string
	failed  bool
}

type doneMsg struct{ idx int }
type errMsg struct{ err error }

// Styles used to render the TUI.
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

func iconFor(st projectStatus) string {
	switch {
	case st.failed:
		return failStyle("✗")
	case st.done:
		return successStyle("✓")
	case st.running:
		return runningStyle("●")
	default:
		return dimStyle(" ")
	}
}

func labelFor(st projectStatus) string {
	switch {
	case st.failed:
		return failStyle(fmt.Sprintf("failed at %s", st.failedPhase))
	case st.done:
		return successStyle("done")
	case st.running:
		return runningStyle(fmt.Sprintf("running %s", st.phase))
	default:
		return dimStyle("pending")
	}
}

// RunMvnBuild launches the interactive Maven build subcommand.
func RunMvnBuild() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	projects, err := FindMavenProjects(dir)
	if err != nil {
		return err
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })

	m := initialModel()
	m.projects = projects

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
