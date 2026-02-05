### SSH 隧道网关

*添加于 v0.53.0*

### 概念

SSH 支持反向代理功能 [rfc](https://www.rfc-editor.org/rfc/rfc4254#page-16)。

frp 支持在 frps 端监听 SSH 端口，以使用 SSH -R 协议实现 TCP 协议代理。此模式不依赖 frpc。

SSH 反向隧道代理和通过 frp 代理 SSH 端口是两个不同的概念。SSH 反向隧道代理本质上是一种基本的反向代理，当您不想使用 frpc 时，可以通过 SSH 客户端连接到 frps 来实现。

```toml
# frps.toml
sshTunnelGateway.bindPort = 0
sshTunnelGateway.privateKeyFile = ""
sshTunnelGateway.autoGenPrivateKeyPath = ""
sshTunnelGateway.authorizedKeysFile = ""
```

| 字段 | 类型 | 描述 | 必需 |
| :--- | :--- | :--- | :--- |
| bindPort| int | frps 监听的 ssh 服务器端口。| 是 |
| privateKeyFile | string | 默认值为空。ssh 服务器使用的私钥文件。如果为空，frps 将读取 autoGenPrivateKeyPath 路径下的私钥文件。可以重用本地计算机上的 /home/user/.ssh/id_rsa 文件，或者指定自定义路径。| 否 |
| autoGenPrivateKeyPath  | string |默认值为 ./.autogen_ssh_key。如果文件不存在或其内容为空，frps 将自动生成 RSA 私钥文件内容并将其存储在此文件中。|否|
| authorizedKeysFile  | string |默认值为空。如果为空，则不进行 ssh 客户端身份验证。如果不为空，则可以实现 ssh 免密码登录身份验证。可以重用本地的 /home/user/.ssh/authorized_keys 文件，或者指定自定义路径。| 否 |

### 基本用法

#### 服务端 frps

最小配置：

```toml
sshTunnelGateway.bindPort = 2200
```

将上述配置放在 frps.toml 中并运行 `./frps -c frps.toml`。它将在端口 2200 上监听并接受 SSH 反向代理请求。

注意：

1. 使用最小配置时，将在当前工作目录中自动创建一个 `.autogen_ssh_key` 私钥文件。frps 的 SSH 服务器将使用此私钥文件进行加密和解密。或者，您可以重用本地计算机上现有的私钥文件，例如 `/home/user/.ssh/id_rsa`。

2. 在最小配置模式下运行 frps 时，通过 SSH 连接到 frps 不需要身份验证。强烈建议在 frps 中配置 token 并在 SSH 命令行中指定 token。

#### 客户端 SSH

命令格式为：

```bash
ssh -R :80:{local_ip:port} v0@{frps_address} -p {frps_ssh_listen_port} {tcp|http|https|stcp|tcpmux} --remote_port {real_remote_port} --proxy_name {proxy_name} --token {frp_token}
```

1. `--proxy_name` 是可选的，如果留空，将随机生成一个。
2. 登录到 frps 的用户名始终为 "v0"，目前没有特殊意义，即 `v0@{frps_address}`。
3. 服务端代理监听由 `--remote_port` 确定的端口。
4. `{tcp|http|https|stcp|tcpmux}` 支持完整的命令参数，可以使用 `--help` 获取。例如：`ssh -R :80::8080 v0@127.0.0.1 -p 2200 http --help`。
5. token 是可选的，但出于安全考虑，强烈建议在 frps 中配置 token。

#### TCP 代理

```bash
ssh -R :80:127.0.0.1:8080 v0@{frp_address} -p 2200 tcp --proxy_name "test-tcp" --remote_port 9090
```

这将在 frps 上设置一个代理，监听端口 9090 并代理端口 8080 上的本地服务。

```bash
frp (via SSH) (Ctrl+C to quit)

User:
ProxyName: test-tcp
Type: tcp
RemoteAddress: :9090
```

等效于：

```bash
frpc tcp --proxy_name "test-tcp" --local_ip 127.0.0.1 --local_port 8080 --remote_port 9090
```

可以通过执行 `--help` 获取更多参数。

#### HTTP 代理

```bash
ssh -R :80:127.0.0.1:8080 v0@{frp address} -p 2200 http --proxy_name "test-http"  --custom_domain test-http.frps.com
```

等效于：
```bash
frpc http --proxy_name "test-http" --custom_domain test-http.frps.com
```

您可以使用以下命令访问 HTTP 服务：

curl 'http://test-http.frps.com'

可以通过执行 --help 获取更多参数。

#### HTTPS/STCP/TCPMUX 代理

要获取使用说明，请使用以下命令：

```bash
ssh -R :80:127.0.0.1:8080 v0@{frp address} -p 2200 {https|stcp|tcpmux} --help
```

### 高级用法

#### 重用本地计算机上的 id_rsa 文件

```toml
# frps.toml
sshTunnelGateway.bindPort = 2200
sshTunnelGateway.privateKeyFile = "/home/user/.ssh/id_rsa"
```

在 SSH 协议握手期间，交换公钥以进行数据加密。因此，frps 端的 SSH 服务器需要指定一个私钥文件，该文件可以从本地计算机上的现有文件重用。如果 privateKeyFile 字段为空，frps 将自动创建一个 RSA 私钥文件。

#### 指定自动生成的私钥文件路径

```toml
# frps.toml
sshTunnelGateway.bindPort = 2200
sshTunnelGateway.autoGenPrivateKeyPath = "/var/frp/ssh-private-key-file"
```

frps 将自动创建一个私钥文件并将其存储在指定路径。

注意：更改 frps 中的私钥文件可能会导致 SSH 客户端登录失败。如果您需要成功登录，可以从 `/home/user/.ssh/known_hosts` 文件中删除旧记录。

#### 使用现有的 authorized_keys 文件进行 SSH 公钥身份验证

```toml
# frps.toml
sshTunnelGateway.bindPort = 2200
sshTunnelGateway.authorizedKeysFile = "/home/user/.ssh/authorized_keys"
```

authorizedKeysFile 是用于 SSH 公钥身份验证的文件，其中包含用户的公钥信息，每行一个密钥。

如果 authorizedKeysFile 为空，frps 将不会对 SSH 客户端进行任何身份验证。Frps 不支持 SSH 用户名和密码身份验证。

您可以重用本地计算机上现有的 `authorized_keys` 文件进行客户端身份验证。

注意：authorizedKeysFile 用于 SSH 登录阶段的用户身份验证，而 token 用于 frps 身份验证。这两种身份验证方法是独立的。SSH 身份验证在前，然后是 frps token 身份验证。强烈建议至少启用其中一种。如果 authorizedKeysFile 为空，强烈建议在 frps 中启用 token 身份验证以避免安全风险。

#### 使用自定义的 authorized_keys 文件进行 SSH 公钥身份验证

```toml
# frps.toml
sshTunnelGateway.bindPort = 2200
sshTunnelGateway.authorizedKeysFile = "/var/frps/custom_authorized_keys_file"
```

指定自定义 `authorized_keys` 文件的路径。

请注意，对 authorizedKeysFile 文件的更改可能会导致 SSH 身份验证失败。您可能需要将公钥信息重新添加到 authorizedKeysFile。
