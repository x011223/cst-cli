// Package deploy implements the remote deployment workflow: rename the old jar,
// move the newly uploaded jar into place, and restart the docker container.
// It runs on the deployment server (no SSH); file upload is out of scope.
package deploy

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Service maps a deployable artifact to its docker container.
type Service struct {
	Name      string `yaml:"name"`
	Jar       string `yaml:"jar"`       // jar file name, e.g. system-application-2.0.0.jar
	Container string `yaml:"container"` // docker container name/tag, e.g. commsoft-system:1.0.0
}

// Config is the deployment descriptor loaded from a YAML file.
type Config struct {
	JarDir   string    `yaml:"jarDir"` // where deployed jars live, default /data/cst/app/jar
	TmpDir   string    `yaml:"tmpDir"` // where freshly uploaded jars wait, default /tmp
	Services []Service `yaml:"services"`
}

// DefaultConfigPath returns ~/.config/cst-cli/deploy.yaml.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "cst-cli", "deploy.yaml")
}

// LoadConfig reads and validates the deployment config.
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
	if c.JarDir == "" {
		c.JarDir = "/data/cst/app/jar"
	}
	if c.TmpDir == "" {
		c.TmpDir = "/tmp"
	}
	if len(c.Services) == 0 {
		return nil, fmt.Errorf("no services defined in %s", path)
	}
	return &c, nil
}

// MatchServices maps jar file names to services, preserving jarNames order
// and skipping duplicate containers. Jars with no matching service are
// returned in unmatched.
func (c *Config) MatchServices(jarNames []string) (matched []Service, unmatched []string) {
	seen := make(map[string]bool, len(c.Services))
	for _, name := range jarNames {
		found := false
		for _, s := range c.Services {
			if s.Jar != name {
				continue
			}
			found = true
			if !seen[s.Container] {
				seen[s.Container] = true
				matched = append(matched, s)
			}
			break
		}
		if !found {
			unmatched = append(unmatched, name)
		}
	}
	return matched, unmatched
}
