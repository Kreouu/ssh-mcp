package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var buildVersion string

func main() {
	var configPath string
	var listHosts bool
	flag.StringVar(&configPath, "config", "", "Path to config file (or set SSH_MCP_CONFIG)")
	flag.BoolVar(&listHosts, "list-hosts", false, "List configured host names and exit")
	flag.Parse()

	cfgPath := resolveConfigPath(configPath)
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}
	cfg.Server.Version = resolveVersion(cfg.Server.Version, buildVersion)

	logger := log.New(os.Stderr, "ssh-mcp: ", log.LstdFlags)
	policy, err := NewCommandPolicy(cfg.Policy, cfg.Server.AllowDangerous)
	if err != nil {
		logger.Println("policy error:", err)
		os.Exit(1)
	}

	app, err := NewApp(cfg, policy, logger)
	if err != nil {
		logger.Println("app error:", err)
		os.Exit(1)
	}

	if listHosts {
		for name := range app.hosts {
			fmt.Println(name)
		}
		return
	}

	server := newMCPServer(cfg, app)

	if err := server.Run(context.Background(), &AutoTransport{}); err != nil {
		logger.Println("mcp run error:", err)
		os.Exit(1)
	}
}

func resolveVersion(configured, built string) string {
	if built != "" {
		return strings.TrimPrefix(built, "v")
	}
	return configured
}

func newMCPServer(cfg *Config, app *App) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    cfg.Server.Name,
		Version: cfg.Server.Version,
	}, nil)
	app.addTools(server)
	return server
}
