package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wujunqiang/cst-cli/internal/jars"
	"github.com/wujunqiang/cst-cli/internal/maven"
)

type jstate int

const (
	jStateSelect jstate = iota
	jStateRunning
	jStateDone
)

type jmodel struct {
	state    jstate
	dest     string
	pattern  string
	jars     []jars.JarFile
	cursor   int
	selected map[int]bool
	results  []jars.CopyResult
	doneCnt  int
	ch       chan tea.Msg
	err      error
}

func initialJModel(jarList []jars.JarFile, dest, pattern string) jmodel {
	sel := map[int]bool{}
	for i := range jarList {
		sel[i] = true // default: all selected
	}
	return jmodel{state: jStateSelect, dest: dest, pattern: pattern, jars: jarList, selected: sel}
}

func (m jmodel) Init() tea.Cmd { return nil }

func (m jmodel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case jStateSelect:
			return m.updateSelect(msg)
		case jStateDone:
			if msg.String() == "q" || msg.String() == "ctrl+c" || msg.String() == "esc" {
				return m, tea.Quit
			}
		}
	case jarCopyMsg:
		m.results[msg.idx].Err = msg.err
		m.doneCnt++
		if m.doneCnt == len(m.results) {
			m.state = jStateDone
			return m, nil
		}
		return m, m.listen()
	case errMsg:
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m jmodel) updateSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		return m.startCollect(chosen)
	}
	return m, nil
}

func (m jmodel) startCollect(chosen []jars.JarFile) (tea.Model, tea.Cmd) {
	m.state = jStateRunning
	m.jars = chosen // results now align 1:1 with m.jars
	m.results = make([]jars.CopyResult, len(chosen))
	m.ch = make(chan tea.Msg, 64)
	dst := jars.ExpandHome(m.dest)
	go func() {
		res, _ := jars.CopyJars(chosen, dst)
		for i, r := range res {
			m.results[i] = r
			m.ch <- jarCopyMsg{idx: i, err: r.Err}
		}
	}()
	return m, m.listen()
}

func (m jmodel) listen() tea.Cmd {
	return func() tea.Msg {
		return <-m.ch
	}
}

func (m jmodel) View() string {
	switch m.state {
	case jStateSelect:
		return m.jViewSelect()
	case jStateRunning:
		return m.jViewRunning()
	case jStateDone:
		return m.jViewDone()
	}
	return ""
}

func (m jmodel) jViewSelect() string {
	var b strings.Builder
	b.WriteString(titleStyle("Select jars to collect") + "\n")
	b.WriteString(dimStyle(fmt.Sprintf("destination: %s", jars.ExpandHome(m.dest))) + "\n")
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
	b.WriteString("\n" + helpStyle("↑/↓ move   space toggle   a all   n none   enter collect   q quit") + "\n")
	b.WriteString("\n" + cursorStyle("▶") + " " + checkStyle(fmt.Sprintf("%d jar(s) selected — press enter to collect", count)) + "\n")
	return b.String()
}

func (m jmodel) jViewRunning() string {
	var b strings.Builder
	b.WriteString(titleStyle("Collecting jars") + "  " + ProgressBar(m.doneCnt, len(m.results), 24) + "\n\n")
	for i, j := range m.jars {
		icon := dimStyle(" ")
		if i < m.doneCnt {
			if m.results[i].Err == nil {
				icon = successStyle("✓")
			} else {
				icon = failStyle("✗")
			}
		} else {
			icon = runningStyle("●")
		}
		b.WriteString(fmt.Sprintf(" %s %s %s\n", icon, projStyle(j.Name), dimStyle("("+j.Project+")")))
	}
	b.WriteString("\n" + helpStyle("Press q to quit.") + "\n")
	return b.String()
}

func (m jmodel) jViewDone() string {
	if m.err != nil {
		return failStyle("Error: ") + m.err.Error() + "\n\n" + helpStyle("Press q to quit.") + "\n"
	}
	if len(m.results) == 0 {
		return dimStyle("No jars collected.") + "\n\n" + helpStyle("Press q to quit.") + "\n"
	}
	ok, fail := 0, 0
	for _, r := range m.results {
		if r.Err == nil {
			ok++
		} else {
			fail++
		}
	}
	var b strings.Builder
	b.WriteString(titleStyle("Jars collected") + "  ")
	if fail == 0 {
		b.WriteString(successStyle(fmt.Sprintf("%d/%d copied", ok, len(m.results))))
	} else {
		b.WriteString(failStyle(fmt.Sprintf("%d copied, %d failed", ok, fail)))
	}
	b.WriteString("\n" + dimStyle(fmt.Sprintf("destination: %s", jars.ExpandHome(m.dest))) + "\n\n")
	for _, r := range m.results {
		if r.Err == nil {
			b.WriteString(successStyle("✓ "+r.Jar.Name) + dimStyle("  ("+r.Jar.Project+")") + "\n")
		} else {
			b.WriteString(failStyle("✗ "+r.Jar.Name) + "  " + r.Err.Error() + "\n")
		}
	}
	b.WriteString("\n" + helpStyle("Press q to quit.") + "\n")
	return b.String()
}

type jarCopyMsg struct {
	idx int
	err error
}

// toJarsProjects converts maven projects to jars.Project for jar discovery.
func toJarsProjects(mp []maven.Project) []jars.Project {
	out := make([]jars.Project, 0, len(mp))
	for _, p := range mp {
		out = append(out, jars.Project{Name: p.Name, Path: p.Path})
	}
	return out
}

// RunJars launches the interactive jar-collection subcommand.
func RunJars(dest, pattern string) error {
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
	p := tea.NewProgram(initialJModel(jarList, dest, pattern), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
