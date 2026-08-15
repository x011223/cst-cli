package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wujunqiang/cst-cli/internal/maven"
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
	phase       maven.Phase
	completed   int // number of successfully finished phases (0..3)
	running     bool
	done        bool
	failed      bool
	failedPhase maven.Phase
}

type model struct {
	state    state
	projects []maven.Project
	cursor   int
	selected map[int]bool
	statuses []projectStatus
	doneCnt  int
	results  []maven.BuildResult
	ch       chan tea.Msg
	err      error
}

func initialModel() model {
	return model{state: stateSelect, selected: map[int]bool{}}
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
		var chosen []maven.Project
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

func (m model) startBuilds(chosen []maven.Project) (tea.Model, tea.Cmd) {
	m.state = stateRunning
	m.statuses = make([]projectStatus, len(chosen))
	for i, p := range chosen {
		m.statuses[i] = projectStatus{name: p.Name}
	}
	m.ch = make(chan tea.Msg, 64)
	go func() {
		results := maven.RunBuilds(chosen,
			func(idx int, phase maven.Phase, running bool, out string, failed bool) {
				m.ch <- phaseMsg{idx: idx, phase: phase, running: running, failed: failed}
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
		b.WriteString(cursor + CheckBox(m.selected[i]) + " " + projStyle(p.Name) + "\n")
	}

	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	b.WriteString("\n" + helpStyle("↑/↓ or k/j move   space toggle   a all   n none   enter run   q quit") + "\n")
	b.WriteString("\n" + cursorStyle("▶") + " " + checkStyle(fmt.Sprintf("%d project(s) selected — press enter to build", count)) + "\n")
	return b.String()
}

func (m model) viewRunning() string {
	total := len(m.statuses)
	var b strings.Builder
	b.WriteString(titleStyle("Building") + "  " + ProgressBar(m.doneCnt, total, 24) + "\n")
	b.WriteString(dimStyle(fmt.Sprintf("Overall: %d/%d projects complete", m.doneCnt, total)) + "\n\n")
	for _, st := range m.statuses {
		b.WriteString(fmt.Sprintf(" %s %-26s %s %s\n", iconFor(st), projStyle(st.name), ProgressBar(st.completed, len(maven.Phases), 10), labelFor(st)))
	}
	b.WriteString("\n" + helpStyle("Press q to quit.") + "\n")
	return b.String()
}

func (m model) viewDone() string {
	if m.err != nil {
		return failStyle("Error: ") + m.err.Error() + "\n\n" + helpStyle("Press q to quit.") + "\n"
	}
	var b strings.Builder
	b.WriteString(titleStyle("Build complete") + "\n\n")
	for _, r := range m.results {
		if r.FailedAt == "" {
			b.WriteString(successStyle("✓ "+r.Project.Name) + dimStyle("  clean, compile, package all succeeded") + "\n")
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

type phaseMsg struct {
	idx     int
	phase   maven.Phase
	running bool
	failed  bool
}

type doneMsg struct{ idx int }
type errMsg struct{ err error }

// RunMvnBuild launches the interactive Maven build subcommand.
func RunMvnBuild() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	projects, err := maven.FindMavenProjects(dir)
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
