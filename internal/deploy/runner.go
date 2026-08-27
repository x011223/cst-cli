package deploy

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// Step is a single executed action within a service deployment.
type Step struct {
	Name    string
	Command string
	Output  string
	Err     error
}

// ServiceResult holds the outcome of deploying one service.
type ServiceResult struct {
	Service    Service
	Steps      []Step
	FailedStep string // empty when every step succeeded
}

// StepHandler streams step progress: running=true while the step executes, then
// running=false with failed=true when it errored.
type StepHandler func(stepIdx int, name, command string, running, failed bool, out string)

// Deploy runs the deployment for s, reporting each step through h.
func (c *Config) Deploy(s Service, h StepHandler) ServiceResult {
	res := ServiceResult{Service: s}
	jarPath := fmt.Sprintf("%s/%s", c.JarDir, s.Jar)
	bakPath := jarPath + ".bak"
	newPath := fmt.Sprintf("%s/%s", c.TmpDir, s.Jar)

	// 1) the freshly uploaded jar must already be present (upload is manual)
	h(0, "check new jar", newPath, true, false, "")
	if _, err := os.Stat(newPath); err != nil {
		h(0, "check new jar", newPath, false, true, "")
		res.FailedStep = "check new jar"
		return res
	}
	h(0, "check new jar", newPath, false, false, "")

	// 2) back up the existing jar (skip if this is the first deploy)
	if _, err := os.Stat(jarPath); err == nil {
		cmd := fmt.Sprintf("mv %s %s", jarPath, bakPath)
		st := runStep("backup old jar", cmd, "mv", jarPath, bakPath)
		h(1, st.Name, st.Command, false, st.Err != nil, st.Output)
		if st.Err != nil {
			res.Steps = append(res.Steps, st)
			res.FailedStep = st.Name
			return res
		}
		res.Steps = append(res.Steps, st)
	} else {
		h(1, "backup old jar", fmt.Sprintf("(skipped: %s not found)", jarPath), false, false, "")
	}

	// 3) move the new jar into place
	cmd := fmt.Sprintf("mv %s %s", newPath, jarPath)
	st := runStep("move new jar", cmd, "mv", newPath, jarPath)
	h(2, st.Name, st.Command, false, st.Err != nil, st.Output)
	res.Steps = append(res.Steps, st)
	if st.Err != nil {
		res.FailedStep = st.Name
		return res
	}

	// 4) restart the docker container
	cmd = fmt.Sprintf("docker restart %s", s.Container)
	st = runStep("restart container", cmd, "docker", "restart", s.Container)
	h(3, st.Name, st.Command, false, st.Err != nil, st.Output)
	res.Steps = append(res.Steps, st)
	if st.Err != nil {
		res.FailedStep = st.Name
	}
	return res
}

func runStep(name, display, cmd string, args ...string) Step {
	c := exec.Command(cmd, args...)
	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf
	err := c.Run()
	out := buf.String()
	if out == "" {
		out = display
	}
	return Step{Name: name, Command: display, Output: out, Err: err}
}
