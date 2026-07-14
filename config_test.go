package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigParsesYAML(t *testing.T) {
	t.Setenv("SSH_MCP_TEST_ADDRESS", "192.0.2.30")
	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := []byte(`
server:
  name: test-server
hosts:
  - name: test-host
    address: ${SSH_MCP_TEST_ADDRESS}
    user: test-user
`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Name != "test-server" || cfg.Hosts[0].Address != "192.0.2.30" {
		t.Fatalf("unexpected parsed configuration: %#v", cfg)
	}
}

func TestConfigNormalizeDefaultsAndExpandsHostValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SSH_MCP_HOST", "dev-board")
	t.Setenv("SSH_MCP_ADDRESS", "192.0.2.10")
	t.Setenv("SSH_MCP_PASSWORD", "test-password")
	t.Setenv("SSH_MCP_JUMP", "bastion")
	t.Setenv("SSH_MCP_ALGORITHM", "rsa-sha2-512")

	cfg := Config{Hosts: []HostConfig{{
		Name:           " ${SSH_MCP_HOST} ",
		Address:        " ${SSH_MCP_ADDRESS} ",
		User:           " test-user ",
		Password:       "${SSH_MCP_PASSWORD}",
		PrivateKeyPath: "~/.ssh/id_ed25519",
		JumpHost:       " ${SSH_MCP_JUMP} ",
		HostKey: HostKeyConfig{Algorithms: []string{
			" ssh-ed25519 ",
			"${SSH_MCP_ALGORITHM}",
			" ",
		}},
	}}}

	if err := cfg.Normalize(); err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Name != "ssh-mcp" || cfg.Server.Version != "0.1.0" {
		t.Fatalf("unexpected server identity: %#v", cfg.Server)
	}
	if cfg.Server.DefaultTimeoutSec != 120 || cfg.Server.ConnectTimeoutSec != 10 || cfg.Server.MaxOutputBytes != 1<<20 {
		t.Fatalf("unexpected server defaults: %#v", cfg.Server)
	}

	host := cfg.Hosts[0]
	if host.Name != "dev-board" || host.Address != "192.0.2.10" || host.User != "test-user" {
		t.Fatalf("host values were not normalized: %#v", host)
	}
	if host.Password != "test-password" || host.ProxyJump != "bastion" || host.Port != 22 {
		t.Fatalf("host defaults or environment expansion failed: %#v", host)
	}
	if host.PrivateKeyPath != filepath.Join(home, ".ssh/id_ed25519") {
		t.Fatalf("private key path = %q", host.PrivateKeyPath)
	}
	if host.HostKey.Mode != "known_hosts" || host.HostKey.KnownHostsPath != filepath.Join(home, ".ssh/known_hosts") {
		t.Fatalf("host key defaults failed: %#v", host.HostKey)
	}
	if len(host.HostKey.Algorithms) != 2 || host.HostKey.Algorithms[0] != "ssh-ed25519" || host.HostKey.Algorithms[1] != "rsa-sha2-512" {
		t.Fatalf("host key algorithms were not normalized: %#v", host.HostKey.Algorithms)
	}
}

func TestConfigNormalizeRejectsMissingHosts(t *testing.T) {
	var cfg Config
	if err := cfg.Normalize(); err == nil {
		t.Fatal("configuration without hosts was accepted")
	}
}

func TestConfigNormalizeSupportsLegacyHostOptions(t *testing.T) {
	cfg := Config{Hosts: []HostConfig{{
		Name:                  "legacy",
		Address:               "192.0.2.20",
		User:                  "user",
		JumpHost:              "jump",
		InsecureIgnoreHostKey: true,
	}}}
	if err := cfg.Normalize(); err != nil {
		t.Fatal(err)
	}
	if cfg.Hosts[0].ProxyJump != "jump" || cfg.Hosts[0].HostKey.Mode != "insecure_ignore" {
		t.Fatalf("legacy host options were not normalized: %#v", cfg.Hosts[0])
	}
}
