// Package maven contains the Maven project detection and build logic.
// It has no dependency on the UI layer.
package maven

import (
	"os"
	"path/filepath"
	"sort"
)

// Project is a Maven project detected in the current directory.
type Project struct {
	Name string
	Path string
}

// FindMavenProjects scans dir for immediate sub-directories that contain a
// pom.xml file and returns them sorted by name.
func FindMavenProjects(dir string) ([]Project, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var projects []Project
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pom := filepath.Join(dir, e.Name(), "pom.xml")
		if info, err := os.Stat(pom); err == nil && !info.IsDir() {
			projects = append(projects, Project{Name: e.Name(), Path: filepath.Join(dir, e.Name())})
		}
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })
	return projects, nil
}
