package maven

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindMavenProjects(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "good"), 0755)
	os.WriteFile(filepath.Join(dir, "good", "pom.xml"), []byte("<project></project>"), 0644)
	os.MkdirAll(filepath.Join(dir, "bad"), 0755)
	os.WriteFile(filepath.Join(dir, "bad", "pom.xml"), []byte("<project></project>"), 0644)
	// a folder without pom.xml must be ignored
	os.MkdirAll(filepath.Join(dir, "notmaven"), 0755)

	ps, err := FindMavenProjects(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 {
		t.Fatalf("expected 2 projects, got %d: %v", len(ps), ps)
	}
	if ps[0].Name != "bad" || ps[1].Name != "good" {
		t.Fatalf("expected sorted order [bad good], got %v", ps)
	}
}

func TestRunBuildsReportsFailure(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a", "b"} {
		p := filepath.Join(dir, name)
		os.MkdirAll(p, 0755)
		os.WriteFile(filepath.Join(p, "pom.xml"), []byte("<project></project>"), 0644)
	}
	ps, _ := FindMavenProjects(dir)
	res := RunBuilds(ps, "", func(int, Phase, bool, string, bool) {}, func(int) {})
	for _, r := range res {
		if r.FailedAt == "" {
			t.Errorf("%s should have failed (no valid maven build)", r.Project.Name)
		}
	}
}
