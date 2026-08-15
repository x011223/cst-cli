// Package git discovers git repositories under a directory and reports their
// uncommitted changes as a structured, tree-ready representation.
package git

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Change describes a single modified/untracked file in a repository.
type Change struct {
	Path   string // repository-relative path
	Status string // raw git status "XY" (e.g. " M", "??", "A ")
	Staged bool   // true when staged in the index
}

// RepoStatus holds the changes of one repository.
type RepoStatus struct {
	Name    string
	Path    string
	Branch  string
	Changes []Change
}

// HasChanges reports whether the repository has any modifications.
func (r RepoStatus) HasChanges() bool { return len(r.Changes) > 0 }

// Discover scans dir for git repositories and returns those with changes,
// sorted by name. A repository is any directory (or dir itself) that contains
// a .git entry. Immediate sub-directories are considered; nested repos inside a
// discovered repo are not separately scanned.
func Discover(dir string) ([]RepoStatus, error) {
	dirs := map[string]struct{}{}
	if isRepo(dir) {
		dirs[dir] = struct{}{}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if isRepo(p) {
			dirs[p] = struct{}{}
		}
	}

	var repos []RepoStatus
	for d := range dirs {
		rs, err := statusOf(d)
		if err != nil {
			continue
		}
		if rs.HasChanges() {
			repos = append(repos, rs)
		}
	}
	sort.Slice(repos, func(i, j int) bool { return repos[i].Name < repos[j].Name })
	return repos, nil
}

func isRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func statusOf(dir string) (RepoStatus, error) {
	branch, _ := runGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := runGit(dir, "status", "--porcelain=v1", "-uall")
	if err != nil {
		return RepoStatus{}, err
	}
	rs := RepoStatus{Name: filepath.Base(dir), Path: dir, Branch: strings.TrimSpace(branch)}
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" || len(line) < 3 {
			continue
		}
		xy := strings.TrimSpace(line[:2])
		rest := line[3:]
		path := rest
		if i := strings.Index(rest, " -> "); i >= 0 {
			path = rest[i+4:]
		}
		rs.Changes = append(rs.Changes, Change{
			Path:   path,
			Status: xy,
			Staged: len(xy) > 0 && xy[0] != ' ' && xy[0] != '?',
		})
	}
	return rs, nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// StatusColor maps a git status code to a short colored-agnostic label.
func StatusLabel(status string) string {
	switch status {
	case "??":
		return "??"
	case "A ", " A", "AM":
		return "A"
	case "D ", " D":
		return "D"
	case "M ", " M", "MM":
		return "M"
	case "R ":
		return "R"
	case "UU", "AA", "DD":
		return "!!"
	default:
		return status
	}
}
