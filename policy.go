package main

import (
	"fmt"
	"regexp"
	"strings"
)

var defaultDenyPatterns = []string{
	`(?i)\brm\s+-rf\s+/(?:\s|$)`,
	`(?i)\brm\s+-rf\s+\*`,
	`(?i)\brm\s+-rf\s+--no-preserve-root`,
	`(?i)\bmkfs(\.[a-z0-9]+)?\b`,
	`(?i)\bdd\b.*\bof=/dev/`,
	`(?i)\bshutdown\b|\breboot\b|\bpoweroff\b|\bhalt\b`,
	`(?i):\s*\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;`,
}

type CommandPolicy struct {
	allow []*regexp.Regexp
	deny  []*regexp.Regexp
}

func NewCommandPolicy(cfg PolicyConfig, allowDangerous bool) (*CommandPolicy, error) {
	var allow []*regexp.Regexp
	for _, p := range cfg.AllowPatterns {
		rx, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid allow pattern %q: %w", p, err)
		}
		allow = append(allow, rx)
	}

	denyList := cfg.DenyPatterns
	if !allowDangerous {
		denyList = append(defaultDenyPatterns, denyList...)
	}
	var deny []*regexp.Regexp
	for _, p := range denyList {
		rx, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid deny pattern %q: %w", p, err)
		}
		deny = append(deny, rx)
	}

	return &CommandPolicy{allow: allow, deny: deny}, nil
}

func (p *CommandPolicy) Check(command string) error {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return fmt.Errorf("command is empty")
	}
	if len(p.allow) > 0 {
		allowed := false
		for _, rx := range p.allow {
			if rx.MatchString(cmd) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("command blocked by allowlist policy")
		}
	}
	for _, rx := range p.deny {
		if rx.MatchString(cmd) {
			return fmt.Errorf("command blocked by denylist policy")
		}
	}
	return nil
}
