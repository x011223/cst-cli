// Package docker lists and restarts remote Docker containers over SSH.
package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/crypto/ssh"
)

// SuccessMarker is the log line snippet that means a service finished starting.
const SuccessMarker = "启动成功"

// DefaultLogTimeout is how long to wait for SuccessMarker after docker restart.
const DefaultLogTimeout = 2 * time.Minute

// Container is one row from `docker ps`.
type Container struct {
	Name   string
	State  string // running, exited, …
	Status string // Up 2 hours, Exited (0) 3 days ago
	Image  string
}

// List returns containers on the remote host (docker ps -a).
func List(client *ssh.Client) ([]Container, error) {
	out, err := run(client, "docker ps -a --format '{{.Names}}|{{.State}}|{{.Status}}|{{.Image}}'")
	if err != nil {
		return nil, err
	}
	return ParsePS(out), nil
}

// ParsePS parses the custom `docker ps --format` table used by List.
func ParsePS(out string) []Container {
	var list []Container
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 3 {
			continue
		}
		name := strings.Split(parts[0], ",")[0]
		name = strings.TrimSpace(name)
		if name == "" || !validName(name) {
			continue
		}
		c := Container{
			Name:   name,
			State:  strings.TrimSpace(parts[1]),
			Status: strings.TrimSpace(parts[2]),
		}
		if len(parts) > 3 {
			c.Image = strings.TrimSpace(parts[3])
		}
		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool {
		ri, rj := list[i].State == "running", list[j].State == "running"
		if ri != rj {
			return ri
		}
		return list[i].Name < list[j].Name
	})
	return list
}

// IsReady reports whether a log line means the application finished starting.
func IsReady(line string) bool {
	return strings.Contains(line, SuccessMarker)
}

// RestartAndFollow restarts container, then follows `docker logs -f` until
// SuccessMarker appears, ctx is cancelled, or timeout elapses.
func RestartAndFollow(ctx context.Context, client *ssh.Client, name string, timeout time.Duration, onLine func(string)) error {
	if !validName(name) {
		return fmt.Errorf("invalid container name %q", name)
	}
	if timeout <= 0 {
		timeout = DefaultLogTimeout
	}
	if _, err := run(client, "docker restart "+name); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}
	if err := session.Start("docker logs -f --tail 80 " + name); err != nil {
		return err
	}

	found := make(chan struct{})
	var ready sync.Once
	signalReady := func() {
		ready.Do(func() { close(found) })
	}
	readErr := make(chan error, 2)
	go scanLines(stdout, onLine, signalReady, readErr)
	go scanLines(stderr, onLine, signalReady, readErr)

	select {
	case <-found:
		_ = session.Close()
		return nil
	case <-ctx.Done():
		_ = session.Close()
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timeout waiting for %s", SuccessMarker)
		}
		return ctx.Err()
	case err := <-readErr:
		_ = session.Close()
		if err != nil {
			return err
		}
		return fmt.Errorf("log stream ended before %s", SuccessMarker)
	}
}

func scanLines(r io.Reader, onLine func(string), signalReady func(), readErr chan error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if onLine != nil {
			onLine(line)
		}
		if IsReady(line) {
			signalReady()
			return
		}
	}
	if err := sc.Err(); err != nil {
		select {
		case readErr <- err:
		default:
		}
	}
}

func run(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	out, err := session.CombinedOutput(cmd)
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg == "" {
			msg = err.Error()
		}
		return msg, fmt.Errorf("%s", msg)
	}
	return msg, nil
}

func validName(name string) bool {
	if name == "" || len(name) > 128 {
		return false
	}
	for i, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			continue
		}
		if r == ':' && i > 0 {
			continue
		}
		return false
	}
	return true
}
