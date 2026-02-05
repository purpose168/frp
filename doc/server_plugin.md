### 服务器插件

frp 服务器插件旨在在不修改 Go 代码的情况下扩展 frp 的功能。

外部服务器应在不同的进程中运行，接收来自 frps 的 RPC 调用。
在 frps 执行某些操作之前，它会发送 RPC 请求通知外部 RPC 服务器，并根据其响应采取行动。

### RPC 请求

RPC 请求基于 HTTP 上的 JSON。

当服务器插件接受操作请求时，可以返回三种不同的响应：

* 拒绝操作并返回原因。
* 允许操作并保持原始内容。
* 允许操作并返回修改后的内容。

### 接口

可以在 frps 中为每个管理插件配置 HTTP 路径。在此示例中，我们假设它是 `/handler`。

对 RPC 服务器的请求如下所示：

```
POST /handler?version=0.1.0&op=Login
{
    "version": "0.1.0",
    "op": "Login",
    "content": {
        ... // 操作信息
    }
}

请求头：
X-Frp-Reqid: 用于追踪
```

响应可以如下所示：

* 非 200 HTTP 响应状态码（这将自动告诉 frps 请求应该失败）

* 拒绝操作：

```
{
    "reject": true,
    "reject_reason": "无效的用户"
}
```

* 允许操作并保持原始内容：

```
{
    "reject": false,
    "unchange": true
}
```

* 允许操作并修改内容

```
{
    "unchange": "false",
    "content": {
        ... // 替换的内容
    }
}
```

### 操作

目前支持 `Login`（登录）、`NewProxy`（新建代理）、`CloseProxy`（关闭代理）、`Ping`（心跳）、`NewWorkConn`（新建工作连接）和 `NewUserConn`（新建用户连接）操作。

#### Login

客户端登录操作

```
{
    "content": {
        "version": <string>,
        "hostname": <string>,
        "os": <string>,
        "arch": <string>,
        "user": <string>,
        "timestamp": <int64>,
        "privilege_key": <string>,
        "run_id": <string>,
        "pool_count": <int>,
        "metas": map<string>string,
        "client_address": <string>
    }
}
```

#### NewProxy

创建新代理

```
{
    "content": {
        "user": {
            "user": <string>,
            "metas": map<string>string
            "run_id": <string>
        },
        "proxy_name": <string>,
        "proxy_type": <string>,
        "use_encryption": <bool>,
        "use_compression": <bool>,
        "bandwidth_limit": <string>,
        "bandwidth_limit_mode": <string>,
        "group": <string>,
        "group_key": <string>,

        // 仅适用于 tcp 和 udp
        "remote_port": <int>,

        // 仅适用于 http 和 https
        "custom_domains": []<string>,
        "subdomain": <string>,
        "locations": []<string>,
        "http_user": <string>,
        "http_pwd": <string>,
        "host_header_rewrite": <string>,
        "headers": map<string>string,

        // 仅适用于 stcp
        "sk": <string>,

        // 仅适用于 tcpmux
        "multiplexer": <string>

        "metas": map<string>string
    }
}
```

#### CloseProxy

关闭先前创建的代理。

请注意，对于每个关闭的代理都会发送一个请求，如果单个客户端绑定了太多代理，请**不要**使用此功能，因为这可能会耗尽服务器的资源。

```
{
    "content": {
        "user": {
            "user": <string>,
            "metas": map<string>string
            "run_id": <string>
        },
        "proxy_name": <string>
    }
}
```

#### Ping

来自 frpc 的心跳

```
{
    "content": {
        "user": {
            "user": <string>,
            "metas": map<string>string
            "run_id": <string>
        },
        "timestamp": <int64>,
        "privilege_key": <string>
    }
}
```

#### NewWorkConn

从 frpc 接收到新的工作连接（在 `run_id` 与现有 frp 连接匹配后发送 RPC）

```
{
    "content": {
        "user": {
            "user": <string>,
            "metas": map<string>string
            "run_id": <string>
        },
        "run_id": <string>
        "timestamp": <int64>,
        "privilege_key": <string>
    }
}
```

#### NewUserConn

从代理接收到新的用户连接（支持 `tcp`、`stcp`、`https` 和 `tcpmux`）。

```
{
    "content": {
        "user": {
            "user": <string>,
            "metas": map<string>string
            "run_id": <string>
        },
        "proxy_name": <string>,
        "proxy_type": <string>,
        "remote_addr": <string>
    }
}
```

### 服务器插件配置

```toml
# frps.toml
bindPort = 7000

[[httpPlugins]]
name = "user-manager"
addr = "127.0.0.1:9000"
path = "/handler"
ops = ["Login"]

[[httpPlugins]]
name = "port-manager"
addr = "127.0.0.1:9001"
path = "/handler"
ops = ["NewProxy"]
```

- addr：外部 RPC 服务监听的地址。默认为 http。对于 https，请指定协议：`addr = "https://127.0.0.1:9001"`。
- path：POST 请求的 HTTP 请求 URL 路径。
- ops：插件需要处理的操作（例如 "Login"、"NewProxy" 等）。
- tlsVerify：当协议为 https 时，默认情况下我们会验证。如果要跳过验证，请将此值设置为 false。

### 元数据

元数据将在每个 RPC 请求中发送到服务器插件。

有两种类型的元数据条目 - 全局元数据和每个代理配置下的元数据。
全局元数据条目将在 `Login` 中以 `metas` 键发送，在其他任何 RPC 请求中以 `user.metas` 发送。
每个代理配置下的元数据条目仅在 `NewProxy` 操作中以 `metas` 发送。

这是元数据条目的示例：

```toml
# frpc.toml
serverAddr = "127.0.0.1"
serverPort = 7000
user = "fake"
metadatas.token = "fake"
metadatas.version = "1.0.0"

[[proxies]]
name = "ssh"
type = "tcp"
localPort = 22
remotePort = 6000
metadatas.id = "123"
```
