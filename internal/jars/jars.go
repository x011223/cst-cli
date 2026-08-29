// Package jars locates Maven build artifacts (target/*.jar) and copies them to a
// destination directory.
package jars

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// JarFile is a discovered build artifact.
type JarFile struct {
	Project string // owning Maven project name (for display)
	Path    string // absolute source path
	Name    string // base file name
}

// CopyResult is the outcome of copying one jar.
type CopyResult struct {
	Jar JarFile
	Dst string
	Err error
}

// FindJars recursively scans each project root for target/*.jar files,
// skipping repackaged originals (original-*.jar). Multi-module builds are
// handled because the walk descends into nested target directories.
func FindJars(projects []Project) []JarFile {
	var out []JarFile
	for _, p := range projects {
		walkProject(p, &out)
	}
	return out
}

// Project is the minimal info FindJars needs from a Maven project.
type Project struct {
	Name string
	Path string
}

func walkProject(p Project, out *[]JarFile) {
	_ = filepath.WalkDir(p.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable paths
		}
		if d.IsDir() && d.Name() == "target" {
			entries, readErr := os.ReadDir(path)
			if readErr != nil {
				return nil
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				n := e.Name()
				if isDeployableJar(n) {
					*out = append(*out, JarFile{
						Project: p.Name,
						Path:    filepath.Join(path, n),
						Name:    n,
					})
				}
			}
		}
		return nil
	})
}

// isDeployableJar filters out jars that are not the runtime artifact: repackage
// originals and the standard Maven attached artifacts (sources/javadoc/tests).
func isDeployableJar(n string) bool {
	if !strings.HasSuffix(n, ".jar") {
		return false
	}
	if strings.HasPrefix(n, "original-") {
		return false
	}
	for _, suf := range []string{"-sources.jar", "-javadoc.jar", "-tests.jar", "-test.jar"} {
		if strings.HasSuffix(n, suf) {
			return false
		}
	}
	return true
}

// DefaultJarPattern selects the deployable application artifacts by convention.
var DefaultJarPattern = "*-application*.jar"

// ApplicationJarPattern is what `cst-cli mvn` copies into the staging folder.
var ApplicationJarPattern = "*-application*.jar"

// ParsePatterns splits a comma-separated pattern string into trimmed non-empty parts.
func ParsePatterns(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// FilterByName keeps only jars whose Name matches any of the given patterns
// (path.Match semantics, e.g. "*-application-*.jar").
func FilterByName(in []JarFile, patterns []string) []JarFile {
	if len(patterns) == 0 {
		return in
	}
	var out []JarFile
	for _, j := range in {
		for _, p := range patterns {
			if ok, _ := path.Match(p, j.Name); ok {
				out = append(out, j)
				break
			}
		}
	}
	return out
}

// FilterExact keeps only jars whose Name equals one of the given names.
func FilterExact(in []JarFile, names []string) []JarFile {
	if len(names) == 0 {
		return nil
	}
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[n] = struct{}{}
	}
	var out []JarFile
	for _, j := range in {
		if _, ok := want[j.Name]; ok {
			out = append(out, j)
		}
	}
	return out
}

// CopyJars copies every jar into dst, creating dst if needed. It overwrites
// existing files in dst with the same name.
func CopyJars(jars []JarFile, dst string) ([]CopyResult, error) {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return nil, fmt.Errorf("create dest %s: %w", dst, err)
	}
	res := make([]CopyResult, len(jars))
	for i, j := range jars {
		d := filepath.Join(dst, j.Name)
		res[i] = CopyResult{Jar: j, Dst: d, Err: copyFile(j.Path, d)}
	}
	return res, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// ListDir returns jar files sitting directly in dir (non-recursive).
func ListDir(dir string) ([]JarFile, error) {
	dir = ExpandHome(dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var out []JarFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(n, ".jar") {
			continue
		}
		out = append(out, JarFile{
			Path: filepath.Join(dir, n),
			Name: n,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ClearDir deletes files in dir (not subdirectories). It refuses to operate
// on empty, root, or home paths.
func ClearDir(dir string) error {
	dir = ExpandHome(dir)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if abs == "" || abs == "/" || abs == string(filepath.Separator) {
		return fmt.Errorf("refusing to clear %s", abs)
	}
	if home, err := os.UserHomeDir(); err == nil && abs == home {
		return fmt.Errorf("refusing to clear home directory %s", abs)
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(abs, 0o755)
		}
		return fmt.Errorf("read %s: %w", abs, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(abs, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// ExpandHome expands a leading ~ to the user's home directory.
func ExpandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
