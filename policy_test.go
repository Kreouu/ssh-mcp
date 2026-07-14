package main

import (
	"strings"
	"testing"
)

func TestCommandPolicyDefaultDenyPatterns(t *testing.T) {
	policy, err := NewCommandPolicy(PolicyConfig{}, false)
	if err != nil {
		t.Fatal(err)
	}

	blocked := []string{
		"rm -rf /",
		"rm -rf *",
		"rm -rf --no-preserve-root /",
		"mkfs.ext4 /dev/sda1",
		"dd if=/dev/zero of=/dev/sda",
		"sudo reboot",
		":(){ :|:& };",
	}
	for _, command := range blocked {
		t.Run(command, func(t *testing.T) {
			if err := policy.Check(command); err == nil || !strings.Contains(err.Error(), "denylist") {
				t.Fatalf("Check(%q) error = %v, want denylist error", command, err)
			}
		})
	}

	for _, command := range []string{"ls -la", "rm -rf /tmp/build"} {
		if err := policy.Check(command); err != nil {
			t.Fatalf("Check(%q) returned unexpected error: %v", command, err)
		}
	}
}

func TestCommandPolicyAllowAndCustomDenyPatterns(t *testing.T) {
	policy, err := NewCommandPolicy(PolicyConfig{
		AllowPatterns: []string{`^ls(?:\s|$)`},
		DenyPatterns:  []string{`(?i)secret`},
	}, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := policy.Check("ls /tmp"); err != nil {
		t.Fatalf("allowed command returned error: %v", err)
	}
	if err := policy.Check("pwd"); err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("non-allowlisted command error = %v, want allowlist error", err)
	}
	if err := policy.Check("ls secret"); err == nil || !strings.Contains(err.Error(), "denylist") {
		t.Fatalf("custom denied command error = %v, want denylist error", err)
	}
}

func TestCommandPolicyAllowDangerousOnlyDisablesDefaults(t *testing.T) {
	policy, err := NewCommandPolicy(PolicyConfig{DenyPatterns: []string{`blocked`}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := policy.Check("reboot"); err != nil {
		t.Fatalf("default deny pattern remained enabled: %v", err)
	}
	if err := policy.Check("echo blocked"); err == nil {
		t.Fatal("custom deny pattern was disabled")
	}
}

func TestCommandPolicyRejectsInvalidPatternsAndEmptyCommands(t *testing.T) {
	if _, err := NewCommandPolicy(PolicyConfig{AllowPatterns: []string{"["}}, false); err == nil {
		t.Fatal("invalid allow pattern was accepted")
	}
	if _, err := NewCommandPolicy(PolicyConfig{DenyPatterns: []string{"["}}, false); err == nil {
		t.Fatal("invalid deny pattern was accepted")
	}

	policy, err := NewCommandPolicy(PolicyConfig{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := policy.Check("   "); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty command error = %v", err)
	}
}
