package tui

import "testing"

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
