package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Hosts  []HostConfig `yaml:"hosts"`
	Policy PolicyConfig `yaml:"policy"`
}

type ServerConfig struct {
	Name              string `yaml:"name"`
	Version           string `yaml:"version"`
	DefaultTimeoutSec int    `yaml:"default_timeout_sec"`
	ConnectTimeoutSec int    `yaml:"connect_timeout_sec"`
	MaxOutputBytes    int64  `yaml:"max_output_bytes"`
	AllowDangerous    bool   `yaml:"allow_dangerous"`
}

type HostConfig struct {
	Name                  string        `yaml:"name"`
	Address               string        `yaml:"address"`
	Port                  int           `yaml:"port"`
	User                  string        `yaml:"user"`
	Password              string        `yaml:"password"`
	PrivateKeyPath        string        `yaml:"private_key_path"`
	PrivateKey            string        `yaml:"private_key"`
	PrivateKeyPassphrase  string        `yaml:"private_key_passphrase"`
	ProxyJump             string        `yaml:"proxy_jump"`
	JumpHost              string        `yaml:"jump_host"`
	HostKey               HostKeyConfig `yaml:"host_key"`
	InsecureIgnoreHostKey bool          `yaml:"insecure_ignore_host_key"`
}

type HostKeyConfig struct {
	Mode           string   `yaml:"mode"`
	KnownHostsPath string   `yaml:"known_hosts_path"`
	Algorithms     []string `yaml:"algorithms"`
}

type PolicyConfig struct {
	AllowPatterns []string `yaml:"allow_patterns"`
	DenyPatterns  []string `yaml:"deny_patterns"`
}

func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Normalize(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Normalize() error {
	if c.Server.Name == "" {
		c.Server.Name = "ssh-mcp"
	}
	if c.Server.Version == "" {
		c.Server.Version = "0.1.0"
	}
	if c.Server.DefaultTimeoutSec <= 0 {
		c.Server.DefaultTimeoutSec = 120
	}
	if c.Server.ConnectTimeoutSec <= 0 {
		c.Server.ConnectTimeoutSec = 10
	}
	if c.Server.MaxOutputBytes <= 0 {
		c.Server.MaxOutputBytes = 1 << 20
	}

	if len(c.Hosts) == 0 {
		return errors.New("no hosts configured")
	}

	for i := range c.Hosts {
		h := &c.Hosts[i]
		h.Name = strings.TrimSpace(expandEnv(h.Name))
		h.Address = strings.TrimSpace(expandEnv(h.Address))
		h.User = strings.TrimSpace(expandEnv(h.User))
		h.Password = expandEnv(h.Password)
		h.PrivateKey = expandEnv(h.PrivateKey)
		h.PrivateKeyPath = expandPath(h.PrivateKeyPath)
		h.PrivateKeyPassphrase = expandEnv(h.PrivateKeyPassphrase)
		h.ProxyJump = strings.TrimSpace(expandEnv(h.ProxyJump))
		h.JumpHost = strings.TrimSpace(expandEnv(h.JumpHost))
		if h.ProxyJump == "" && h.JumpHost != "" {
			h.ProxyJump = h.JumpHost
		}
		if h.Port == 0 {
			h.Port = 22
		}
		if h.HostKey.Mode == "" {
			h.HostKey.Mode = "known_hosts"
		}
		if h.HostKey.KnownHostsPath == "" && h.HostKey.Mode == "known_hosts" {
			h.HostKey.KnownHostsPath = "~/.ssh/known_hosts"
		}
		h.HostKey.KnownHostsPath = expandPath(h.HostKey.KnownHostsPath)
		var hostKeyAlgorithms []string
		for j := range h.HostKey.Algorithms {
			algorithm := strings.TrimSpace(expandEnv(h.HostKey.Algorithms[j]))
			if algorithm != "" {
				hostKeyAlgorithms = append(hostKeyAlgorithms, algorithm)
			}
		}
		h.HostKey.Algorithms = hostKeyAlgorithms
		if h.InsecureIgnoreHostKey {
			h.HostKey.Mode = "insecure_ignore"
		}
	}

	return nil
}

func resolveConfigPath(flagPath string) string {
	if strings.TrimSpace(flagPath) != "" {
		return flagPath
	}
	if env := strings.TrimSpace(os.Getenv("SSH_MCP_CONFIG")); env != "" {
		return env
	}
	return filepath.Join(".", "config.yaml")
}

func expandEnv(value string) string {
	return os.ExpandEnv(value)
}

func expandPath(value string) string {
	value = strings.TrimSpace(expandEnv(value))
	if value == "" {
		return value
	}
	if strings.HasPrefix(value, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			value = filepath.Join(home, strings.TrimPrefix(value, "~"))
		}
	}
	return value
}
