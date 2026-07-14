# ssh-mcp

让 Codex、Claude Code 等 MCP 客户端通过 SSH 执行远程命令和传输文件。支持私钥、密码、`known_hosts` 校验和跳板机。

## 安装

推荐把下面这段话交给 Agent：

```text
Install and configure the latest ssh-mcp release by following this guide:
https://github.com/Kreouu/ssh-mcp/blob/main/docs/agent-install.md
```

Agent 会自动选择系统版本、寻找现有 SSH 配置并注册 MCP。也可以直接从 [GitHub Releases](https://github.com/Kreouu/ssh-mcp/releases) 下载预编译文件或离线包。

## 使用

不需要记工具名，直接告诉 Agent：

```text
列出我配置的 SSH 主机。

在 pi 上运行 uname -a，并告诉我结果。

把本地文件上传到 pi 的 /home/pi/workspace。
```

## 手动使用

需要 Go 1.24.4 或更高版本：

```bash
go build -o ssh-mcp .
cp config.example.yaml config.yaml
./ssh-mcp -config ./config.yaml -list-hosts
codex mcp add ssh -- /absolute/path/ssh-mcp -config /absolute/path/config.yaml
```

修改 `config.yaml`，至少填写一个主机的名称、地址、用户和认证方式。完整配置见 [`config.example.yaml`](config.example.yaml)。

## 安全

- 优先使用私钥和 `known_hosts` 校验。
- 不要把密码或私钥内容提交到仓库。
- 远程账号应只拥有完成任务所需的权限。
- `ssh_exec` 会执行真实 Shell 命令，本项目不是安全沙箱。

## License

MIT
