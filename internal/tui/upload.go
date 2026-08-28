package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wujunqiang/cst-cli/internal/deploy"
	"github.com/wujunqiang/cst-cli/internal/jars"
	"github.com/wujunqiang/cst-cli/internal/maven"
	"github.com/wujunqiang/cst-cli/internal/upload"
)

type ustate int

const (
	uStateEnv ustate = iota
	uStateJars
	uStateRunning
	uStateConfirmRestart
	uStateRestarting
	uStateDone
)

type rstStatus struct {
	svc     deploy.Service
	running bool
	waiting bool
	done    bool
	failed  bool
	output  string
}

type umodel struct {
	state          ustate
	cfg            *upload.Config
	deployCfg      *deploy.Config
	deployCfgErr   error
	envIdx         int
	env            upload.Environment
	pattern        string
	jars           []jars.JarFile
	cursor         int
	selected       map[int]bool
	results        []error
	upStatus       []upFileStatus
	doneCnt        int
	toRestart      []deploy.Service
	unmatched      []string
	rstStatuses    []rstStatus
	rstResults     []error
	rstDoneCnt     int
	restartSkipped bool
	ch             chan tea.Msg
	err            error
}

func initialUModel(cfg *upload.Config, deployCfg *deploy.Config, deployCfgErr error, envName string, jarList []jars.JarFile, pattern string) umodel {
	m := umodel{
		state:        uStateEnv,
		cfg:          cfg,
		deployCfg:    deployCfg,
		deployCfgErr: deployCfgErr,
		jars:         jarList,
		pattern:      pattern,
		selected:     map[int]bool{},
	}
	if envName != "" {
		for i, e := range cfg.Environments {
			if e.Name == envName {
				m.envIdx = i
				m.env = e
				m.state = uStateJars
				break
			}
		}
	}
	return m
}

func (m umodel) Init() tea.Cmd { return nil }

func (m umodel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case uStateEnv:
			return m.updateEnv(msg)
		case uStateJars:
			return m.updateJars(msg)
		case uStateConfirmRestart:
			return m.updateConfirmRestart(msg)
		case uStateDone:
			if msg.String() == "q" || msg.String() == "ctrl+c" || msg.String() == "esc" {
				return m, tea.Quit
			}
		}
	case upMsg:
		st := &m.upStatus[msg.idx]
		st.written = msg.written
		if msg.total > 0 {
			st.total = msg.total
		}
		if msg.running {
			st.running = true
			return m, m.listen()
		}
		st.running = false
		st.done = true
		st.failed = msg.err != nil
		m.results[msg.idx] = msg.err
		m.doneCnt++
		if m.doneCnt == len(m.results) {
			return m.afterUpload()
		}
		return m, m.listen()
	case rstMsg:
		st := &m.rstStatuses[msg.idx]
		if msg.done {
			st.done = true
			st.running = false
			st.waiting = false
			st.failed = msg.failed
			if msg.out != "" {
				st.output = msg.out
			}
			m.rstResults[msg.idx] = nilPtr(msg.failed, msg.out)
			m.rstDoneCnt++
			if m.rstDoneCnt == len(m.rstStatuses) {
				m.state = uStateDone
				return m, nil
			}
			return m, m.listen()
		}
		st.running = msg.running
		st.waiting = msg.waiting
		if !msg.running && !msg.waiting {
			st.output = msg.out
			if msg.failed {
				st.failed = true
			}
		}
		return m, m.listen()
	case errMsg:
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m umodel) updateEnv(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
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
		m.state = uStateJars
	}
	return m, nil
}

func (m umodel) updateJars(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.jars)-1 {
			m.cursor++
		}
	case " ":
		if len(m.jars) > 0 {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}
	case "a":
		for i := range m.jars {
			m.selected[i] = true
		}
	case "n":
		for i := range m.jars {
			m.selected[i] = false
		}
	case "enter":
		var chosen []jars.JarFile
		for i, j := range m.jars {
			if m.selected[i] {
				chosen = append(chosen, j)
			}
		}
		if len(chosen) == 0 {
			return m, nil
		}
		return m.startUpload(chosen)
	}
	return m, nil
}

func (m umodel) updateConfirmRestart(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "n":
		m.restartSkipped = true
		m.state = uStateDone
	case "enter":
		return m.startRestart()
	}
	return m, nil
}

func (m umodel) startUpload(chosen []jars.JarFile) (tea.Model, tea.Cmd) {
	m.state = uStateRunning
	m.jars = chosen
	m.results = make([]error, len(chosen))
	m.upStatus = make([]upFileStatus, len(chosen))
	for i, j := range chosen {
		st := upFileStatus{running: true}
		if info, err := os.Stat(j.Path); err == nil {
			st.total = info.Size()
		}
		m.upStatus[i] = st
	}
	if len(chosen) == 0 {
		m.state = uStateDone
		return m, nil
	}
	m.ch = make(chan tea.Msg, 256)
	env := m.env
	go func() {
		upload.UploadAll(env, chosen, func(idx int, name, dest string, written, total int64, running, failed bool, out string) {
			msg := upMsg{idx: idx, written: written, total: total, running: running, err: nilPtr(failed, out)}
			if running {
				select {
				case m.ch <- msg:
				default:
				}
				return
			}
			m.ch <- msg
		})
	}()
	return m, m.listen()
}

func (m umodel) afterUpload() (tea.Model, tea.Cmd) {
	var names []string
	for i, err := range m.results {
		if err == nil {
			names = append(names, m.jars[i].Name)
		}
	}
	if m.deployCfg == nil || len(names) == 0 {
		m.state = uStateDone
		return m, nil
	}
	matched, unmatched := m.deployCfg.MatchServices(names)
	m.toRestart = matched
	m.unmatched = unmatched
	if len(matched) == 0 {
		m.state = uStateDone
		return m, nil
	}
	m.state = uStateConfirmRestart
	return m, nil
}

func (m umodel) startRestart() (tea.Model, tea.Cmd) {
	m.state = uStateRestarting
	m.rstStatuses = make([]rstStatus, len(m.toRestart))
	containers := make([]string, len(m.toRestart))
	for i, s := range m.toRestart {
		m.rstStatuses[i] = rstStatus{svc: s}
		containers[i] = s.Container
	}
	m.rstResults = make([]error, len(m.toRestart))
	m.ch = make(chan tea.Msg, 64)
	env := m.env
	go func() {
		upload.RestartContainers(env, containers, upload.RestartPause, func(idx int, container string, running, waiting, done, failed bool, out string) {
			m.ch <- rstMsg{idx: idx, running: running, waiting: waiting, done: done, failed: failed, out: out}
		})
	}()
	return m, m.listen()
}

func nilPtr(failed bool, out string) error {
	if failed {
		return fmt.Errorf("%s", out)
	}
	return nil
}

func (m umodel) listen() tea.Cmd {
	return func() tea.Msg {
		return <-m.ch
	}
}

func (m umodel) View() string {
	switch m.state {
	case uStateEnv:
		return m.uViewEnv()
	case uStateJars:
		return m.uViewJars()
	case uStateRunning:
		return m.uViewRunning()
	case uStateConfirmRestart:
		return m.uViewConfirmRestart()
	case uStateRestarting:
		return m.uViewRestarting()
	case uStateDone:
		return m.uViewDone()
	}
	return ""
}

func (m umodel) uViewEnv() string {
	var b strings.Builder
	b.WriteString(titleStyle("Select upload environment") + "\n\n")
	for i, e := range m.cfg.Environments {
		cursor := "  "
		if i == m.envIdx {
			cursor = cursorStyle("❯ ")
		}
		b.WriteString(fmt.Sprintf("%s%s  %s:%d  %s\n", cursor, projStyle(e.Name), dimStyle(e.Host), e.Port, dimStyle(e.DestDir)))
	}
	b.WriteString("\n" + helpStyle("↑/↓ move   enter select   q quit") + "\n")
	return b.String()
}

func (m umodel) uViewJars() string {
	var b strings.Builder
	b.WriteString(titleStyle("Select jars to upload") + "\n")
	b.WriteString(dimStyle(fmt.Sprintf("environment: %s   ·   destination: %s", m.env.Name, m.env.DestDir)) + "\n")
	b.WriteString(dimStyle(fmt.Sprintf("filter: %s", m.pattern)) + "\n\n")
	if len(m.jars) == 0 {
		b.WriteString(dimStyle("No built jars found (run `cst-cli mvn` first to package the projects).") + "\n\n")
		b.WriteString(helpStyle("Press q to quit.") + "\n")
		return b.String()
	}
	for i, j := range m.jars {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle("❯ ")
		}
		b.WriteString(fmt.Sprintf("%s%s %s %s\n", cursor, CheckBox(m.selected[i]), projStyle(j.Name), dimStyle("("+j.Project+")")))
	}
	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	b.WriteString("\n" + helpStyle("↑/↓ move   space toggle   a all   n none   enter upload   q quit") + "\n")
	b.WriteString("\n" + cursorStyle("▶") + " " + checkStyle(fmt.Sprintf("%d jar(s) selected — press enter to upload", count)) + "\n")
	return b.String()
}

func (m umodel) uViewRunning() string {
	var written, total int64
	for _, st := range m.upStatus {
		written += st.written
		total += st.total
	}
	var b strings.Builder
	b.WriteString(titleStyle("Uploading") + "  " + ProgressBar64(written, total, 24))
	if total > 0 {
		b.WriteString("  " + dimStyle(fmt.Sprintf("%s / %s", FormatSize(written), FormatSize(total))))
		b.WriteString("  " + runningStyle(fmt.Sprintf("%d%%", percent(written, total))))
	}
	b.WriteString("\n")
	b.WriteString(dimStyle(fmt.Sprintf("to %s@%s:%d:%s", m.env.User, m.env.Host, m.env.Port, m.env.DestDir)) + "\n\n")
	for i, j := range m.jars {
		st := upFileStatus{}
		if i < len(m.upStatus) {
			st = m.upStatus[i]
		}
		icon := runningStyle("●")
		switch {
		case st.failed:
			icon = failStyle("✗")
		case st.done:
			icon = successStyle("✓")
		case !st.running && !st.done:
			icon = dimStyle(" ")
		}
		bar := ProgressBar64(st.written, st.total, 16)
		size := dimStyle(FormatSize(st.written))
		if st.total > 0 {
			size = dimStyle(fmt.Sprintf("%s/%s", FormatSize(st.written), FormatSize(st.total)))
		}
		pct := runningStyle(fmt.Sprintf("%3d%%", percent(st.written, st.total)))
		if st.done && !st.failed {
			pct = successStyle("100%")
		}
		if st.failed {
			pct = failStyle("fail")
		}
		b.WriteString(fmt.Sprintf(" %s %s  %s %s %s\n", icon, projStyle(j.Name), bar, size, pct))
	}
	b.WriteString("\n" + helpStyle("Please wait…") + "\n")
	return b.String()
}

func percent(written, total int64) int {
	if total <= 0 {
		return 0
	}
	p := int(written * 100 / total)
	if p > 100 {
		return 100
	}
	return p
}

func (m umodel) uViewConfirmRestart() string {
	var b strings.Builder
	ok, fail := uploadCounts(m.results)
	b.WriteString(titleStyle("Upload complete") + "  ")
	if fail == 0 {
		b.WriteString(successStyle(fmt.Sprintf("%d/%d uploaded", ok, len(m.results))))
	} else {
		b.WriteString(failStyle(fmt.Sprintf("%d uploaded, %d failed", ok, fail)))
	}
	b.WriteString("\n\n")
	b.WriteString(titleStyle("Restart docker services?") + "\n")
	b.WriteString(restartRecordLine(0, 0, len(m.toRestart)) + "\n")
	b.WriteString(dimStyle(fmt.Sprintf("one at a time on %s, wait %s after each", m.env.Host, upload.RestartPause)) + "\n\n")
	b.WriteString(renderRestartRows(restartRowSpecs(m.toRestart, nil, nil)))
	if len(m.unmatched) > 0 {
		b.WriteString("\n" + dimStyle("no mapping in deploy.yaml:") + "\n")
		for _, name := range m.unmatched {
			b.WriteString(dimStyle("  · "+name) + "\n")
		}
	}
	b.WriteString("\n" + helpStyle("enter restart   n skip   q quit") + "\n")
	return b.String()
}

func (m umodel) uViewRestarting() string {
	ok, fail, total := liveRestartCounts(m.rstStatuses)
	var b strings.Builder
	b.WriteString(titleStyle("Restarting docker") + "  " + ProgressBar(ok+fail, total, 24) + "\n")
	b.WriteString(restartRecordLine(ok, fail, total) + "\n")
	b.WriteString(dimStyle(fmt.Sprintf("on %s@%s — one at a time, wait %s after each", m.env.User, m.env.Host, upload.RestartPause)) + "\n\n")
	specs := make([]restartRow, len(m.rstStatuses))
	for i, st := range m.rstStatuses {
		label := dimStyle("pending")
		icon := dimStyle("·")
		switch {
		case st.failed:
			icon = failStyle("✗")
			label = failStyle("failed")
		case st.done:
			icon = successStyle("✓")
			label = successStyle("restarted")
		case st.waiting:
			icon = runningStyle("●")
			label = runningStyle("waiting " + upload.RestartPause.String())
		case st.running:
			icon = runningStyle("●")
			label = runningStyle("docker restart")
		}
		specs[i] = restartRow{
			index:     i + 1,
			total:     total,
			icon:      icon,
			name:      st.svc.Name,
			container: st.svc.Container,
			extra:     label,
		}
	}
	b.WriteString(renderRestartRows(specs))
	b.WriteString("\n" + helpStyle("Please wait…") + "\n")
	return b.String()
}

func (m umodel) uViewDone() string {
	if m.err != nil {
		return failStyle("Error: ") + m.err.Error() + "\n\n" + helpStyle("Press q to quit.") + "\n"
	}
	if len(m.results) == 0 {
		return dimStyle("No built jars found (run `cst-cli mvn` first to package the projects).") +
			"\n\n" + helpStyle("Press q to quit.") + "\n"
	}
	ok, fail := uploadCounts(m.results)
	var b strings.Builder
	b.WriteString(titleStyle("Upload complete") + "  ")
	if fail == 0 {
		b.WriteString(successStyle(fmt.Sprintf("%d/%d uploaded", ok, len(m.results))))
	} else {
		b.WriteString(failStyle(fmt.Sprintf("%d uploaded, %d failed", ok, fail)))
	}
	b.WriteString("\n" + dimStyle(fmt.Sprintf("to %s@%s:%d:%s", m.env.User, m.env.Host, m.env.Port, m.env.DestDir)) + "\n\n")
	for i, e := range m.results {
		if e == nil {
			b.WriteString(successStyle("✓ "+m.jars[i].Name) + dimStyle("  ("+m.jars[i].Project+")") + "\n")
		} else {
			b.WriteString(failStyle("✗ "+m.jars[i].Name) + "  " + e.Error() + "\n")
		}
	}

	b.WriteString("\n")
	switch {
	case m.deployCfgErr != nil:
		b.WriteString(dimStyle("Docker restart skipped: ") + failStyle(m.deployCfgErr.Error()) + "\n")
	case m.restartSkipped:
		b.WriteString(dimStyle("Docker restart skipped.") + "\n")
	case len(m.toRestart) == 0:
		b.WriteString(dimStyle("No matching docker services in deploy.yaml — nothing to restart.") + "\n")
	default:
		rok, rfail := 0, 0
		for _, e := range m.rstResults {
			if e == nil {
				rok++
			} else {
				rfail++
			}
		}
		b.WriteString(titleStyle("Docker restart") + "\n")
		b.WriteString(restartRecordLine(rok, rfail, len(m.rstResults)) + "\n\n")
		extras := make([]string, len(m.toRestart))
		icons := make([]string, len(m.toRestart))
		for i := range m.toRestart {
			if i < len(m.rstResults) && m.rstResults[i] != nil {
				icons[i] = failStyle("✗")
				extras[i] = failStyle(m.rstResults[i].Error())
			} else {
				icons[i] = successStyle("✓")
				extras[i] = successStyle("restarted")
			}
		}
		b.WriteString(renderRestartRows(restartRowSpecs(m.toRestart, icons, extras)))
	}
	if len(m.unmatched) > 0 {
		b.WriteString("\n" + dimStyle("uploaded but no mapping in deploy.yaml:") + "\n")
		for _, name := range m.unmatched {
			b.WriteString(dimStyle("  · "+name) + "\n")
		}
	}
	b.WriteString("\n" + helpStyle("Press q to quit.") + "\n")
	return b.String()
}

func uploadCounts(results []error) (ok, fail int) {
	for _, e := range results {
		if e == nil {
			ok++
		} else {
			fail++
		}
	}
	return ok, fail
}

func liveRestartCounts(statuses []rstStatus) (ok, fail, total int) {
	total = len(statuses)
	for _, st := range statuses {
		if st.failed {
			fail++
			continue
		}
		if st.done || st.waiting {
			ok++
		}
	}
	return ok, fail, total
}

func restartRecordLine(ok, fail, total int) string {
	line := fmt.Sprintf("已重启 %d / 共 %d", ok, total)
	if fail > 0 {
		return titleStyle("重启记录") + "  " + failStyle(line) + "  " + failStyle(fmt.Sprintf("失败 %d", fail))
	}
	style := successStyle
	if ok == 0 && total > 0 {
		style = dimStyle
	}
	return titleStyle("重启记录") + "  " + style(line)
}

type restartRow struct {
	index     int
	total     int
	icon      string
	name      string
	container string
	extra     string
}

func restartRowSpecs(svcs []deploy.Service, icons, extras []string) []restartRow {
	rows := make([]restartRow, len(svcs))
	for i, s := range svcs {
		icon := dimStyle("●")
		if i < len(icons) && icons[i] != "" {
			icon = icons[i]
		}
		extra := ""
		if i < len(extras) {
			extra = extras[i]
		}
		rows[i] = restartRow{
			index:     i + 1,
			total:     len(svcs),
			icon:      icon,
			name:      s.Name,
			container: s.Container,
			extra:     extra,
		}
	}
	return rows
}

func renderRestartRows(rows []restartRow) string {
	if len(rows) == 0 {
		return ""
	}
	idxW := lipgloss.Width(fmt.Sprintf("%d/%d", rows[0].total, rows[0].total))
	iconW := 1
	nameW := 0
	actW := 0
	for _, r := range rows {
		iconW = max(iconW, lipgloss.Width(r.icon))
		nameW = max(nameW, lipgloss.Width(r.name))
		actW = max(actW, lipgloss.Width("docker restart "+r.container))
	}
	var b strings.Builder
	for _, r := range rows {
		idx := padRight(dimStyle(fmt.Sprintf("%d/%d", r.index, r.total)), idxW)
		icon := padRight(r.icon, iconW)
		name := padRight(projStyle(r.name), nameW)
		act := padRight(dimStyle("docker restart "+r.container), actW)
		b.WriteString("  ")
		if r.extra != "" {
			b.WriteString(tableRow(idx, icon, name, act, r.extra))
		} else {
			b.WriteString(tableRow(idx, icon, name, act))
		}
	}
	return b.String()
}

type upFileStatus struct {
	written int64
	total   int64
	running bool
	done    bool
	failed  bool
}

type upMsg struct {
	idx     int
	written int64
	total   int64
	running bool
	err     error
}

type rstMsg struct {
	idx     int
	running bool
	waiting bool
	done    bool
	failed  bool
	out     string
}

// RunUpload launches the interactive upload subcommand.
func RunUpload(configPath, env, pattern, deployConfigPath string) error {
	cfg, err := upload.LoadConfig(configPath)
	if err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	projects, err := maven.FindMavenProjects(dir)
	if err != nil {
		return err
	}
	jarList := jars.FindJars(toJarsProjects(projects))
	if pattern == "" {
		pattern = jars.DefaultJarPattern
	}
	jarList = jars.FilterByName(jarList, jars.ParsePatterns(pattern))
	dcfg, dErr := deploy.LoadConfig(deployConfigPath)
	p := tea.NewProgram(initialUModel(cfg, dcfg, dErr, env, jarList, pattern), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
