package main

import (
	"fmt"
	"log"
)

type App struct {
	cfg    *Config
	policy *CommandPolicy
	hosts  map[string]*HostConfig
	logger *log.Logger
}

func NewApp(cfg *Config, policy *CommandPolicy, logger *log.Logger) (*App, error) {
	hosts := make(map[string]*HostConfig, len(cfg.Hosts))
	for i := range cfg.Hosts {
		h := &cfg.Hosts[i]
		if h.Name == "" {
			return nil, fmt.Errorf("host with empty name in config")
		}
		if _, exists := hosts[h.Name]; exists {
			return nil, fmt.Errorf("duplicate host name: %s", h.Name)
		}
		hosts[h.Name] = h
	}
	return &App{
		cfg:    cfg,
		policy: policy,
		hosts:  hosts,
		logger: logger,
	}, nil
}

func (a *App) getHost(name string) (*HostConfig, error) {
	if h, ok := a.hosts[name]; ok {
		return h, nil
	}
	return nil, fmt.Errorf("unknown host: %s", name)
}
