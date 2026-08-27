package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wujunqiang/cst-cli/internal/deploy"
)

type dstate int

const (
	dStateSelect dstate = iota
	dStateConfirm
	dStateRunning
	dStateDone
)

// dServiceStatus tracks live deployment status of one service.
type dServiceStatus struct {
	svc        deploy.Service
	stepIdx    int
	stepName   string
	running    bool
	done       bool
	failed     bool
	failedStep string
	output     string
}

type dmodel struct {
	state       dstate
	cfg         *deploy.Config
	cursor      int
	selected    map[int]bool
	confirmList []deploy.Service
	statuses    []dServiceStatus
	results     []deploy.ServiceResult
	doneCnt     int
	ch          chan tea.Msg
	err         error
}

func initialDModel(cfg *deploy.Config) dmodel {
	return dmodel{state: dStateSelect, cfg: cfg, selected: map[int]bool{}}
}

func (m dmodel) Init() tea.Cmd { return nil }

func (m dmodel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case dStateSelect:
			return m.updateSelect(msg)
		case dStateConfirm:
			switch msg.String() {
			case "q", "ctrl+c", "esc":
				return m, tea.Quit
			case "enter":
				return m.startDeploy()
			case "n":
				m.state = dStateSelect
			}
		case dStateDone:
			if msg.String() == "q" || msg.String() == "ctrl+c" || msg.String() == "esc" {
				return m, tea.Quit
			}
		}
	case dstepMsg:
		st := &m.statuses[msg.svcIdx]
		st.stepIdx = msg.stepIdx
		st.stepName = msg.name
		st.running = msg.running
		if !msg.running {
			st.output = msg.out
			if msg.failed {
				st.failed = true
				st.failedStep = msg.name
			}
		}
		return m, m.listen()
	case dServiceDoneMsg:
		m.statuses[msg.svcIdx].done = true
		m.doneCnt++
		if m.doneCnt == len(m.statuses) {
			m.state = dStateDone
			return m, nil
		}
		return m, m.listen()
	case errMsg:
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m dmodel) updateSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.cfg.Services)-1 {
			m.cursor++
		}
	case " ":
		m.selected[m.cursor] = !m.selected[m.cursor]
	case "a":
		for i := range m.cfg.Services {
			m.selected[i] = true
		}
	case "n":
		for i := range m.cfg.Services {
			m.selected[i] = false
		}
	case "enter":
		var chosen []deploy.Service
		for i, s := range m.cfg.Services {
			if m.selected[i] {
				chosen = append(chosen, s)
			}
		}
		if len(chosen) == 0 {
			return m, nil
		}
		return m.showConfirm(chosen)
	}
	return m, nil
}

func (m dmodel) showConfirm(chosen []deploy.Service) (tea.Model, tea.Cmd) {
	m.state = dStateConfirm
	m.confirmList = chosen
	m.statuses = make([]dServiceStatus, len(chosen))
	for i, s := range chosen {
		m.statuses[i] = dServiceStatus{svc: s}
	}
	return m, nil
}

func (m dmodel) startDeploy() (tea.Model, tea.Cmd) {
	m.state = dStateRunning
	m.statuses = make([]dServiceStatus, len(m.confirmList))
	for i, s := range m.confirmList {
		m.statuses[i] = dServiceStatus{svc: s}
	}
	m.results = make([]deploy.ServiceResult, len(m.confirmList))
	m.ch = make(chan tea.Msg, 64)
	chosen := m.confirmList
	cfg := m.cfg
	go func() {
		for i, s := range chosen {
			res := cfg.Deploy(s, func(stepIdx int, name, command string, running, failed bool, out string) {
				m.ch <- dstepMsg{svcIdx: i, stepIdx: stepIdx, name: name, running: running, failed: failed, out: out}
			})
			m.results[i] = res
			m.ch <- dServiceDoneMsg{svcIdx: i}
		}
	}()
	return m, m.listen()
}

func (m dmodel) listen() tea.Cmd {
	return func() tea.Msg {
		return <-m.ch
	}
}

func (m dmodel) View() string {
	switch m.state {
	case dStateSelect:
		return m.dViewSelect()
	case dStateConfirm:
		return m.dViewConfirm()
	case dStateRunning:
		return m.dViewRunning()
	case dStateDone:
		return m.dViewDone()
	}
	return ""
}

func (m dmodel) dViewSelect() string {
	var b strings.Builder
	b.WriteString(titleStyle("Select services to deploy") + "\n")
	b.WriteString(dimStyle(fmt.Sprintf("jar dir: %s   ·   tmp dir: %s", m.cfg.JarDir, m.cfg.TmpDir)) + "\n\n")
	for i, s := range m.cfg.Services {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle("❯ ")
		}
		check := CheckBox(m.selected[i])
		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, projStyle(s.Name)))
		b.WriteString(dimStyle(fmt.Sprintf("      jar: %s   container: %s\n", s.Jar, s.Container)))
	}
	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	b.WriteString("\n" + helpStyle("↑/↓ move   space toggle   a all   n none   enter next   q quit") + "\n")
	b.WriteString("\n" + cursorStyle("▶") + " " + checkStyle(fmt.Sprintf("%d service(s) selected", count)) + "\n")
	return b.String()
}

func (m dmodel) dViewConfirm() string {
	var b strings.Builder
	b.WriteString(titleStyle("Confirm deployment") + "\n\n")
	for _, s := range m.confirmList {
		b.WriteString(projStyle("● "+s.Name) + "\n")
		b.WriteString(dimStyle(fmt.Sprintf("  mv %s/%s -> .bak\n", m.cfg.JarDir, s.Jar)))
		b.WriteString(dimStyle(fmt.Sprintf("  mv %s/%s -> %s/\n", m.cfg.TmpDir, s.Jar, m.cfg.JarDir)))
		b.WriteString(dimStyle(fmt.Sprintf("  docker restart %s\n\n", s.Container)))
	}
	b.WriteString(helpStyle("enter deploy   n back   q quit") + "\n")
	return b.String()
}

func (m dmodel) dViewRunning() string {
	var b strings.Builder
	b.WriteString(titleStyle("Deploying") + "  " + ProgressBar(m.doneCnt, len(m.statuses), 24) + "\n\n")
	for _, st := range m.statuses {
		icon := " "
		switch {
		case st.failed:
			icon = failStyle("✗")
		case st.done:
			icon = successStyle("✓")
		case st.running:
			icon = runningStyle("●")
		}
		label := "pending"
		switch {
		case st.failed:
			label = failStyle(st.failedStep)
		case st.done:
			label = successStyle("done")
		case st.running:
			label = runningStyle(st.stepName)
		}
		b.WriteString(fmt.Sprintf(" %s %-30s %s\n", icon, projStyle(st.svc.Name), label))
	}
	b.WriteString("\n" + helpStyle("Press q to quit.") + "\n")
	return b.String()
}

func (m dmodel) dViewDone() string {
	if m.err != nil {
		return failStyle("Error: ") + m.err.Error() + "\n\n" + helpStyle("Press q to quit.") + "\n"
	}
	ok, fail := 0, 0
	for _, r := range m.results {
		if r.FailedStep == "" {
			ok++
		} else {
			fail++
		}
	}
	var b strings.Builder
	b.WriteString(titleStyle("Deployment complete") + "  ")
	if fail == 0 {
		b.WriteString(successStyle(fmt.Sprintf("%d/%d succeeded", ok, len(m.results))))
	} else {
		b.WriteString(failStyle(fmt.Sprintf("%d succeeded, %d failed", ok, fail)))
	}
	b.WriteString("\n\n")

	for _, r := range m.results {
		if r.FailedStep == "" {
			b.WriteString(successStyle("✓ "+r.Service.Name) + dimStyle("  all steps succeeded") + "\n")
			continue
		}
		b.WriteString(failStyle("✗ "+r.Service.Name) + dimStyle("   failed step: ") + failStyle(r.FailedStep) + "\n")
		for _, step := range r.Steps {
			if step.Err != nil {
				b.WriteString(errorBoxStyle.Render(fmt.Sprintf("%s\n%s", step.Command, step.Output)) + "\n\n")
			}
		}
	}
	b.WriteString(helpStyle("Press q to quit.") + "\n")
	return b.String()
}

type dstepMsg struct {
	svcIdx  int
	stepIdx int
	name    string
	running bool
	failed  bool
	out     string
}

type dServiceDoneMsg struct{ svcIdx int }

// RunDeploy launches the interactive deploy subcommand.
func RunDeploy(configPath string) error {
	cfg, err := deploy.LoadConfig(configPath)
	if err != nil {
		return err
	}
	p := tea.NewProgram(initialDModel(cfg), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
