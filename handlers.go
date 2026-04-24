package main

import (
	"context"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type HostInfo struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	User    string `json:"user"`
	Port    int    `json:"port"`
}

type ListHostsOutput struct {
	Hosts []HostInfo `json:"hosts"`
}

func (a *App) handleListHosts(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, ListHostsOutput, error) {
	names := make([]string, 0, len(a.hosts))
	for name := range a.hosts {
		names = append(names, name)
	}
	sort.Strings(names)

	out := ListHostsOutput{Hosts: make([]HostInfo, 0, len(names))}
	for _, name := range names {
		h := a.hosts[name]
		out.Hosts = append(out.Hosts, HostInfo{
			Name:    h.Name,
			Address: h.Address,
			User:    h.User,
			Port:    h.Port,
		})
	}
	return nil, out, nil
}

func (a *App) handleExec(ctx context.Context, req *mcp.CallToolRequest, input ExecInput) (*mcp.CallToolResult, ExecOutput, error) {
	host, err := a.getHost(input.Host)
	if err != nil {
		return nil, ExecOutput{}, err
	}
	out, err := a.runCommand(ctx, host, input)
	if err != nil {
		return nil, ExecOutput{}, err
	}
	return nil, out, nil
}

func (a *App) handleUpload(ctx context.Context, req *mcp.CallToolRequest, input UploadInput) (*mcp.CallToolResult, UploadOutput, error) {
	host, err := a.getHost(input.Host)
	if err != nil {
		return nil, UploadOutput{}, err
	}
	out, err := a.upload(ctx, host, input)
	if err != nil {
		return nil, UploadOutput{}, err
	}
	return nil, out, nil
}

func (a *App) handleDownload(ctx context.Context, req *mcp.CallToolRequest, input DownloadInput) (*mcp.CallToolResult, DownloadOutput, error) {
	host, err := a.getHost(input.Host)
	if err != nil {
		return nil, DownloadOutput{}, err
	}
	out, err := a.download(ctx, host, input)
	if err != nil {
		return nil, DownloadOutput{}, err
	}
	return nil, out, nil
}
