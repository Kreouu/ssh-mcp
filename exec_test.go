package main

import (
	"strings"
	"testing"
)

func TestBuildShellScriptEscapesWorkdirAndEnvironment(t *testing.T) {
	script, err := buildShellScript(
		`printf '%s' "$TOKEN"`,
		`/tmp/project's files`,
		map[string]string{"TOKEN": `value's content`},
	)
	if err != nil {
		t.Fatal(err)
	}

	want := "cd '/tmp/project'\\''s files'\n" +
		"export TOKEN='value'\\''s content'\n" +
		`printf '%s' "$TOKEN"`
	if script != want {
		t.Fatalf("script mismatch\n got: %q\nwant: %q", script, want)
	}
}

func TestBuildShellScriptRejectsInvalidEnvironmentKey(t *testing.T) {
	_, err := buildShellScript("true", "", map[string]string{"INVALID-KEY": "value"})
	if err == nil || !strings.Contains(err.Error(), "invalid env key") {
		t.Fatalf("error = %v, want invalid env key", err)
	}
}

func TestResolveVersionPrefersBuildVersion(t *testing.T) {
	if got := resolveVersion("configured", "v1.2.3"); got != "1.2.3" {
		t.Fatalf("resolveVersion() = %q, want 1.2.3", got)
	}
	if got := resolveVersion("configured", ""); got != "configured" {
		t.Fatalf("resolveVersion() = %q, want configured", got)
	}
}
