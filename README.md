# SSH MCP Server

一个基于 `stdio` 传输的独立 MCP Server，用来让支持 MCP 的客户端通过 SSH 执行远程命令，并通过 SFTP 上传、下载文件或目录。

这个项目适合接入 Codex、Claude Desktop 或其他支持 MCP 的工具，把一组 SSH 主机以工具形式暴露出来，而不是每次都手动开终端连接。

## 功能概览

- 通过 `ssh_exec` 在远程主机执行命令
- 通过 `ssh_upload` 上传本地文件或目录到远程
- 通过 `ssh_download` 下载远程文件或目录到本地
- 通过 `ssh_list_hosts` 列出配置中的主机
- 支持密码认证和私钥认证
- 支持加密私钥口令
- 支持 `proxy_jump` / `jump_host` 跳板机链路
- 支持 `known_hosts` 校验和 `insecure_ignore` 两种主机指纹策略
- 支持命令黑名单 / 白名单策略
- 支持配置中的环境变量展开，如 `${SSH_PROD_PASSWORD}`

## 适用场景

- 让 AI 客户端安全地执行远程诊断命令
- 远程拉日志、传配置、同步部署文件
- 通过跳板机访问内网机器
- 对远程命令增加最基本的危险操作拦截

## 项目结构

当前仓库的核心文件大致如下：

- `main.go`：程序入口，注册 MCP tools
- `config.go`：配置解析、默认值填充、环境变量和路径展开
- `ssh_client.go`：SSH 连接、认证、跳板机、主机指纹校验
- `exec.go`：远程命令执行
- `transfer.go`：SFTP 上传和下载
- `policy.go`：命令策略校验
- `config.example.yaml`：配置示例

## 环境要求

- Go `1.22+`
- 目标主机已开启 SSH
- 当前机器可以访问目标主机，或可以通过跳板机访问

## 构建

```bash
go build -o ssh-mcp
```

如果你只是本地测试，也可以直接运行：

```bash
go run .
```

## 快速开始

### 1. 复制配置文件

```bash
cp config.example.yaml config.yaml
```

### 2. 按你的环境修改 `config.yaml`

至少要保证：

- `hosts` 里有一个可连通的主机
- 为该主机配置了 `password` 或 `private_key_path`
- `host_key.mode` 与你的环境匹配

### 3. 启动服务

```bash
./ssh-mcp -config ./config.yaml
```

也可以通过环境变量指定配置文件路径：

```bash
SSH_MCP_CONFIG=./config.yaml ./ssh-mcp
```

### 4. 仅检查主机配置是否被识别

```bash
./ssh-mcp -config ./config.yaml -list-hosts
```

这个命令会直接输出配置中的主机名并退出，适合排查配置是否加载成功。

## MCP 工具列表

服务启动后会注册以下工具：

### `ssh_list_hosts`

列出当前配置文件中的主机信息。

返回字段：

- `name`
- `address`
- `user`
- `port`

### `ssh_exec`

在远程主机上执行 Shell 命令。

输入字段：

- `host`：主机名，必须与配置中的 `hosts[].name` 一致
- `command`：要执行的命令
- `workdir`：可选，远程执行目录
- `env`：可选，远程环境变量
- `timeout_sec`：可选，覆盖默认超时时间

输出字段：

- `ok`：命令是否成功退出
- `host`
- `command`
- `exit_code`
- `stdout`
- `stderr`
- `truncated`：输出是否因上限而被截断
- `duration_ms`

实现细节：

- 命令通过 `bash -lc` 执行
- 如果配置了 `workdir`，会先 `cd` 到目标目录
- 如果配置了 `env`，会先导出环境变量再执行命令
- 会受到命令策略限制，命中黑名单时会直接拒绝执行

### `ssh_upload`

将本地文件或目录上传到远程主机。

输入字段：

- `host`
- `local_path`
- `remote_path`
- `overwrite`：是否覆盖已存在的远程文件
- `preserve_mode`：是否保留文件权限

输出字段：

- `host`
- `local_path`
- `remote_path`
- `files`
- `directories`
- `bytes`
- `duration_ms`

说明：

- 支持上传单个文件
- 支持递归上传目录
- 远程目录不存在时会自动创建
- 如果上传的是单文件，且 `remote_path` 指向目录或以 `/` 结尾，会自动拼接文件名

### `ssh_download`

将远程文件或目录下载到本地。

输入字段：

- `host`
- `remote_path`
- `local_path`
- `overwrite`
- `preserve_mode`

输出字段：

- `host`
- `remote_path`
- `local_path`
- `files`
- `directories`
- `bytes`
- `duration_ms`

说明：

- 支持下载单个文件
- 支持递归下载目录
- 本地目录不存在时会自动创建
- 如果下载的是单文件，且 `local_path` 是目录，会自动拼接远程文件名

## 配置文件说明

配置文件格式为 YAML，默认读取当前目录下的 `config.yaml`。也可以通过：

- 启动参数 `-config`
- 环境变量 `SSH_MCP_CONFIG`

来指定其它路径。

### 配置示例

```yaml
server:
  name: "ssh-mcp"
  version: "0.1.0"
  default_timeout_sec: 120
  connect_timeout_sec: 10
  max_output_bytes: 1048576
  allow_dangerous: false

hosts:
  - name: "aliyun"
    address: "47.110.154.57"
    port: 22
    user: "root"
    private_key_path: "~/.ssh/id_ed25519_aliyun"
    host_key:
      mode: "known_hosts"
      known_hosts_path: "~/.ssh/known_hosts"

  - name: "ubuntu_ali"
    address: "127.0.0.1"
    port: 2222
    user: "ng"
    private_key_path: "~/.ssh/id_ed25519_aliyun"
    proxy_jump: "aliyun"
    host_key:
      mode: "known_hosts"
      known_hosts_path: "~/.ssh/known_hosts"

  - name: "prod"
    address: "127.0.0.1"
    port: 22
    user: "root"
    password: "${SSH_PROD_PASSWORD}"
    host_key:
      mode: "known_hosts"
      known_hosts_path: "~/.ssh/known_hosts"

policy:
  # allow_patterns:
  #   - "^ls "
  # deny_patterns:
  #   - "(?i)rm\\s+-rf"
```

### `server` 字段

- `name`：MCP Server 名称，默认是 `ssh-mcp`
- `version`：服务版本号，默认是 `0.1.0`
- `default_timeout_sec`：远程命令默认超时秒数
- `connect_timeout_sec`：SSH 建连超时秒数
- `max_output_bytes`：命令输出上限，超出后会截断
- `allow_dangerous`：是否关闭默认危险命令拦截，默认 `false`

### `hosts` 字段

每一项代表一个可调用的远程主机。

- `name`：主机唯一标识，工具调用时使用这个名字
- `address`：主机地址或 IP
- `port`：SSH 端口，默认为 `22`
- `user`：SSH 用户名
- `password`：密码认证
- `private_key_path`：私钥文件路径
- `private_key`：直接内嵌私钥内容
- `private_key_passphrase`：加密私钥口令
- `proxy_jump`：跳板机主机名
- `jump_host`：`proxy_jump` 的别名，最终会归一化到 `proxy_jump`
- `host_key.mode`：主机指纹校验模式
- `host_key.known_hosts_path`：`known_hosts` 文件路径
- `host_key.algorithms`：可选，手动指定主机密钥算法顺序
- `insecure_ignore_host_key`：兼容字段，等价于把 `host_key.mode` 设为 `insecure_ignore`

### `policy` 字段

用于限制 `ssh_exec` 可执行的命令。

- `allow_patterns`：白名单正则，配置后只有匹配的命令允许执行
- `deny_patterns`：黑名单正则，匹配则拒绝执行

如果 `allow_dangerous: false`，程序会默认附带一组危险命令拦截规则，例如：

- `rm -rf /`
- `rm -rf *`
- `mkfs`
- `dd of=/dev/...`
- `shutdown`
- `reboot`
- fork bomb

这不是完整安全沙箱，只是最基础的一层防护。

## 认证方式

### 1. 私钥认证

推荐方式：

```yaml
hosts:
  - name: "prod"
    address: "1.2.3.4"
    user: "root"
    private_key_path: "~/.ssh/id_ed25519"
    host_key:
      mode: "known_hosts"
```

如果私钥有口令：

```yaml
private_key_passphrase: "${SSH_KEY_PASSPHRASE}"
```

### 2. 密码认证

```yaml
hosts:
  - name: "prod"
    address: "1.2.3.4"
    user: "root"
    password: "${SSH_PROD_PASSWORD}"
    host_key:
      mode: "known_hosts"
```

### 3. 同时配置私钥和密码

程序会把可用认证方式一起加入 SSH Auth 列表，由服务端协商使用。

## 主机指纹校验

### `known_hosts` 模式

推荐在正式环境使用：

```yaml
host_key:
  mode: "known_hosts"
  known_hosts_path: "~/.ssh/known_hosts"
```

特点：

- 会校验远程主机公钥
- 更安全
- 如果第一次连某主机，通常要先手动用 `ssh` 连一次，让主机指纹写入 `known_hosts`

### `insecure_ignore` 模式

```yaml
host_key:
  mode: "insecure_ignore"
```

特点：

- 不校验主机指纹
- 适合临时测试
- 不建议用于生产环境

### 关于 `host_key.algorithms`

有些服务器只在特定 host key 算法下握手稳定，比如只接受 `ssh-ed25519`。这时可以显式配置：

```yaml
host_key:
  mode: "known_hosts"
  known_hosts_path: "~/.ssh/known_hosts"
  algorithms: ["ssh-ed25519"]
```

如果你不手动配置，程序在 `known_hosts` 模式下会尝试从 `known_hosts` 中推断该主机的算法顺序。

## 跳板机配置

如果一台机器不能直接访问，需要通过另一台 SSH 主机中转，可以这样写：

```yaml
hosts:
  - name: "bastion"
    address: "1.2.3.4"
    user: "root"
    private_key_path: "~/.ssh/id_ed25519"
    host_key:
      mode: "known_hosts"

  - name: "internal-app"
    address: "10.0.0.8"
    port: 22
    user: "ubuntu"
    private_key_path: "~/.ssh/id_ed25519"
    proxy_jump: "bastion"
    host_key:
      mode: "known_hosts"
```

调用 `internal-app` 时，程序会先连接 `bastion`，再从跳板机拨到目标主机。

程序内部还会检查跳板链路是否出现循环引用。

## 环境变量与路径展开

配置文件支持环境变量展开，例如：

```yaml
password: "${SSH_PROD_PASSWORD}"
private_key_passphrase: "${SSH_KEY_PASSPHRASE}"
```

也支持 `~` 展开，例如：

```yaml
private_key_path: "~/.ssh/id_ed25519"
known_hosts_path: "~/.ssh/known_hosts"
```

## 接入 Codex 的示例

可以在 Codex 的 MCP 配置中加入：

```toml
[mcp_servers.ssh]
command = "/Users/Ng/Documents/ssh-mcp/ssh-mcp"
args = ["-config", "/Users/Ng/Documents/ssh-mcp/config.yaml"]
```

如果你已经设置了环境变量 `SSH_MCP_CONFIG`，也可以只写：

```toml
[mcp_servers.ssh]
command = "/Users/Ng/Documents/ssh-mcp/ssh-mcp"
```

## 使用示例

### 示例 1：列出主机

调用：

```json
{
  "tool": "ssh_list_hosts"
}
```

### 示例 2：执行命令

调用：

```json
{
  "tool": "ssh_exec",
  "host": "prod",
  "command": "uname -a && df -h",
  "timeout_sec": 30
}
```

### 示例 3：指定工作目录和环境变量

```json
{
  "tool": "ssh_exec",
  "host": "prod",
  "workdir": "/srv/app",
  "env": {
    "APP_ENV": "prod"
  },
  "command": "go test ./..."
}
```

### 示例 4：上传文件

```json
{
  "tool": "ssh_upload",
  "host": "prod",
  "local_path": "./dist/app.tar.gz",
  "remote_path": "/tmp/",
  "overwrite": true
}
```

### 示例 5：下载目录

```json
{
  "tool": "ssh_download",
  "host": "prod",
  "remote_path": "/var/log/myapp",
  "local_path": "./downloads/logs",
  "overwrite": true
}
```

## 安全说明

这个项目做的是“带基础约束的远程执行”，不是完全隔离的安全执行环境，因此有几个边界要明确：

- `ssh_exec` 本质上仍然是在目标机器上执行 Shell 命令
- 默认危险命令拦截只能挡住一部分明显高风险操作
- 如果把 `allow_dangerous` 设为 `true`，默认黑名单会被关闭
- `insecure_ignore` 会跳过主机指纹校验，存在中间人风险
- 如果 MCP 客户端本身权限过大，这个服务就等于把远程 SSH 能力暴露给它

更稳妥的实践建议：

- 优先使用私钥而不是密码
- 优先使用 `known_hosts` 校验
- 给远程账号最小权限
- 在 `policy.allow_patterns` 中只放行业务允许命令
- 不要直接把 root 主机暴露给不受控的自动化流程

## 常见问题

### 1. 提示 `no hosts configured`

说明配置文件里没有可用的 `hosts`，或者配置文件没有被正确读取。先检查：

- `-config` 路径是否正确
- `SSH_MCP_CONFIG` 是否指向正确文件
- YAML 缩进是否正确

### 2. 提示 `no auth methods for host`

说明该主机没有配置任何可用认证方式。至少需要以下之一：

- `password`
- `private_key`
- `private_key_path`

### 3. 提示 `load known_hosts`

通常是 `known_hosts` 文件不存在或路径写错。你可以：

- 检查 `host_key.known_hosts_path`
- 先手动执行一次 `ssh user@host`
- 临时改成 `insecure_ignore` 做排查

### 4. 命令被策略拦截

如果返回类似 “command blocked by denylist policy” 或 “command blocked by allowlist policy”，说明命中了策略规则。需要检查：

- `policy.allow_patterns`
- `policy.deny_patterns`
- `server.allow_dangerous`

### 5. 输出不完整

如果返回里 `truncated: true`，说明命令输出超过了 `server.max_output_bytes`。可以：

- 提高 `max_output_bytes`
- 改为输出摘要
- 先在远程写文件，再分段下载

### 6. 跳板机连接失败

检查：

- `proxy_jump` 指向的主机名是否存在
- 跳板机本身是否可连
- 跳板机是否能访问目标主机地址和端口
- 跳板链路里是否出现循环引用

## 开发说明

本项目目前是一个单二进制程序，启动后通过 `stdio` 提供 MCP 服务。核心流程如下：

1. 读取 YAML 配置
2. 归一化默认值、环境变量和路径
3. 构建命令策略
4. 初始化主机映射
5. 注册 MCP tools
6. 通过 `stdio` 运行 MCP Server

如果你后面要扩展功能，比较自然的方向包括：

- 增加端口转发类工具
- 增加远程文件内容读写工具
- 增加更细粒度的命令审计日志
- 增加更严格的命令模板限制

## License

本项目采用 [MIT License](./LICENSE)。
