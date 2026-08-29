package jars

import (
	"os"
	"path"
	"path/filepath"
	"testing"
)

func TestApplicationJarPattern(t *testing.T) {
	ok := []string{
		"system-application-2.0.0.jar",
		"meeting-application-1.0.0.jar",
		"foo-application.jar",
	}
	for _, n := range ok {
		match, err := path.Match(ApplicationJarPattern, n)
		if err != nil || !match {
			t.Errorf("%q should match %s", n, ApplicationJarPattern)
		}
	}
	if match, _ := path.Match(ApplicationJarPattern, "commsoft-auth.jar"); match {
		t.Fatal("commsoft-auth.jar should not match")
	}
}

func TestFilterExact(t *testing.T) {
	in := []JarFile{
		{Name: "commsoft-auth.jar"},
		{Name: "system-application-2.0.0.jar"},
		{Name: "other.jar"},
	}
	got := FilterExact(in, []string{"commsoft-auth.jar", "system-application-2.0.0.jar"})
	if len(got) != 2 || got[0].Name != "commsoft-auth.jar" || got[1].Name != "system-application-2.0.0.jar" {
		t.Fatalf("got %+v", got)
	}
	if FilterExact(in, nil) != nil {
		t.Fatal("empty names should match nothing")
	}
}

func TestListDirAndClearDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a-application-1.jar"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}

	list, err := ListDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "a-application-1.jar" {
		t.Fatalf("list = %+v", list)
	}

	if err := ClearDir(dir); err != nil {
		t.Fatal(err)
	}
	list, err = ListDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty after clear, got %+v", list)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); !os.IsNotExist(err) {
		t.Fatal("non-jar files should also be cleared")
	}
	if _, err := os.Stat(filepath.Join(dir, "sub")); err != nil {
		t.Fatal("subdirectories should be left in place")
	}
}

func TestClearDirRefusesHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	if err := ClearDir(home); err == nil {
		t.Fatal("expected error clearing home")
	}
}
