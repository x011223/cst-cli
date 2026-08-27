package upload

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestValidContainerName(t *testing.T) {
	ok := []string{"commsoft-system", "commsoft_auth", "a", "svc.1", "name:1.0.0"}
	for _, n := range ok {
		if !validContainerName(n) {
			t.Errorf("validContainerName(%q) = false, want true", n)
		}
	}
	bad := []string{"", "has space", "rm -rf /", "a;reboot", "$(id)", "x/y"}
	for _, n := range bad {
		if validContainerName(n) {
			t.Errorf("validContainerName(%q) = true, want false", n)
		}
	}
}

func TestProgressReaderReportsBytes(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 64*1024)
	var last int64
	calls := 0
	pr := &progressReader{
		r:     bytes.NewReader(data),
		total: int64(len(data)),
		h: func(idx int, name, dest string, written, total int64, running, failed bool, out string) {
			calls++
			last = written
			if total != int64(len(data)) {
				t.Errorf("total = %d, want %d", total, len(data))
			}
			if !running || failed {
				t.Errorf("running=%v failed=%v", running, failed)
			}
		},
		last: time.Time{}, // force first report
	}
	n, err := io.Copy(io.Discard, pr)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(data)) {
		t.Fatalf("copied %d, want %d", n, len(data))
	}
	if calls == 0 {
		t.Fatal("expected progress callbacks")
	}
	if last != int64(len(data)) {
		t.Fatalf("last written = %d, want %d", last, len(data))
	}
}
