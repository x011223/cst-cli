package maven

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
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
// Progress is reported via onPhase (running=false with failed=true means the
// phase errored, and out holds the Maven output). onLine receives each stdout
// or stderr line as Maven prints it. Final results are returned.
func RunBuilds(ctx context.Context, projects []Project, profile string, onPhase func(projectIndex int, phase Phase, running bool, out string, failed bool), onLine func(projectIndex int, line string), onDone func(projectIndex int)) []BuildResult {
	if ctx == nil {
		ctx = context.Background()
	}
	results := make([]BuildResult, len(projects))
	var wg sync.WaitGroup
	for i, p := range projects {
		wg.Add(1)
		go func(idx int, proj Project) {
			defer wg.Done()
			res := BuildResult{Project: proj}
			for _, phase := range Phases {
				if ctx.Err() != nil {
					res.FailedAt = phase
					res.Results = append(res.Results, PhaseResult{Phase: phase, Err: ctx.Err(), Out: ctx.Err().Error()})
					break
				}
				onPhase(idx, phase, true, "", false)
				out, err := runMaven(ctx, proj.Path, string(phase), profile, func(line string) {
					if onLine != nil {
						onLine(idx, line)
					}
				})
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
// directory and streams combined output line by line. On failure the full
// output is still returned alongside the error.
func runMaven(ctx context.Context, dir, phase, profile string, onLine func(string)) (string, error) {
	args := []string{"-B"}
	if profile != "" {
		args = append(args, "-P"+profile)
	}
	args = append(args, phase)
	cmd := exec.CommandContext(ctx, "mvn", args...)
	cmd.Dir = dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	var mu sync.Mutex
	write := func(line string) {
		mu.Lock()
		buf.WriteString(line)
		buf.WriteByte('\n')
		mu.Unlock()
		if onLine != nil {
			onLine(line)
		}
	}
	var readers sync.WaitGroup
	readers.Add(2)
	go func() {
		defer readers.Done()
		scanLines(stdout, write)
	}()
	go func() {
		defer readers.Done()
		scanLines(stderr, write)
	}()
	err = cmd.Wait()
	readers.Wait()
	return buf.String(), err
}

func scanLines(r io.Reader, write func(string)) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		write(sc.Text())
	}
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
