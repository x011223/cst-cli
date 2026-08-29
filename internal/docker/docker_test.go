package docker

import (
	"strings"
	"testing"
)

func TestParsePS(t *testing.T) {
	out := strings.Join([]string{
		"commsoft-teaching|running|Up 3 hours|teaching:1.0",
		"commsoft-auth,auth|exited|Exited (0) 2 days ago|auth:1.0",
		"commsoft-system|running|Up 10 minutes|system:2.0",
		"",
		"badline",
	}, "\n")
	list := ParsePS(out)
	if len(list) != 3 {
		t.Fatalf("len=%d want 3: %+v", len(list), list)
	}
	if list[0].Name != "commsoft-system" || list[1].Name != "commsoft-teaching" {
		t.Fatalf("running should sort first by name: %s %s", list[0].Name, list[1].Name)
	}
	if list[2].Name != "commsoft-auth" || list[2].State != "exited" {
		t.Fatalf("alias/exited: %+v", list[2])
	}
}

func TestIsReady(t *testing.T) {
	if !IsReady("启动成功") {
		t.Fatal("expected ready")
	}
	if !IsReady("教学管理模块启动成功") {
		t.Fatal("expected ready")
	}
	if !IsReady("2026-08-29 INFO  (♥) 教学管理模块启动成功  (´▽`)") {
		t.Fatal("expected ready in decorated line")
	}
	if IsReady("Listening config") {
		t.Fatal("did not expect ready")
	}
}

func TestValidName(t *testing.T) {
	if !validName("commsoft-system") {
		t.Fatal("expected valid")
	}
	if validName("rm -rf /") || validName("") {
		t.Fatal("expected invalid")
	}
}
