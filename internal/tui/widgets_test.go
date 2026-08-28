package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/wujunqiang/cst-cli/internal/deploy"
)

func TestFormatSize(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{2048, "2.0 KB"},
		{5 << 20, "5.0 MB"},
		{15 << 30, "15.0 GB"},
	}
	for _, c := range cases {
		if got := FormatSize(c.n); got != c.want {
			t.Errorf("FormatSize(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestProgressBar64Full(t *testing.T) {
	empty := ProgressBar64(0, 0, 4)
	if empty == "" {
		t.Fatal("empty bar should still render")
	}
	full := ProgressBar64(100, 100, 4)
	half := ProgressBar64(50, 100, 4)
	if full == half {
		t.Fatal("full and half bars should differ")
	}
}

func TestLiveRestartCounts(t *testing.T) {
	ok, fail, total := liveRestartCounts([]rstStatus{
		{done: true},
		{waiting: true},
		{failed: true},
		{running: true},
		{},
	})
	if total != 5 || ok != 2 || fail != 1 {
		t.Fatalf("got ok=%d fail=%d total=%d, want 2 1 5", ok, fail, total)
	}
}

func TestPadRightANSI(t *testing.T) {
	styled := projStyle("ab")
	padded := padRight(styled, 6)
	if lipgloss.Width(padded) != 6 {
		t.Fatalf("width = %d, want 6", lipgloss.Width(padded))
	}
	if padRight("abc", 2) != "abc" {
		t.Fatal("padRight should not truncate")
	}
}

func TestRenderRestartRowsAlignsColumns(t *testing.T) {
	out := renderRestartRows(restartRowSpecs([]deploy.Service{
		{Name: "meeting", Container: "commsoft-meeting"},
		{Name: "system", Container: "commsoft-system"},
		{Name: "teaching", Container: "commsoft-teaching"},
	}, nil, nil))
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines:\n%s", len(lines), out)
	}
	col := -1
	for i, line := range lines {
		plain := stripANSI(line)
		idx := strings.Index(plain, "docker restart")
		if idx < 0 {
			t.Fatalf("line %d missing action: %q", i, plain)
		}
		if col < 0 {
			col = idx
		} else if idx != col {
			t.Fatalf("line %d action at col %d, want %d\n%s", i, idx, col, dumpLines(lines))
		}
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func dumpLines(lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(stripANSI(line))
		b.WriteByte('\n')
	}
	return b.String()
}
