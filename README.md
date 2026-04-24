# SSH MCP Server (stdio)

A standalone MCP server that executes SSH commands and transfers files (upload/download) over SFTP.
Designed for Codex or any MCP-compatible client using stdio transport.

## Requirements

- Go 1.22+ (tested with newer Go versions)
- SSH access to your target hosts

## Build

```bash
go build -o ssh-mcp
```

## Configuration

Copy and edit the example:

```bash
cp config.example.yaml config.yaml
```

Key fields:

- `hosts[].name` is used by tools (e.g. `host: "prod"`).
- Use `password` or `private_key_path` (or both). Private key is recommended.
- `host_key.mode` defaults to `known_hosts`. For first-time hosts, connect with `ssh` once or use `insecure_ignore`.
- `host_key.algorithms` can be used to pin the host key algorithm order when a server only succeeds with a specific key type such as `ssh-ed25519`.
- `policy` can be used to allow/deny commands (default deny list includes `rm -rf /`, `mkfs`, `dd of=/dev/...`).
- `proxy_jump` / `jump_host` enables jump-host chaining between hosts defined in the same config.

Environment variables in the config are expanded (e.g. `${SSH_PROD_PASSWORD}`).

## Run

```bash
./ssh-mcp -config ./config.yaml
```

Or set `SSH_MCP_CONFIG` and run without flags:

```bash
SSH_MCP_CONFIG=./config.yaml ./ssh-mcp
```

## Codex stdio config (example)

```toml
[mcp_servers.ssh]
command = "/Users/hanhan/Desktop/code/ssh-mcp/ssh-mcp"
args = ["-config", "/Users/hanhan/Desktop/code/ssh-mcp/config.yaml"]
```

## Available tools

- `ssh_list_hosts`: list configured hosts
- `ssh_exec`: execute a shell command on a host
- `ssh_upload`: upload local files/directories to remote
- `ssh_download`: download remote files/directories to local

## Notes

- This server is non-interactive: commands run via `bash -lc`.
- Use `allow_dangerous: true` in config to disable default deny rules.
