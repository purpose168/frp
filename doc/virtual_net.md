# 虚拟网络 (VirtualNet)

*添加于 v0.62.0 的 Alpha 功能*

VirtualNet 功能使 frp 能够通过 TUN 接口在客户端和访问者之间创建和管理虚拟网络连接。这允许在机器之间进行 IP 级别的路由，将 frp 从简单的端口转发扩展到支持完整的网络连接。

> **注意**：VirtualNet 是一个 Alpha 阶段的功能，目前不稳定。其配置方法和功能可能会在后续版本中随时调整和更改。不要在生产环境中使用此功能；仅建议用于测试和评估目的。

## 启用 VirtualNet

由于 VirtualNet 目前是 Alpha 功能，您需要在配置中使用功能门来启用它：

```toml
# frpc.toml
featureGates = { VirtualNet = true }
```

## 基本配置

要使用虚拟网络功能：

1. 首先，为您的 frpc 配置虚拟网络地址：

```toml
# frpc.toml
serverAddr = "x.x.x.x"
serverPort = 7000
featureGates = { VirtualNet = true }

# 配置虚拟网络接口
virtualNet.address = "100.86.0.1/24"
```

2. 对于客户端代理，使用 `virtual_net` 插件：

```toml
# frpc.toml (服务端)
[[proxies]]
name = "vnet-server"
type = "stcp"
secretKey = "your-secret-key"
[proxies.plugin]
type = "virtual_net"
```

3. 对于访问者连接，配置 `virtual_net` 访问者插件：

```toml
# frpc.toml (客户端)
serverAddr = "x.x.x.x"
serverPort = 7000
featureGates = { VirtualNet = true }

# 配置虚拟网络接口
virtualNet.address = "100.86.0.2/24"

[[visitors]]
name = "vnet-visitor"
type = "stcp"
serverName = "vnet-server"
secretKey = "your-secret-key"
bindPort = -1
[visitors.plugin]
type = "virtual_net"
destinationIP = "100.86.0.1"
```

## 要求和限制

- **权限**：创建 TUN 接口需要提升的权限（root/admin）
- **平台支持**：目前在 Linux 和 macOS 上支持
- **默认状态**：作为 Alpha 功能，VirtualNet 默认处于禁用状态
- **配置**：必须为虚拟网络中的每个端点提供有效的 IP/CIDR
