# Agent installation guide

Install and configure `ssh-mcp` with minimal user interaction.

## Rules

- Detect the operating system, CPU architecture, home directory, and MCP client configuration instead of asking the user.
- Look for a matching entry in `~/.ssh/config` first. Never read or print private-key contents.
- Ask only for missing connection details, one short question at a time: host name/address, SSH user, and authentication method.
- Prefer an existing SSH private-key path. Do not place passwords directly in chat, logs, or a public file.
- Back up an existing MCP client configuration before editing it.
- Do not connect, run a command, upload, or download until the user agrees to the verification step.

## Install

1. Detect the OS and architecture.
2. Download the matching archive from the latest GitHub Release and verify it with `checksums.txt`.
3. Install the binary and create a private configuration directory:
   - macOS/Linux: `~/.local/bin/ssh-mcp` and `~/.config/ssh-mcp/config.yaml`
   - Windows: `%LOCALAPPDATA%\ssh-mcp\ssh-mcp.exe` and `%APPDATA%\ssh-mcp\config.yaml`
4. Inspect `~/.ssh/config` and common key paths. Briefly tell the user which existing SSH profile can be reused, without exposing secrets.
5. If no usable profile exists, ask only for the missing values and create `config.yaml` from `config.example.yaml`.
6. Use `known_hosts` verification by default. Use `insecure_ignore` only after explaining the risk and receiving explicit approval.
7. Register the server as a local stdio MCP using the absolute binary and config paths.
8. Run `ssh-mcp -config <path> -list-hosts` without opening a network connection.
9. Ask whether the user wants a read-only connection test. If approved, run `ssh_exec` with `pwd` on the chosen host.

If GitHub is unavailable, ask the user to provide the complete offline release bundle. Do not silently switch to an untrusted mirror.

## Final response

Keep the result short:

```text
ssh-mcp installed.
Host: <configured host name>
Authentication: <private key or password environment variable>

Try: "列出 SSH 主机，然后查看 <host> 的系统信息。"
```

Mention restart only if the MCP client actually requires it. Keep advanced details out of the user-facing response.
