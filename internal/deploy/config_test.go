package deploy

import "testing"

func TestResolveLocalJarDir(t *testing.T) {
	if got := ResolveLocalJarDir(nil); got == "" || got[0] == '~' {
		t.Fatalf("nil config should expand home, got %q", got)
	}
	cfg := &Config{LocalJarDir: "/tmp/cst-jars"}
	if got := ResolveLocalJarDir(cfg); got != "/tmp/cst-jars" {
		t.Fatalf("got %q", got)
	}
}

func TestMatchServices(t *testing.T) {
	cfg := &Config{Services: []Service{
		{Name: "system", Jar: "system-application-2.0.0.jar", Container: "commsoft-system"},
		{Name: "notify", Jar: "notify-application-1.0.0.jar", Container: "commsoft-notify"},
		{Name: "auth", Jar: "commsoft-auth.jar", Container: "commsoft-auth"},
	}}

	matched, unmatched := cfg.MatchServices([]string{
		"notify-application-1.0.0.jar",
		"missing.jar",
		"system-application-2.0.0.jar",
		"notify-application-1.0.0.jar",
	})
	if len(unmatched) != 1 || unmatched[0] != "missing.jar" {
		t.Fatalf("unmatched = %v", unmatched)
	}
	if len(matched) != 2 {
		t.Fatalf("matched len = %d, want 2: %+v", len(matched), matched)
	}
	if matched[0].Name != "notify" || matched[1].Name != "system" {
		t.Fatalf("order = %s, %s; want notify, system", matched[0].Name, matched[1].Name)
	}
}
