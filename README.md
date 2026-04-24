# SSH MCP Server

一个通过 `stdio` 提供 MCP 能力的 SSH 工具服务，支持远程执行命令，以及通过 SFTP 上传、下载文件。

适合接入 Codex、Claude Desktop 或其他 MCP 客户端，把 SSH 主机作为工具暴露出来使用。

## 功能

- `ssh_list_hosts`：列出配置中的主机
- `ssh_exec`：远程执行命令
- `ssh_upload`：上传本地文件或目录
- `ssh_download`：下载远程文件或目录
- 支持密码、私钥、跳板机、`known_hosts` 校验
- 支持基础命令黑名单 / 白名单策略

## 环境要求

- Go `1.22+`
- 目标主机已开启 SSH

## 构建

```bash
go build -o ssh-mcp
```

## 快速开始

1. 复制配置：

```bash
cp config.example.yaml config.yaml
```

2. 修改 `config.yaml`

至少配置一个可连接主机，并设置：

- `name`
- `address`
- `user`
- `password` 或 `private_key_path`
- `host_key.mode`

3. 启动服务：

```bash
./ssh-mcp -config ./config.yaml
```

也可以：

```bash
SSH_MCP_CONFIG=./config.yaml ./ssh-mcp
```

4. 检查主机是否加载成功：

```bash
./ssh-mcp -config ./config.yaml -list-hosts
```

## 配置示例

```yaml
server:
  name: "ssh-mcp"
  version: "0.1.0"
  default_timeout_sec: 120
  connect_timeout_sec: 10
  max_output_bytes: 1048576
  allow_dangerous: false

hosts:
  - name: "prod"
    address: "1.2.3.4"
    port: 22
    user: "root"
    private_key_path: "~/.ssh/id_ed25519"
    host_key:
      mode: "known_hosts"
      known_hosts_path: "~/.ssh/known_hosts"

policy:
  # allow_patterns:
  #   - "^ls "
  # deny_patterns:
  #   - "(?i)rm\\s+-rf"
```

配置要点：

- `hosts[].name` 是工具调用时使用的主机名
- 支持 `${ENV_NAME}` 环境变量展开
- `proxy_jump` / `jump_host` 可用于跳板机
- `host_key.mode` 支持 `known_hosts` 和 `insecure_ignore`
- `allow_dangerous: false` 时会启用默认危险命令拦截

## 工具参数

### `ssh_exec`

- `host`：主机名
- `command`：要执行的命令
- `workdir`：可选，远程工作目录
- `env`：可选，环境变量
- `timeout_sec`：可选，超时秒数

命令通过 `bash -lc` 执行。

### `ssh_upload`

- `host`
- `local_path`
- `remote_path`
- `overwrite`
- `preserve_mode`

### `ssh_download`

- `host`
- `remote_path`
- `local_path`
- `overwrite`
- `preserve_mode`

## Codex 配置示例

```toml
[mcp_servers.ssh]
command = "/Users/Ng/Documents/ssh-mcp/ssh-mcp"
args = ["-config", "/Users/Ng/Documents/ssh-mcp/config.yaml"]
```

## 安全说明

这个项目不是沙箱。`ssh_exec` 本质上仍然是在远程机器上执行 Shell 命令，建议：

- 优先使用私钥认证
- 优先使用 `known_hosts`
- 给远程账号最小权限
- 按需配置 `policy.allow_patterns`

## License

本项目采用 [MIT License](./LICENSE)。
