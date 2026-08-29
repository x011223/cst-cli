package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/crypto/ssh"

	"github.com/wujunqiang/cst-cli/internal/docker"
	"github.com/wujunqiang/cst-cli/internal/upload"
)

type dkstate int

const (
	dkStateEnv dkstate = iota
	dkStateLoading
	dkStateList
	dkStateRestarting
)

type dkMark struct {
	ok     bool
	failed bool
	err    string
}

type dkItem struct {
	c      docker.Container
	ok     bool
	failed bool
	err    string
}

type dkmodel struct {
	state   dkstate
	cfg     *upload.Config
	envIdx  int
	env     upload.Environment
	client  *ssh.Client
	cancel  context.CancelFunc
	timeout time.Duration

	items    []dkItem
	cursor   int
	selected map[int]bool
	marks    map[string]dkMark
	loading  bool
	listErr  error

	group     []docker.Container
	groupStat []int // 0 running, 1 ok, 2 fail
	groupLogs [][]string
	doneCnt   int

	width  int
	height int
	ch     chan tea.Msg
	err    error
}

func initialDkModel(cfg *upload.Config, envName string, timeout time.Duration) dkmodel {
	m := dkmodel{
		state:    dkStateEnv,
		cfg:      cfg,
		selected: map[int]bool{},
		marks:    map[string]dkMark{},
		timeout:  timeout,
		width:    80,
		height:   24,
	}
	if timeout <= 0 {
		m.timeout = docker.DefaultLogTimeout
	}
	if envName != "" {
		for i, e := range cfg.Environments {
			if e.Name == envName {
				m.envIdx = i
				m.env = e
				m.state = dkStateLoading
				break
			}
		}
	}
	return m
}

func (m dkmodel) Init() tea.Cmd {
	if m.state == dkStateLoading {
		return m.connect()
	}
	return nil
}

func (m dkmodel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		switch m.state {
		case dkStateEnv:
			return m.updateEnv(msg)
		case dkStateList:
			return m.updateList(msg)
		case dkStateRestarting, dkStateLoading:
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m.quit()
			}
		}
	case dkConnMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = dkStateList
			return m, nil
		}
		m.client = msg.client
		return m.refresh()
	case dkListMsg:
		m.loading = false
		m.listErr = msg.err
		if msg.err == nil {
			items := make([]dkItem, len(msg.list))
			for i, c := range msg.list {
				it := dkItem{c: c}
				if mk, ok := m.marks[c.Name]; ok {
					it.ok = mk.ok
					it.failed = mk.failed
					it.err = mk.err
				}
				items[i] = it
			}
			m.items = items
			if m.cursor >= len(m.items) {
				m.cursor = 0
			}
		}
		// A refresh started from the list must not yank the UI out of a live group restart.
		if m.state != dkStateRestarting {
			m.state = dkStateList
		}
		return m, nil
	case dkBeginMsg:
		if msg.idx >= 0 && msg.idx < len(m.groupStat) {
			m.groupStat[msg.idx] = 0
		}
		return m, m.listen()
	case dkLogMsg:
		idx := msg.idx
		if idx < 0 || idx >= len(m.groupLogs) {
			for i, c := range m.group {
				if c.Name == msg.name {
					idx = i
					break
				}
			}
		}
		if idx >= 0 && idx < len(m.groupLogs) {
			m.groupLogs[idx] = append(m.groupLogs[idx], msg.line)
			if len(m.groupLogs[idx]) > 200 {
				m.groupLogs[idx] = m.groupLogs[idx][len(m.groupLogs[idx])-120:]
			}
		}
		return m, m.listen()
	case dkItemDoneMsg:
		m.doneCnt++
		if msg.idx >= 0 && msg.idx < len(m.group) {
			name := m.group[msg.idx].Name
			mk := m.marks[name]
			if msg.err != nil {
				mk.failed = true
				mk.ok = false
				mk.err = msg.err.Error()
				if msg.idx < len(m.groupStat) {
					m.groupStat[msg.idx] = 2
				}
			} else {
				mk.ok = true
				mk.failed = false
				mk.err = ""
				if msg.idx < len(m.groupStat) {
					m.groupStat[msg.idx] = 1
				}
			}
			m.marks[name] = mk
			for i := range m.items {
				if m.items[i].c.Name == name {
					m.items[i].ok = mk.ok
					m.items[i].failed = mk.failed
					m.items[i].err = mk.err
					break
				}
			}
		}
		return m, m.listen()
	case dkGroupDoneMsg:
		m.groupLogs = nil
		m.selected = map[int]bool{}
		m.state = dkStateList
		return m.refresh()
	case errMsg:
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m dkmodel) updateEnv(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m.quit()
	case "up", "k":
		if m.envIdx > 0 {
			m.envIdx--
		}
	case "down", "j":
		if m.envIdx < len(m.cfg.Environments)-1 {
			m.envIdx++
		}
	case "enter", " ":
		m.env = m.cfg.Environments[m.envIdx]
		m.state = dkStateLoading
		return m, m.connect()
	}
	return m, nil
}

func (m dkmodel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m.quit()
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case " ":
		if len(m.items) > 0 {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}
	case "a":
		for i := range m.items {
			m.selected[i] = true
		}
	case "n":
		m.selected = map[int]bool{}
	case "r":
		return m.refresh()
	case "enter":
		var group []docker.Container
		for i, it := range m.items {
			if m.selected[i] {
				group = append(group, it.c)
			}
		}
		if len(group) == 0 {
			return m, nil
		}
		return m.startGroup(group)
	}
	return m, nil
}

func (m dkmodel) connect() tea.Cmd {
	env := m.env
	return func() tea.Msg {
		client, err := env.Dial()
		return dkConnMsg{client: client, err: err}
	}
}

func (m dkmodel) refresh() (tea.Model, tea.Cmd) {
	if m.client == nil {
		m.state = dkStateLoading
		return m, m.connect()
	}
	m.loading = true
	if m.state != dkStateList {
		m.state = dkStateLoading
	}
	client := m.client
	return m, func() tea.Msg {
		list, err := docker.List(client)
		return dkListMsg{list: list, err: err}
	}
}

func (m dkmodel) startGroup(group []docker.Container) (tea.Model, tea.Cmd) {
	m.state = dkStateRestarting
	m.group = group
	m.groupStat = make([]int, len(group))
	m.groupLogs = make([][]string, len(group))
	m.doneCnt = 0
	m.ch = make(chan tea.Msg, 1024)
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	client := m.client
	timeout := m.timeout
	ch := m.ch
	go func() {
		var wg sync.WaitGroup
		for i, c := range group {
			wg.Add(1)
			go func(i int, c docker.Container) {
				defer wg.Done()
				if ctx.Err() != nil {
					return
				}
				ch <- dkBeginMsg{idx: i}
				name := c.Name
				err := docker.RestartAndFollow(ctx, client, name, timeout, func(line string) {
					select {
					case ch <- dkLogMsg{idx: i, name: name, line: line}:
					default:
					}
				})
				ch <- dkItemDoneMsg{idx: i, err: err}
			}(i, c)
		}
		wg.Wait()
		if ctx.Err() == nil {
			ch <- dkGroupDoneMsg{}
		}
	}()
	return m, m.listen()
}

func (m dkmodel) listen() tea.Cmd {
	return func() tea.Msg {
		return <-m.ch
	}
}

func (m dkmodel) quit() (tea.Model, tea.Cmd) {
	if m.cancel != nil {
		m.cancel()
	}
	if m.client != nil {
		_ = m.client.Close()
		m.client = nil
	}
	return m, tea.Quit
}

func (m dkmodel) View() string {
	if m.err != nil {
		return failStyle("Error: ") + m.err.Error() + "\n\n" + helpStyle("Press q to quit.") + "\n"
	}
	switch m.state {
	case dkStateEnv:
		return m.viewEnv()
	case dkStateLoading:
		return titleStyle("Docker") + "\n\n" + runningStyle("connecting "+m.env.Host+"…") + "\n"
	case dkStateList:
		return m.viewList()
	case dkStateRestarting:
		return m.viewRestarting()
	}
	return ""
}

func (m dkmodel) viewEnv() string {
	var b strings.Builder
	b.WriteString(titleStyle("Select docker host") + "\n\n")
	for i, e := range m.cfg.Environments {
		cursor := "  "
		if i == m.envIdx {
			cursor = cursorStyle("❯ ")
		}
		b.WriteString(fmt.Sprintf("%s%s  %s:%d\n", cursor, projStyle(e.Name), dimStyle(e.Host), e.Port))
	}
	b.WriteString("\n" + helpStyle("↑/↓ move   enter select   q quit") + "\n")
	return b.String()
}

func (m dkmodel) viewList() string {
	var b strings.Builder
	b.WriteString(titleStyle("Docker containers") + "  " + dimStyle(fmt.Sprintf("%s  %s@%s", m.env.Name, m.env.User, m.env.Host)))
	if m.loading {
		b.WriteString("  " + runningStyle("refreshing…"))
	}
	b.WriteString("\n")
	if m.listErr != nil {
		b.WriteString("\n" + failStyle(m.listErr.Error()) + "\n\n" + helpStyle("r refresh   q quit") + "\n")
		return b.String()
	}
	if len(m.items) == 0 {
		b.WriteString("\n" + dimStyle("No containers.") + "\n\n" + helpStyle("r refresh   q quit") + "\n")
		return b.String()
	}
	b.WriteString("\n")
	nameW, statusW := 0, 0
	for _, it := range m.items {
		nameW = max(nameW, len(it.c.Name))
		statusW = max(statusW, len(it.c.Status))
	}
	visible := m.height - 8
	if visible < 5 {
		visible = 5
	}
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := start + visible
	if end > len(m.items) {
		end = len(m.items)
	}
	for i := start; i < end; i++ {
		it := m.items[i]
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle("❯ ")
		}
		state := dimStyle("●")
		switch it.c.State {
		case "running":
			state = successStyle("●")
		case "exited", "dead":
			state = failStyle("●")
		default:
			state = runningStyle("●")
		}
		mark := ""
		if it.ok {
			mark = "  " + successStyle("✓ 已重启")
		} else if it.failed {
			mark = "  " + failStyle("✗ 已重启")
		}
		b.WriteString(cursor + CheckBox(m.selected[i]) + " " + state + " " +
			padRight(projStyle(it.c.Name), nameW) + "  " +
			padRight(dimStyle(it.c.Status), statusW) + mark + "\n")
	}
	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	b.WriteString("\n" + helpStyle("↑/↓ move   space toggle   a all   n none   enter restart group   r refresh   q quit") + "\n")
	b.WriteString(cursorStyle("▶") + " " + checkStyle(fmt.Sprintf("%d selected — enter restarts this group in parallel", count)) + "\n")
	return b.String()
}

func (m dkmodel) viewRestarting() string {
	total := len(m.group)
	var b strings.Builder
	b.WriteString(titleStyle("Restarting group") + "  " + ProgressBar(m.doneCnt, total, 20) +
		"  " + dimStyle(fmt.Sprintf("%d/%d done", m.doneCnt, total)) + "\n")
	b.WriteString(dimStyle("parallel  ·  one log pane per container  ·  waiting for "+docker.SuccessMarker) + "\n")

	avail := m.height - 3
	if avail < 6 {
		avail = 6
	}
	n := total
	if n < 1 {
		n = 1
	}
	paneH := avail / n
	if paneH < 3 {
		paneH = 3
	}
	extra := avail - paneH*n
	if extra < 0 {
		extra = 0
	}
	logW := m.width - 4
	if logW < 16 {
		logW = 16
	}
	for i, c := range m.group {
		h := paneH
		if extra > 0 {
			h++
			extra--
		}
		b.WriteString(m.renderLogPane(i, c, h, logW))
	}
	b.WriteString(helpStyle("q quit") + "\n")
	return b.String()
}

func (m dkmodel) renderLogPane(i int, c docker.Container, height, logW int) string {
	st := 0
	if i < len(m.groupStat) {
		st = m.groupStat[i]
	}
	icon := runningStyle("●")
	label := runningStyle("restarting")
	switch st {
	case 1:
		icon = successStyle("✓")
		label = successStyle("启动成功")
	case 2:
		icon = failStyle("✗")
		label = failStyle("failed")
		if mk, ok := m.marks[c.Name]; ok && mk.err != "" {
			label = failStyle(clipWidth(mk.err, logW))
		}
	}
	var b strings.Builder
	b.WriteString(" " + icon + " " + projStyle(c.Name) + "  " + label + "\n")
	logH := height - 1
	if logH < 1 {
		logH = 1
	}
	var lines []string
	if i < len(m.groupLogs) {
		lines = m.groupLogs[i]
	}
	if len(lines) == 0 {
		b.WriteString(" " + dimStyle("│") + " " + dimStyle("waiting for logs…") + "\n")
		for j := 1; j < logH; j++ {
			b.WriteString(" " + dimStyle("│") + "\n")
		}
		return b.String()
	}
	if len(lines) > logH {
		lines = lines[len(lines)-logH:]
	}
	bar := dimStyle("│")
	for _, line := range lines {
		clipped := clipWidth(line, logW)
		if docker.IsReady(line) {
			b.WriteString(" " + bar + " " + successStyle(clipped) + "\n")
		} else {
			b.WriteString(" " + bar + " " + dimStyle(clipped) + "\n")
		}
	}
	for j := len(lines); j < logH; j++ {
		b.WriteString(" " + bar + "\n")
	}
	return b.String()
}

type dkConnMsg struct {
	client *ssh.Client
	err    error
}

type dkListMsg struct {
	list []docker.Container
	err  error
}

type dkBeginMsg struct{ idx int }

type dkLogMsg struct {
	idx  int
	name string
	line string
}

type dkItemDoneMsg struct {
	idx int
	err error
}

type dkGroupDoneMsg struct{}

// RunDocker launches the lazydocker-style container restart TUI.
func RunDocker(configPath, env string, timeout time.Duration) error {
	cfg, err := upload.LoadConfig(configPath)
	if err != nil {
		return err
	}
	p := tea.NewProgram(initialDkModel(cfg, env, timeout), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
