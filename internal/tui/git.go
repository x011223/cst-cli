package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wujunqiang/cst-cli/internal/git"
)

var (
	dirStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	fileStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	countStyle = func(n int) string {
		return lipgloss.NewStyle().Bold(true).Foreground(yellow).Render(fmt.Sprintf("%d", n))
	}
	cursorStyle2 = lipgloss.NewStyle().Foreground(accent).Bold(true)
)

func statusColor(s string) lipgloss.Color {
	switch git.StatusLabel(s) {
	case "??":
		return yellow
	case "A":
		return green
	case "M":
		return yellow
	case "D", "!!":
		return red
	case "R":
		return cyan
	default:
		return lipgloss.Color("245")
	}
}

// gline is one rendered row of the git status view.
type gline struct {
	isHeader  bool
	repoIdx   int
	isDir     bool
	name      string
	status    string // branch for header, git status for files
	collapsed bool
	key       string // toggle key: "repo:<idx>" for header, path for dirs
	depth     int
	prefix    string
}

type gitModel struct {
	repos         []git.RepoStatus
	trees         []*git.TreeNode
	width         int
	height        int
	offset        int
	cursor        int
	repoCollapsed map[int]bool
	dirCollapsed  map[string]bool
}

func initialGitModel(repos []git.RepoStatus) gitModel {
	trees := make([]*git.TreeNode, len(repos))
	dirCollapsed := map[string]bool{}
	for i, r := range repos {
		trees[i] = r.Tree()
		markCollapsed(trees[i], "repo"+strconv.Itoa(i), dirCollapsed)
	}
	return gitModel{
		repos:         repos,
		trees:         trees,
		repoCollapsed: map[int]bool{},
		dirCollapsed:  dirCollapsed,
	}
}

// markCollapsed marks every directory node as collapsed so folders start
// folded by default (repo headers themselves remain expanded).
func markCollapsed(n *git.TreeNode, prefix string, collapsed map[string]bool) {
	for _, c := range n.SortedChildren() {
		key := prefix + "/" + c.Name
		if !c.IsFile {
			collapsed[key] = true
			markCollapsed(c, key, collapsed)
		}
	}
}

func (m gitModel) Init() tea.Cmd { return nil }

func (m gitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			m.cursor++
		case "pgup":
			m.cursor -= m.pageSize()
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "pgdown", " ":
			if msg.String() == "pgdown" {
				m.cursor += m.pageSize()
			} else {
				m.toggle()
			}
		case "enter":
			m.toggle()
		}
	}
	return m, nil
}

func (m *gitModel) toggle() {
	gls := m.buildVisible()
	if m.cursor < 0 || m.cursor >= len(gls) {
		return
	}
	g := gls[m.cursor]
	if g.isHeader {
		m.repoCollapsed[g.repoIdx] = !g.collapsed
	} else if g.isDir {
		m.dirCollapsed[g.key] = !g.collapsed
	}
}

func (m gitModel) pageSize() int {
	if m.height > 2 {
		return m.height - 2
	}
	return 1
}

// buildVisible walks the repo trees honoring collapsed state and returns the
// rows that should be displayed, with tree prefixes precomputed.
func (m gitModel) buildVisible() []gline {
	if len(m.repos) == 0 {
		return nil
	}
	var lines []gline
	for ri, repo := range m.repos {
		collapsed := m.repoCollapsed[ri]
		lines = append(lines, gline{isHeader: true, repoIdx: ri, name: repo.Name, status: repo.Branch, collapsed: collapsed, key: "repo:" + strconv.Itoa(ri)})
		if collapsed {
			continue
		}
		root := m.trees[ri]
		var walk func(n *git.TreeNode, prefix string, depth int, lastStack []bool)
		walk = func(n *git.TreeNode, prefix string, depth int, lastStack []bool) {
			kids := n.SortedChildren()
			for i, c := range kids {
				last := i == len(kids)-1
				var b strings.Builder
				for _, l := range lastStack {
					if l {
						b.WriteString("    ")
					} else {
						b.WriteString("│   ")
					}
				}
				connector := "└── "
				if !last {
					connector = "├── "
				}
				if depth > 0 {
					b.WriteString(connector)
				}
				key := prefix + "/" + c.Name
				if c.IsFile {
					lines = append(lines, gline{repoIdx: ri, isDir: false, name: c.Name, status: c.Status, depth: depth, prefix: b.String()})
				} else {
					dc := m.dirCollapsed[key]
					lines = append(lines, gline{repoIdx: ri, isDir: true, name: c.Name, depth: depth, prefix: b.String(), collapsed: dc, key: key})
					if !dc {
						walk(c, key, depth+1, append(lastStack, last))
					}
				}
			}
		}
		walk(root, "repo"+strconv.Itoa(ri), 0, nil)
	}
	return lines
}

func (m gitModel) renderLine(g gline, selected bool) string {
	caret := "▼ "
	if g.collapsed {
		caret = "▶ "
	}
	switch {
	case g.isHeader:
		rep := m.repos[g.repoIdx]
		return caret + titleStyle("● "+g.name) + dimStyle("  ("+g.status+")") +
			"  " + countStyle(len(rep.Changes)) + dimStyle(" changed")
	case g.isDir:
		return g.prefix + caret + dirStyle.Render(g.name)
	default:
		return g.prefix + lipgloss.NewStyle().Foreground(statusColor(g.status)).Render(git.StatusLabel(g.status)) +
			" " + fileStyle.Render(g.name)
	}
}

func (m gitModel) View() string {
	footer := helpStyle("↑/↓ move   enter/space expand   pgup/pgdn page   q quit")
	if len(m.repos) == 0 {
		return dimStyle("No git repositories with changes in this directory.") + "\n" + footer
	}
	gls := m.buildVisible()
	if m.cursor > len(gls)-1 {
		m.cursor = len(gls) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	avail := m.height - 1
	if avail < 1 {
		avail = 1
	}
	if m.offset > len(gls)-avail {
		m.offset = max(0, len(gls)-avail)
	}
	if m.offset < 0 {
		m.offset = 0
	}
	end := m.offset + avail
	if end > len(gls) {
		end = len(gls)
	}
	var b strings.Builder
	for i := m.offset; i < end; i++ {
		line := m.renderLine(gls[i], i == m.cursor)
		if i == m.cursor {
			line = cursorStyle2.Render("❯ ") + line
		} else {
			line = "  " + line
		}
		b.WriteString(line + "\n")
	}
	return b.String() + footer
}

// RunGitStatus launches the interactive git change viewer for the cwd.
func RunGitStatus() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	repos, err := git.Discover(dir)
	if err != nil {
		return err
	}
	p := tea.NewProgram(initialGitModel(repos), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
