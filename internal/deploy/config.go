// Package deploy loads the deployment descriptor: local staging folder
// and jar-to-container mapping. Remote upload path is per-environment destDir
// in servers.yaml, not this file.
package deploy

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/wujunqiang/cst-cli/internal/jars"
)

// DefaultLocalJarDir is where `cst-cli mvn` copies built application jars
// and where `cst-cli deploy` reads them from.
const DefaultLocalJarDir = "~/Documents/Jars"

// Service maps a deployable artifact to its docker container.
type Service struct {
	Name      string `yaml:"name"`
	Jar       string `yaml:"jar"`       // jar file name, e.g. system-application-2.0.0.jar
	Container string `yaml:"container"` // docker container name, e.g. commsoft-system
}

// Config is the deployment descriptor loaded from a YAML file.
type Config struct {
	LocalJarDir string    `yaml:"localJarDir"` // local staging folder for built jars
	Services    []Service `yaml:"services"`
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
	if c.LocalJarDir == "" {
		c.LocalJarDir = DefaultLocalJarDir
	}
	if len(c.Services) == 0 {
		return nil, fmt.Errorf("no services defined in %s", path)
	}
	return &c, nil
}

// ResolveLocalJarDir returns the expanded local staging folder.
func ResolveLocalJarDir(c *Config) string {
	dir := DefaultLocalJarDir
	if c != nil && c.LocalJarDir != "" {
		dir = c.LocalJarDir
	}
	return jars.ExpandHome(dir)
}

// JarNames returns the jar file names listed in services.
func (c *Config) JarNames() []string {
	if c == nil {
		return nil
	}
	out := make([]string, 0, len(c.Services))
	for _, s := range c.Services {
		if s.Jar != "" {
			out = append(out, s.Jar)
		}
	}
	return out
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
