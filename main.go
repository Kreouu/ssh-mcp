package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

	server := mcp.NewServer(&mcp.Implementation{
		Name:    cfg.Server.Name,
		Version: cfg.Server.Version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ssh_list_hosts",
		Description: "List configured SSH hosts.",
	}, app.handleListHosts)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ssh_exec",
		Description: "Execute a shell command on a remote host via SSH.",
	}, app.handleExec)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ssh_upload",
		Description: "Upload a local file or directory to a remote host via SFTP.",
	}, app.handleUpload)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ssh_download",
		Description: "Download a remote file or directory via SFTP.",
	}, app.handleDownload)

	if err := server.Run(context.Background(), &AutoTransport{}); err != nil {
		logger.Println("mcp run error:", err)
		os.Exit(1)
	}
}
