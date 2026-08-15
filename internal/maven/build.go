package maven

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// Phase represents a Maven lifecycle phase executed by the tool.
type Phase string

const (
	PhaseClean   Phase = "clean"
	PhaseCompile Phase = "compile"
	PhasePackage Phase = "package"
)

// Phases is the ordered list of phases run inside a single project (serial).
var Phases = []Phase{PhaseClean, PhaseCompile, PhasePackage}

// PhaseResult holds the outcome of a single phase for a project.
type PhaseResult struct {
	Phase Phase
	Err   error
	Out   string
}

// BuildResult holds the per-project build outcome.
type BuildResult struct {
	Project  Project
	Results  []PhaseResult
	FailedAt Phase // zero value means success for all phases
}

// RunBuilds runs the selected projects' builds. Projects run in parallel, and
// within a project the three phases run serially. The given profile is passed
// to Maven via -P (e.g. "dev"/"prod"); an empty profile activates the default.
// Progress is reported via the onPhase callback (running=false with failed=true
// means the phase errored, and out holds the Maven output); final results are
// returned.
func RunBuilds(projects []Project, profile string, onPhase func(projectIndex int, phase Phase, running bool, out string, failed bool), onDone func(projectIndex int)) []BuildResult {
	results := make([]BuildResult, len(projects))
	var wg sync.WaitGroup
	for i, p := range projects {
		wg.Add(1)
		go func(idx int, proj Project) {
			defer wg.Done()
			res := BuildResult{Project: proj}
			for _, phase := range Phases {
				onPhase(idx, phase, true, "", false)
				out, err := runMaven(proj.Path, string(phase), profile)
				failed := err != nil
				res.Results = append(res.Results, PhaseResult{Phase: phase, Err: err, Out: out})
				onPhase(idx, phase, false, out, failed)
				if err != nil {
					res.FailedAt = phase
					break
				}
			}
			results[idx] = res
			onDone(idx)
		}(i, p)
	}
	wg.Wait()
	return results
}

// runMaven executes `mvn -B [-P<profile>] <phase>` in the given project
// directory and returns the combined output. On failure the output (including
// the Maven error) is returned alongside the error so callers can display it.
func runMaven(dir, phase, profile string) (string, error) {
	args := []string{"-B"}
	if profile != "" {
		args = append(args, "-P"+profile)
	}
	args = append(args, phase)
	cmd := exec.Command("mvn", args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// FormatResult builds a human-readable summary for a single build result.
func FormatResult(r BuildResult) string {
	if r.FailedAt == "" {
		return fmt.Sprintf("✓ %s  (clean, compile, package all succeeded)", r.Project.Name)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("✗ %s  failed at phase: %s\n", r.Project.Name, r.FailedAt))
	for _, pr := range r.Results {
		if pr.Err != nil {
			b.WriteString(fmt.Sprintf("---- mvn %s output ----\n", pr.Phase))
			b.WriteString(pr.Out)
			b.WriteString("\n")
		}
	}
	return b.String()
}
