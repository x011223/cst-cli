package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wujunqiang/cst-cli/internal/deploy"
	"github.com/wujunqiang/cst-cli/internal/jars"
	"github.com/wujunqiang/cst-cli/internal/maven"
)

type state int

const (
	stateEnv state = iota
	stateSelect
	stateRunning
	stateDone
)

// envOptions is the list of build environments (Maven profiles) selectable
// before a build. Add entries here to support more environments (e.g. test, cdp).
var envOptions = []string{"dev", "test", "prod"}

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
	state       state
	env         string
	envCursor   int
	projects    []maven.Project
	cursor      int
	selected    map[int]bool
	statuses    []projectStatus
	doneCnt     int
	results     []maven.BuildResult
	localJarDir string
	copyResults []jars.CopyResult
	copyErr     error
	ch          chan tea.Msg
	err         error
}

func initialModel(env string) model {
	m := model{selected: map[int]bool{}, env: env}
	if env != "" {
		m.state = stateSelect // preset env, skip the env screen
	} else {
		m.state = stateEnv
	}
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateEnv:
			return m.updateEnv(msg)
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
		// a project finished; only update progress here. The final result
		// slice is delivered via buildDoneMsg to avoid a data race.
		m.statuses[msg.idx].done = true
		m.doneCnt++
		return m, m.listen()
	case buildDoneMsg:
		// results arrive on the main goroutine, so the done view can rely on
		// m.results being fully populated.
		m.results = msg.results
		m.copyResults, m.copyErr = stageBuiltJars(msg.results, m.localJarDir)
		m.state = stateDone
		return m, nil
	case errMsg:
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m model) updateEnv(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.envCursor > 0 {
			m.envCursor--
		}
	case "down", "j":
		if m.envCursor < len(envOptions)-1 {
			m.envCursor++
		}
	case "enter", " ":
		m.env = envOptions[m.envCursor]
		m.state = stateSelect
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
		if err := jars.ClearDir(m.localJarDir); err != nil {
			m.err = fmt.Errorf("clear staging dir %s: %w", m.localJarDir, err)
			m.state = stateDone
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
		results := maven.RunBuilds(chosen, m.env,
			func(idx int, phase maven.Phase, running bool, out string, failed bool) {
				m.ch <- phaseMsg{idx: idx, phase: phase, running: running, failed: failed}
			},
			func(idx int) {
				m.ch <- doneMsg{idx: idx}
			})
		m.ch <- buildDoneMsg{results: results}
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
	case stateEnv:
		return m.viewEnv()
	case stateSelect:
		return m.viewSelect()
	case stateRunning:
		return m.viewRunning()
	case stateDone:
		return m.viewDone()
	}
	return ""
}

func (m model) viewEnv() string {
	var b strings.Builder
	b.WriteString(titleStyle("Select build environment") + "\n")
	b.WriteString(dimStyle("maps to the Maven profile (-P<env>) used for the build") + "\n\n")
	for i, e := range envOptions {
		cursor := "  "
		if i == m.envCursor {
			cursor = cursorStyle("❯ ")
		}
		b.WriteString(cursor + projStyle(e) + "\n")
	}
	b.WriteString("\n" + helpStyle("↑/↓ or k/j move   enter select   q quit") + "\n")
	return b.String()
}

func (m model) viewSelect() string {
	var b strings.Builder
	b.WriteString(titleStyle("Select Maven projects to build") + "\n")
	b.WriteString(dimStyle(fmt.Sprintf("env: %s   ·   staging: %s   ·   clean → compile → package", m.env, m.localJarDir)) + "\n\n")

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
	b.WriteString(dimStyle(fmt.Sprintf("env: %s   ·   Overall: %d/%d projects complete", m.env, m.doneCnt, total)) + "\n\n")
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
	ok, fail := 0, 0
	for _, r := range m.results {
		if r.FailedAt == "" {
			ok++
		} else {
			fail++
		}
	}

	var b strings.Builder
	b.WriteString(titleStyle("Build complete") + "  ")
	if fail == 0 {
		b.WriteString(successStyle(fmt.Sprintf("%d/%d succeeded", ok, len(m.results))))
	} else {
		b.WriteString(failStyle(fmt.Sprintf("%d succeeded, %d failed", ok, fail)))
	}
	b.WriteString("\n\n")

	envLabel := m.env
	if envLabel == "" {
		envLabel = "default"
	}
	tag := dimStyle("[" + envLabel + "] ")
	profile := ""
	if m.env != "" {
		profile = "-P" + m.env + " "
	}
	for _, r := range m.results {
		if r.FailedAt == "" {
			b.WriteString(successStyle("✓ "+tag+r.Project.Name) + dimStyle("  clean, compile, package all succeeded") + "\n")
			continue
		}
		cmd := fmt.Sprintf("mvn %s%s", profile, r.FailedAt)
		b.WriteString(failStyle("✗ "+tag+r.Project.Name) +
			dimStyle("   failing command: ") + failStyle(cmd) + "\n")
		for _, pr := range r.Results {
			if pr.Err == nil {
				continue
			}
			b.WriteString(errorBoxStyle.Render(mavenErrorSummary(pr.Out)) + "\n\n")
		}
	}

	b.WriteString("\n")
	switch {
	case m.copyErr != nil:
		b.WriteString(failStyle("Copy to staging failed: ") + m.copyErr.Error() + "\n")
	case len(m.copyResults) == 0:
		b.WriteString(dimStyle(fmt.Sprintf("No *-application*.jar copied to %s", m.localJarDir)) + "\n")
	default:
		okc, failc := 0, 0
		for _, r := range m.copyResults {
			if r.Err == nil {
				okc++
			} else {
				failc++
			}
		}
		b.WriteString(titleStyle("Staged jars") + "  ")
		if failc == 0 {
			b.WriteString(successStyle(fmt.Sprintf("%d copied → %s", okc, m.localJarDir)))
		} else {
			b.WriteString(failStyle(fmt.Sprintf("%d copied, %d failed → %s", okc, failc, m.localJarDir)))
		}
		b.WriteString("\n")
		for _, r := range m.copyResults {
			if r.Err == nil {
				b.WriteString(successStyle("  ✓ "+r.Jar.Name) + dimStyle("  ("+r.Jar.Project+")") + "\n")
			} else {
				b.WriteString(failStyle("  ✗ "+r.Jar.Name) + "  " + r.Err.Error() + "\n")
			}
		}
	}
	b.WriteString("\n" + helpStyle("Press q to quit.") + "\n")
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
type buildDoneMsg struct{ results []maven.BuildResult }
type errMsg struct{ err error }

// mavenErrorSummary extracts the most relevant error lines from raw Maven output
// so failures are readable at a glance: it prefers the BUILD FAILURE line and the
// [ERROR] messages, falling back to the last non-empty lines.
func mavenErrorSummary(out string) string {
	var errors, failure []string
	for _, l := range strings.Split(out, "\n") {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if strings.Contains(t, "BUILD FAILURE") {
			failure = append(failure, t)
		}
		if strings.HasPrefix(t, "[ERROR]") {
			e := strings.TrimSpace(strings.TrimPrefix(t, "[ERROR]"))
			if e != "" {
				errors = append(errors, e)
			}
		}
	}
	lines := append(failure, errors...)
	if len(lines) == 0 {
		// no [ERROR] markers: show the tail of the output
		nonEmpty := []string{}
		for _, l := range strings.Split(out, "\n") {
			if strings.TrimSpace(l) != "" {
				nonEmpty = append(nonEmpty, strings.TrimSpace(l))
			}
		}
		if len(nonEmpty) > 8 {
			nonEmpty = nonEmpty[len(nonEmpty)-8:]
		}
		lines = nonEmpty
	}
	if len(lines) > 12 {
		lines = lines[:12]
	}
	if len(lines) == 0 {
		return "(no output captured)"
	}
	return strings.Join(lines, "\n")
}

// RunMvnBuild launches the interactive Maven build subcommand. When env is
// non-empty it is used as the Maven profile and the environment screen is
// skipped; otherwise the user is prompted to pick one (dev/prod/...).
func RunMvnBuild(env string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	if env != "" {
		valid := false
		for _, e := range envOptions {
			if e == env {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("unknown environment %q (valid: %v)", env, envOptions)
		}
	}
	projects, err := maven.FindMavenProjects(dir)
	if err != nil {
		return err
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })

	m := initialModel(env)
	m.projects = projects
	dcfg, _ := deploy.LoadConfig("")
	m.localJarDir = deploy.ResolveLocalJarDir(dcfg)

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func stageBuiltJars(results []maven.BuildResult, dest string) ([]jars.CopyResult, error) {
	var ok []maven.Project
	for _, r := range results {
		if r.FailedAt == "" {
			ok = append(ok, r.Project)
		}
	}
	if len(ok) == 0 {
		return nil, nil
	}
	found := jars.FindJars(toJarsProjects(ok))
	found = jars.FilterByName(found, jars.ParsePatterns(jars.ApplicationJarPattern))
	if len(found) == 0 {
		return nil, nil
	}
	return jars.CopyJars(found, dest)
}
