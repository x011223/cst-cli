// Package upload handles uploading build artifacts to remote servers over SFTP,
// using connection details from a local servers config (multiple environments).
package upload

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"

	"github.com/wujunqiang/cst-cli/internal/jars"
)

// RestartPause is how long to wait after each docker restart completes
// before starting the next one.
const RestartPause = 5 * time.Second

// Environment describes one remote server target.
type Environment struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DestDir  string `yaml:"destDir"` // remote directory the jars are uploaded into
}

// Config holds the configured environments.
type Config struct {
	Environments []Environment `yaml:"environments"`
}

// DefaultConfigPath returns ~/.config/cst-cli/servers.yaml.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "cst-cli", "servers.yaml")
}

// LoadConfig reads and validates the servers config.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if len(c.Environments) == 0 {
		return nil, fmt.Errorf("no environments defined in %s", path)
	}
	for i := range c.Environments {
		if c.Environments[i].Port == 0 {
			c.Environments[i].Port = 22
		}
		if c.Environments[i].DestDir == "" {
			c.Environments[i].DestDir = "/tmp"
		}
	}
	return &c, nil
}

const progressInterval = 100 * time.Millisecond

// UploadHandler streams per-file upload progress.
// running=true while bytes are in flight (written/total are set);
// running=false when the file finished (failed indicates the outcome).
type UploadHandler func(idx int, name, dest string, written, total int64, running, failed bool, out string)

// UploadAll uploads every jar to the environment in parallel, reporting progress
// through h. It returns the per-file errors (nil on success).
func UploadAll(env Environment, files []jars.JarFile, h UploadHandler) []error {
	errs := make([]error, len(files))
	var wg sync.WaitGroup
	for i, f := range files {
		wg.Add(1)
		go func(idx int, f jars.JarFile) {
			defer wg.Done()
			dst := filepath.Join(env.DestDir, f.Name)
			total := fileSize(f.Path)
			h(idx, f.Name, dst, 0, total, true, false, "")
			written, err := env.uploadOne(idx, f.Path, env.DestDir, h)
			if err != nil {
				h(idx, f.Name, dst, written, total, false, true, err.Error())
			} else {
				h(idx, f.Name, dst, written, total, false, false, "")
			}
			errs[idx] = err
		}(i, f)
	}
	wg.Wait()
	return errs
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func (e Environment) uploadOne(idx int, local, destDir string, h UploadHandler) (int64, error) {
	client, err := e.dial()
	if err != nil {
		return 0, err
	}
	defer client.Close()
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return 0, err
	}
	defer sftpClient.Close()

	if err := sftpClient.MkdirAll(destDir); err != nil {
		return 0, fmt.Errorf("mkdir %s: %w", destDir, err)
	}
	src, err := os.Open(local)
	if err != nil {
		return 0, err
	}
	defer src.Close()
	info, _ := src.Stat()
	total := int64(0)
	if info != nil {
		total = info.Size()
	}
	name := filepath.Base(local)
	dst := filepath.Join(destDir, name)
	out, err := sftpClient.Create(dst)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	pr := &progressReader{r: src, total: total, idx: idx, name: name, dest: dst, h: h}
	written, err := io.Copy(out, pr)
	if err != nil {
		return written, err
	}
	return written, out.Close()
}

// progressReader wraps a local file and reports byte progress while it is read.
type progressReader struct {
	r       io.Reader
	written int64
	total   int64
	idx     int
	name    string
	dest    string
	last    time.Time
	h       UploadHandler
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.written += int64(n)
	}
	now := time.Now()
	done := err != nil || (p.total > 0 && p.written >= p.total)
	if p.h != nil && (done || p.last.IsZero() || now.Sub(p.last) >= progressInterval) {
		p.last = now
		p.h(p.idx, p.name, p.dest, p.written, p.total, true, false, "")
	}
	return n, err
}

// RestartHandler streams per-container restart progress.
// running=true while `docker restart` executes; waiting=true during RestartPause;
// done=true after the pause (the item is fully finished).
type RestartHandler func(idx int, container string, running, waiting, done, failed bool, out string)

// RestartContainers restarts each docker container over SSH, one at a time.
// After each restart command finishes it waits pause (typically RestartPause)
// before moving to the next container.
func RestartContainers(env Environment, containers []string, pause time.Duration, h RestartHandler) []error {
	errs := make([]error, len(containers))
	if len(containers) == 0 {
		return errs
	}
	client, err := env.dial()
	if err != nil {
		for i, c := range containers {
			h(i, c, false, false, true, true, err.Error())
			errs[i] = err
		}
		return errs
	}
	defer client.Close()

	for i, c := range containers {
		h(i, c, true, false, false, false, "")
		out, err := restartOne(client, c)
		failed := err != nil
		errs[i] = err
		h(i, c, false, false, false, failed, out)
		if pause > 0 {
			h(i, c, false, true, false, false, "")
			time.Sleep(pause)
		}
		h(i, c, false, false, true, failed, out)
	}
	return errs
}

func restartOne(client *ssh.Client, container string) (string, error) {
	if !validContainerName(container) {
		return "", fmt.Errorf("invalid container name %q", container)
	}
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	out, err := session.CombinedOutput("docker restart " + container)
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg == "" {
			msg = err.Error()
		}
		return msg, fmt.Errorf("%s", msg)
	}
	return msg, nil
}

func validContainerName(name string) bool {
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

func (e Environment) Dial() (*ssh.Client, error) {
	return e.dial()
}

func (e Environment) dial() (*ssh.Client, error) {
	cfg := &ssh.ClientConfig{
		User: e.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(e.Password),
			ssh.KeyboardInteractive(func(name, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = e.Password
				}
				return answers, nil
			}),
		},
		// Internal tool on trusted networks; we skip host-key verification.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", e.Host, e.Port)
	return ssh.Dial("tcp", addr, cfg)
}
