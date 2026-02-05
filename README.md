# frp

frp是一个开源项目，其持续开发完全依赖于我们优秀赞助商的支持。如果您想加入他们，请考虑[赞助frp的开发](https://github.com/sponsors/fatedier)。

<h3 align="center">金牌赞助商</h3>
<!--gold sponsors start-->
<p align="center">
  <a href="https://requestly.com/?utm_source=github&utm_medium=partnered&utm_campaign=frp" target="_blank">
    <img width="480px" src="https://github.com/user-attachments/assets/24670320-997d-4d62-9bca-955c59fe883d">
    <br>
    <b>Requestly - Postman的免费开源替代方案</b>
    <br>
    <sub>一站式API测试、模拟和拦截平台。</sub>
  </a>
</p>

<p align="center">
  <a href="https://jb.gg/frp" target="_blank">
    <img width="420px" src="https://raw.githubusercontent.com/purpose168/frp/dev/doc/pic/sponsor_jetbrains.jpg">
	<br>
	<b>专为专业Go开发者打造的完整IDE</b>
  </a>
</p>

<p align="center">
  <a href="https://github.com/beclab/Olares" target="_blank">
    <img width="420px" src="https://raw.githubusercontent.com/purpose168/frp/dev/doc/pic/sponsor_olares.jpeg">
	<br>
	<b>由您掌控的主权云</b>
	<br>
	<sub>开源、自托管的公共云替代方案，为数据所有权和隐私而构建</sub>
  </a>
</p>
<div align="center">

## Recall.ai - 会议录制API

如果您正在寻找会议录制API，请考虑查看[Recall.ai](https://www.recall.ai/?utm_source=github&utm_medium=sponsorship&utm_campaign=fatedier-frp)，

一个可以录制Zoom、Google Meet、Microsoft Teams、面对面会议等的API。

</div>
<!--gold sponsors end-->

## 什么是frp？

frp是一个快速的反向代理，允许您将位于NAT或防火墙后面的本地服务器暴露到互联网。它目前支持**TCP**和**UDP**，以及**HTTP**和**HTTPS**协议，允许通过域名将请求转发到内部服务。

frp还提供P2P连接模式。

## 目录

<!-- vim-markdown-toc GFM -->

* [开发状态](#development-status)
    * [关于V2版本](#about-v2)
* [架构](#architecture)
* [使用示例](#example-usage)
    * [通过SSH访问局域网中的计算机](#access-your-computer-in-a-lan-network-via-ssh)
    * [多个SSH服务共享同一个端口](#multiple-ssh-services-sharing-the-same-port)
    * [使用自定义域名访问局域网内的Web服务](#accessing-internal-web-services-with-custom-domains-in-lan)
    * [转发DNS查询请求](#forward-dns-query-requests)
    * [转发Unix域套接字](#forward-unix-domain-socket)
    * [暴露简单的HTTP文件服务器](#expose-a-simple-http-file-server)
    * [为本地HTTP(S)服务启用HTTPS](#enable-https-for-a-local-https-service)
    * [私密暴露您的服务](#expose-your-service-privately)
    * [P2P模式](#p2p-mode)
* [特性](#features)
    * [配置文件](#configuration-files)
    * [使用环境变量](#using-environment-variables)
    * [将配置拆分到不同文件](#split-configures-into-different-files)
    * [服务器仪表盘](#server-dashboard)
    * [客户端管理界面](#client-admin-ui)
    * [监控](#monitor)
        * [Prometheus](#prometheus)
    * [客户端认证](#authenticating-the-client)
        * [Token认证](#token-authentication)
        * [OIDC认证](#oidc-authentication)
    * [加密和压缩](#encryption-and-compression)
        * [TLS](#tls)
    * [热重载frpc配置](#hot-reloading-frpc-configuration)
    * [从客户端获取代理状态](#get-proxy-status-from-client)
    * [仅允许服务器上的特定端口](#only-allowing-certain-ports-on-the-server)
    * [端口复用](#port-reuse)
    * [带宽限制](#bandwidth-limit)
        * [每个代理](#for-each-proxy)
    * [TCP流多路复用](#tcp-stream-multiplexing)
    * [支持KCP协议](#support-kcp-protocol)
    * [支持QUIC协议](#support-quic-protocol)
    * [连接池](#connection-pooling)
    * [负载均衡](#load-balancing)
    * [服务健康检查](#service-health-check)
    * [重写HTTP Host头](#rewriting-the-http-host-header)
    * [设置其他HTTP头](#setting-other-http-headers)
    * [获取真实IP](#get-real-ip)
        * [HTTP X-Forwarded-For](#http-x-forwarded-for)
        * [Proxy Protocol](#proxy-protocol)
    * [为Web服务要求HTTP基本认证（密码）](#require-http-basic-auth-password-for-web-services)
    * [自定义子域名](#custom-subdomain-names)
    * [URL路由](#url-routing)
    * [TCP端口多路复用](#tcp-port-multiplexing)
    * [通过PROXY连接到frps](#connecting-to-frps-via-proxy)
    * [端口范围映射](#port-range-mapping)
    * [客户端插件](#client-plugins)
    * [服务器管理插件](#server-manage-plugins)
    * [SSH隧道网关](#ssh-tunnel-gateway)
    * [虚拟网络（VirtualNet）](#virtual-network-virtualnet)
* [特性开关](#feature-gates)
    * [可用的特性开关](#available-feature-gates)
    * [启用特性开关](#enabling-feature-gates)
    * [特性生命周期](#feature-lifecycle)
* [相关项目](#related-projects)
* [贡献](#contributing)
* [捐赠](#donation)
    * [GitHub赞助商](#github-sponsors)
    * [PayPal](#paypal)

<!-- vim-markdown-toc -->

## 开发状态

frp目前正在开发中。您可以在`master`分支中尝试最新的发布版本，或使用`dev`分支访问当前正在开发的版本。

我们目前正在开发版本2，并尝试进行一些代码重构和改进。但请注意，它将与版本1不兼容。

我们会在适当的时候从版本0过渡到版本1，并且只接受错误修复和改进，而不是大型功能请求。

### 关于V2版本

v2版本的复杂性和难度远远超出预期。我只能在碎片时间里进行开发，频繁的中断严重影响了生产力。鉴于这种情况，我们将继续优化和迭代当前版本，直到有更多空闲时间来进行重大版本的彻底改造。

v2的概念基于我多年在云原生领域的经验和思考，特别是在K8s和ServiceMesh方面。其核心是一个现代化的四层和七层代理，类似于envoy。这个代理本身具有高度可扩展性，不仅能够实现内网穿透的功能，还适用于各种其他领域。基于这个高度可扩展的核心，我们的目标是实现frp v1的所有功能，同时解决以前无法实现或难以以优雅方式实现的功能。此外，我们将保持高效的开发和迭代能力。

另外，我设想frp本身成为一个高度可扩展的系统和平台，类似于我们可以基于K8s提供一系列扩展能力。在K8s中，我们可以根据企业需求进行自定义开发，利用CRD、控制器模式、webhook、CSI和CNI等特性。在frp v1中，我们引入了服务器插件的概念，实现了一些基本的可扩展性。但它依赖于简单的HTTP协议，要求用户启动独立的进程并自行管理。这种方法远非灵活和方便，现实世界的需求差异很大。期望一个由少数人维护的非营利开源项目满足每个人的需求是不现实的。

最后，我们承认当前配置管理、权限验证、证书管理和API管理等模块的设计不够现代化。虽然我们可能会在v1版本中进行一些优化，但确保兼容性仍然是一个具有挑战性的问题，需要大量的努力来解决。

我们真诚地感谢您对frp的支持。

## 架构

![architecture](/doc/pic/architecture.png)

## 使用示例

首先，从[Release](https://github.com/purpose168/frp/releases)页面下载适合您操作系统和架构的最新程序。

接下来，将`frps`二进制文件和服务器配置文件放在具有公网IP地址的服务器A上。

最后，将`frpc`二进制文件和客户端配置文件放在位于无法从公共互联网直接访问的局域网中的服务器B上。

一些防病毒软件会错误地将frpc标记为恶意软件并删除它。这是因为frp是一个能够创建反向代理的网络工具。防病毒软件有时会标记反向代理，因为它们能够绕过防火墙端口限制。如果您使用防病毒软件，则可能需要在防病毒设置中将frpc添加到白名单/排除项中，以避免意外隔离/删除。有关更多详细信息，请参阅[issue 3637](https://github.com/purpose168/frp/issues/3637)。

### 通过SSH访问局域网中的计算机

1. 修改服务器A上的`frps.toml`，设置frp客户端连接的`bindPort`：

  ```toml
  # frps.toml
  bindPort = 7000
  ```

2. 在服务器A上启动`frps`：

  `./frps -c ./frps.toml`

3. 修改服务器B上的`frpc.toml`，将`serverAddr`字段设置为frps服务器的公网IP地址：

  ```toml
  # frpc.toml
  serverAddr = "x.x.x.x"
  serverPort = 7000

  [[proxies]]
  name = "ssh"
  type = "tcp"
  localIP = "127.0.0.1"
  localPort = 22
  remotePort = 6000
  ```

请注意，`localPort`（在客户端上监听）和`remotePort`（在服务器上暴露）用于frp系统的流量进出，而`serverPort`用于frps和frpc之间的通信。

4. 在服务器B上启动`frpc`：

  `./frpc -c ./frpc.toml`

5. 要通过服务器A从另一台机器访问服务器B（假设用户名为`test`），使用以下命令：

  `ssh -oPort=6000 test@x.x.x.x`

### 多个SSH服务共享同一个端口

此示例实现了使用tcpmux类型的代理通过同一端口暴露多个SSH服务。同样，只要客户端支持HTTP Connect代理连接方法，就可以通过这种方式实现端口复用。

1. 在具有公网IP的机器上部署frps并修改frps.toml文件。以下是简化配置：

  ```toml
  bindPort = 7000
  tcpmuxHTTPConnectPort = 5002
  ```

2. 在内部机器A上部署frpc，配置如下：

  ```toml
  serverAddr = "x.x.x.x"
  serverPort = 7000

  [[proxies]]
  name = "ssh1"
  type = "tcpmux"
  multiplexer = "httpconnect"
  customDomains = ["machine-a.example.com"]
  localIP = "127.0.0.1"
  localPort = 22
  ```

3. 在内部机器B上部署另一个frpc，配置如下：

  ```toml
  serverAddr = "x.x.x.x"
  serverPort = 7000

  [[proxies]]
  name = "ssh2"
  type = "tcpmux"
  multiplexer = "httpconnect"
  customDomains = ["machine-b.example.com"]
  localIP = "127.0.0.1"
  localPort = 22
  ```

4. 使用SSH ProxyCommand访问内部机器A，假设用户名为"test"：

  `ssh -o 'proxycommand socat - PROXY:x.x.x.x:%h:%p,proxyport=5002' test@machine-a.example.com`

5. 要访问内部机器B，唯一的区别是域名，假设用户名为"test"：

  `ssh -o 'proxycommand socat - PROXY:x.x.x.x:%h:%p,proxyport=5002' test@machine-b.example.com`

### 使用自定义域名访问局域网内的Web服务

有时我们需要将NAT网络后面的本地Web服务暴露给他人，以便使用我们自己的域名进行测试。

不幸的是，我们无法将域名解析到本地IP。但是，我们可以使用frp来暴露HTTP(S)服务。

1. 修改`frps.toml`并将vhost的HTTP端口设置为8080：

  ```toml
  # frps.toml
  bindPort = 7000
  vhostHTTPPort = 8080
  ```

  如果要配置https代理，需要设置`vhostHTTPSPort`。

2. 启动`frps`：

  `./frps -c ./frps.toml`

3. 修改`frpc.toml`并将`serverAddr`设置为远程frps服务器的IP地址。指定Web服务的`localPort`：

  ```toml
  # frpc.toml
  serverAddr = "x.x.x.x"
  serverPort = 7000

  [[proxies]]
  name = "web"
  type = "http"
  localPort = 80
  customDomains = ["www.example.com"]
  ```

4. 启动`frpc`：

  `./frpc -c ./frpc.toml`

5. 将`www.example.com`的A记录映射到远程frps服务器的公网IP，或将CNAME记录指向您的原始域名。

6. 使用URL `http://www.example.com:8080` 访问您的本地Web服务。

### 转发DNS查询请求

1. 修改`frps.toml`：

  ```toml
  # frps.toml
  bindPort = 7000
  ```

2. 启动`frps`：

  `./frps -c ./frps.toml`

3. 修改`frpc.toml`并将`serverAddr`设置为远程frps服务器的IP地址。将DNS查询请求转发到Google公共DNS服务器`8.8.8.8:53`：

  ```toml
  # frpc.toml
  serverAddr = "x.x.x.x"
  serverPort = 7000

  [[proxies]]
  name = "dns"
  type = "udp"
  localIP = "8.8.8.8"
  localPort = 53
  remotePort = 6000
  ```

4. 启动frpc：

  `./frpc -c ./frpc.toml`

5. 使用`dig`命令测试DNS解析：

  `dig @x.x.x.x -p 6000 www.google.com`

### 转发Unix域套接字

将Unix域套接字（例如Docker守护进程套接字）作为TCP暴露。

按上述方式配置`frps`。

1. 使用以下配置启动`frpc`：

  ```toml
  # frpc.toml
  serverAddr = "x.x.x.x"
  serverPort = 7000

  [[proxies]]
  name = "unix_domain_socket"
  type = "tcp"
  remotePort = 6000
  [proxies.plugin]
  type = "unix_domain_socket"
  unixPath = "/var/run/docker.sock"
  ```

2. 通过使用`curl`获取docker版本来测试配置：

  `curl http://x.x.x.x:6000/version`

### 暴露简单的HTTP文件服务器

暴露一个简单的HTTP文件服务器，以便从公共互联网访问存储在局域网中的文件。

按上述方式配置`frps`，然后：

1. 使用以下配置启动`frpc`：

  ```toml
  # frpc.toml
  serverAddr = "x.x.x.x"
  serverPort = 7000

  [[proxies]]
  name = "test_static_file"
  type = "tcp"
  remotePort = 6000
  [proxies.plugin]
  type = "static_file"
  localPath = "/tmp/files"
  stripPrefix = "static"
  httpUser = "abc"
  httpPassword = "abc"
  ```

2. 从浏览器访问`http://x.x.x.x:6000/static/`，并指定正确的用户名和密码以查看`frpc`机器上`/tmp/files`中的文件。

### 为本地HTTP(S)服务启用HTTPS

您可以将`https2https`替换为插件，并将`localAddr`指向HTTPS端点。

1. 使用以下配置启动`frpc`：

  ```toml
  # frpc.toml
  serverAddr = "x.x.x.x"
  serverPort = 7000

  [[proxies]]
  name = "test_https2http"
  type = "https"
  customDomains = ["test.example.com"]

  [proxies.plugin]
  type = "https2http"
  localAddr = "127.0.0.1:80"
  crtPath = "./server.crt"
  keyPath = "./server.key"
  hostHeaderRewrite = "127.0.0.1"
  requestHeaders.set.x-from-where = "frp"
  ```

2. 访问`https://test.example.com`。

### 私密暴露您的服务

为了降低将某些服务直接暴露到公共网络的风险，STCP（Secret TCP）模式要求使用预共享密钥从其他客户端访问服务。

按上述方式配置`frps`。

1. 在机器B上使用以下配置启动`frpc`。此示例用于暴露SSH服务（端口22），请注意`secretKey`字段用于预共享密钥，并且此处删除了`remotePort`字段：

  ```toml
  # frpc.toml
  serverAddr = "x.x.x.x"
  serverPort = 7000

  [[proxies]]
  name = "secret_ssh"
  type = "stcp"
  secretKey = "abcdefg"
  localIP = "127.0.0.1"
  localPort = 22
  ```

2. 使用以下配置启动另一个`frpc`（通常在另一台机器C上），以使用安全密钥（`secretKey`字段）访问SSH服务：

  ```toml
  # frpc.toml
  serverAddr = "x.x.x.x"
  serverPort = 7000

  [[visitors]]
  name = "secret_ssh_visitor"
  type = "stcp"
  serverName = "secret_ssh"
  secretKey = "abcdefg"
  bindAddr = "127.0.0.1"
  bindPort = 6000
  ```

3. 在机器C上，使用以下命令连接到机器B上的SSH：

  `ssh -oPort=6000 127.0.0.1`

### P2P模式

**xtcp**旨在直接在客户端之间传输大量数据。仍然需要frps服务器，因为这里的P2P仅指实际的数据传输。

请注意，它可能不适用于所有类型的NAT设备。如果xtcp不起作用，您可能需要回退到stcp。

1. 在机器B上启动`frpc`，并暴露SSH端口。请注意`remotePort`字段已被删除：

  ```toml
  # frpc.toml
  serverAddr = "x.x.x.x"
  serverPort = 7000
  # 如果默认的stun服务器不可用，请设置新的stun服务器。
  # natHoleStunServer = "xxx"

  [[proxies]]
  name = "p2p_ssh"
  type = "xtcp"
  secretKey = "abcdefg"
  localIP = "127.0.0.1"
  localPort = 22
  ```

2. 使用配置启动另一个`frpc`（通常在另一台机器C上），以使用P2P模式连接到SSH：

  ```toml
  # frpc.toml
  serverAddr = "x.x.x.x"
  serverPort = 7000
  # 如果默认的stun服务器不可用，请设置新的stun服务器。
  # natHoleStunServer = "xxx"

  [[visitors]]
  name = "p2p_ssh_visitor"
  type = "xtcp"
  serverName = "p2p_ssh"
  secretKey = "abcdefg"
  bindAddr = "127.0.0.1"
  bindPort = 6000
  # 当需要自动隧道持久化时，将其设置为true
  keepTunnelOpen = false
  ```

3. 在机器C上，使用以下命令连接到机器B上的SSH：

  `ssh -oPort=6000 127.0.0.1`

## 特性

### 配置文件

从v0.52.0开始，我们支持TOML、YAML和JSON进行配置。请注意，INI已被弃用，将在未来版本中删除。新特性将仅在TOML、YAML或JSON中可用。需要这些新特性的用户应相应地切换其配置格式。

阅读完整的示例配置文件，以了解这里未描述的更多特性。

示例使用TOML格式，但您仍然可以使用YAML或JSON。

这些配置文件仅供参考。请不要直接使用此配置来运行程序，因为它可能存在各种问题。

[frps（服务器）的完整配置文件](./conf/frps_full_example.toml)

[frpc（客户端）的完整配置文件](./conf/frpc_full_example.toml)

### 使用环境变量

环境变量可以在配置文件中引用，使用Go的标准格式：

```toml
# frpc.toml
serverAddr = "{{ .Envs.FRP_SERVER_ADDR }}"
serverPort = 7000

[[proxies]]
name = "ssh"
type = "tcp"
localIP = "127.0.0.1"
localPort = 22
remotePort = {{ .Envs.FRP_SSH_REMOTE_PORT }}
```

使用上面的配置，可以通过以下方式将变量传递到`frpc`程序：

```
export FRP_SERVER_ADDR=x.x.x.x
export FRP_SSH_REMOTE_PORT=6000
./frpc -c ./frpc.toml
```

`frpc`将使用OS环境变量渲染配置文件模板。请记住在引用前加上`.Envs`前缀。

### 将配置拆分到不同文件

您可以将多个代理配置拆分到不同文件中，并在主文件中包含它们。

```toml
# frpc.toml
serverAddr = "x.x.x.x"
serverPort = 7000
includes = ["./confd/*.toml"]
```

```toml
# ./confd/test.toml

[[proxies]]
name = "ssh"
type = "tcp"
localIP = "127.0.0.1"
localPort = 22
remotePort = 6000
```

### 服务器仪表盘

通过仪表盘检查frp的状态和代理的统计信息。

配置仪表盘端口以启用此功能：

```toml
# 默认值为127.0.0.1。当您想从公共网络访问它时，将其更改为0.0.0.0。
webServer.addr = "0.0.0.0"
webServer.port = 7500
# dashboard的用户名和密码都是可选的
webServer.user = "admin"
webServer.password = "admin"
```

然后访问`http://[serverAddr]:7500`查看仪表盘，用户名和密码均为`admin`。

此外，您可以使用HTTPS端口，使用您的域的通配符或普通SSL证书：

```toml
webServer.port = 7500
# dashboard的用户名和密码都是可选的
webServer.user = "admin"
webServer.password = "admin"
webServer.tls.certFile = "server.crt"
webServer.tls.keyFile = "server.key"
```

然后访问`https://[serverAddr]:7500`以安全的HTTPS连接查看仪表盘，用户名和密码均为`admin`。

![dashboard](/doc/pic/dashboard.png)

### 客户端管理界面

客户端管理界面帮助您检查和管理frpc的配置。

配置管理界面地址以启用此功能：

```toml
webServer.addr = "127.0.0.1"
webServer.port = 7400
webServer.user = "admin"
webServer.password = "admin"
```

然后访问`http://127.0.0.1:7400`查看管理界面，用户名和密码均为`admin`。

### 监控

启用Web服务器后，frps将在缓存中保存7天的监控数据。进程重启后将被清除。

还支持Prometheus。

#### Prometheus

首先启用仪表盘，然后在`frps.toml`中配置`enablePrometheus = true`。

`http://{dashboard_addr}/metrics`将提供prometheus监控数据。

### 客户端认证

有2种认证方法可以认证frpc和frps。

您可以通过在`frpc.toml`和`frps.toml`中配置`auth.method`来决定使用哪一种，默认是token。

配置`auth.additionalScopes = ["HeartBeats"]`将使用配置的认证方法在frpc和frps之间的每次心跳中添加和验证认证。

配置`auth.additionalScopes = ["NewWorkConns"]`将对frpc和frps之间的每个新工作连接执行相同的操作。

#### Token认证

当在`frpc.toml`和`frps.toml`中指定`auth.method = "token"`时，将使用基于token的认证。

确保在`frps.toml`和`frpc.toml`中指定相同的`auth.token`，以便frpc通过frps验证。

##### Token来源

frp支持使用`tokenSource`配置从外部源读取认证token。目前支持基于文件的token源。

**基于文件的token源：**

```toml
# frpc.toml
auth.method = "token"
auth.tokenSource.type = "file"
auth.tokenSource.file.path = "/path/to/token/file"
```

token将在启动时从指定文件中读取。这对于token由外部系统管理或出于安全原因需要与配置文件分开保存的场景非常有用。

#### OIDC认证

当在`frpc.toml`和`frps.toml`中指定`auth.method = "oidc"`时，将使用基于OIDC的认证。

OIDC代表OpenID Connect，使用的流程称为[客户端凭证授予](https://tools.ietf.org/html/rfc6749#section-4.4)。

要使用此认证类型，请按如下方式配置`frpc.toml`和`frps.toml`：

```toml
# frps.toml
auth.method = "oidc"
auth.oidc.issuer = "https://example-oidc-issuer.com/"
auth.oidc.audience = "https://oidc-audience.com/.default"
```

```toml
# frpc.toml
auth.method = "oidc"
auth.oidc.clientID = "98692467-37de-409a-9fac-bb2585826f18" # 替换为OIDC客户端ID
auth.oidc.clientSecret = "oidc_secret"
auth.oidc.audience = "https://oidc-audience.com/.default"
auth.oidc.tokenEndpointURL = "https://example-oidc-endpoint.com/oauth2/v2.0/token"
```

### 加密和压缩

这些特性默认是关闭的。您可以开启加密和/或压缩：

```toml
# frpc.toml

[[proxies]]
name = "ssh"
type = "tcp"
localPort = 22
remotePort = 6000
transport.useEncryption = true
transport.useCompression = true
```

#### TLS

从v0.50.0开始，`transport.tls.enable`和`transport.tls.disableCustomTLSFirstByte`的默认值已更改为true，默认启用了tls。

对于端口复用，frp发送第一个字节`0x17`来拨打TLS连接。这仅在您将`transport.tls.disableCustomTLSFirstByte`设置为false时生效。

要**强制**`frps`仅接受TLS连接，请在`frps.toml`中配置`transport.tls.force = true`。**这是可选的。**

**`frpc` TLS设置：**

```toml
transport.tls.enable = true
transport.tls.certFile = "certificate.crt"
transport.tls.keyFile = "certificate.key"
transport.tls.trustedCaFile = "ca.crt"
```

**`frps` TLS设置：**

```toml
transport.tls.force = true
transport.tls.certFile = "certificate.crt"
transport.tls.keyFile = "certificate.key"
transport.tls.trustedCaFile = "ca.crt"
```

您将需要**一个根CA证书**和**至少一个SSL/TLS证书**。它**可以**是自签名的或常规的（例如Let's Encrypt或其他SSL/TLS证书提供商）。

如果您通过IP地址而不是主机名使用`frp`，请确保在生成SSL/TLS证书时在主题备用名称（SAN）区域中设置适当的IP地址。

举个例子：

* 准备openssl配置文件。它存在于Linux系统的`/etc/pki/tls/openssl.cnf`和MacOS的`/System/Library/OpenSSL/openssl.cnf`中，您可以将其复制到当前路径，如`cp /etc/pki/tls/openssl.cnf ./my-openssl.cnf`。如果没有，您可以自己构建，如：
```
cat > my-openssl.cnf << EOF
[ ca ]
default_ca = CA_default
[ CA_default ]
x509_extensions = usr_cert
[ req ]
default_bits        = 2048
default_md          = sha256
default_keyfile     = privkey.pem
distinguished_name  = req_distinguished_name
attributes          = req_attributes
x509_extensions     = v3_ca
string_mask         = utf8only
[ req_distinguished_name ]
[ req_attributes ]
[ usr_cert ]
basicConstraints       = CA:FALSE
nsComment              = "OpenSSL Generated Certificate"
subjectKeyIdentifier   = hash
authorityKeyIdentifier = keyid,issuer
[ v3_ca ]
subjectKeyIdentifier   = hash
authorityKeyIdentifier = keyid:always,issuer
basicConstraints       = CA:true
EOF
```

* 构建ca证书：
```
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -subj "/CN=example.ca.com" -days 5000 -out ca.crt
```

* 构建frps证书：
```
openssl genrsa -out server.key 2048

openssl req -new -sha256 -key server.key \
    -subj "/C=XX/ST=DEFAULT/L=DEFAULT/O=DEFAULT/CN=server.com" \
    -reqexts SAN \
    -config <(cat my-openssl.cnf <(printf "\n[SAN]\nsubjectAltName=DNS:localhost,IP:127.0.0.1,DNS:example.server.com")) \
    -out server.csr

openssl x509 -req -days 365 -sha256 \
	-in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
	-extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1,DNS:example.server.com") \
	-out server.crt
```

* 构建frpc证书：
```
openssl genrsa -out client.key 2048
openssl req -new -sha256 -key client.key \
    -subj "/C=XX/ST=DEFAULT/L=DEFAULT/O=DEFAULT/CN=client.com" \
    -reqexts SAN \
    -config <(cat my-openssl.cnf <(printf "\n[SAN]\nsubjectAltName=DNS:client.com,DNS:example.client.com")) \
    -out client.csr

openssl x509 -req -days 365 -sha256 \
    -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
	-extfile <(printf "subjectAltName=DNS:client.com,DNS:example.client.com") \
	-out client.crt
```

### 热重载frpc配置

`webServer`字段是启用HTTP API所必需的：

```toml
# frpc.toml
webServer.addr = "127.0.0.1"
webServer.port = 7400
```

然后运行命令`frpc reload -c ./frpc.toml`，等待约10秒，让`frpc`创建、更新或删除代理。

**注意，全局客户端参数不会被修改，除了'start'。**

您可以在重载之前运行命令`frpc verify -c ./frpc.toml`来检查是否存在配置错误。

### 从客户端获取代理状态

使用`frpc status -c ./frpc.toml`获取所有代理的状态。`webServer`字段是启用HTTP API所必需的。

### 仅允许服务器上的特定端口

`frps.toml`中的`allowPorts`用于避免端口滥用：

```toml
# frps.toml
allowPorts = [
  { start = 2000, end = 3000 },
  { single = 3001 },
  { single = 3003 },
  { start = 4000, end = 50000 }
]
```

### 端口复用

frps中的`vhostHTTPPort`和`vhostHTTPSPort`可以与`bindPort`使用相同的端口。frps将检测连接的协议并相应地处理它。

您需要注意的是，如果要将`vhostHTTPSPort`和`bindPort`配置为相同的端口，您需要首先将`transport.tls.disableCustomTLSFirstByte`设置为false。

我们希望将来尝试允许多个代理使用不同的协议绑定到同一个远程端口。

### 带宽限制

#### 每个代理

```toml
# frpc.toml

[[proxies]]
name = "ssh"
type = "tcp"
localPort = 22
remotePort = 6000
transport.bandwidthLimit = "1MB"
```

在每个代理的配置中设置`transport.bandwidthLimit`以启用此功能。支持的单位是`MB`和`KB`。

设置`transport.bandwidthLimitMode`为`client`或`server`以限制客户端或服务器端的带宽。默认为`client`。

### TCP流多路复用

frp从v0.10.0开始支持TCP流多路复用，如HTTP2多路复用，在这种情况下，到同一frpc的所有逻辑连接都被多路复用到同一个TCP连接中。

您可以通过修改`frps.toml`和`frpc.toml`来禁用此功能：

```toml
# frps.toml和frpc.toml，必须相同
transport.tcpMux = false
```

### 支持KCP协议

KCP是一种快速可靠的协议，可以实现平均延迟减少30%至40%，最大延迟减少三倍的传输效果，代价是比TCP多浪费10%至20%的带宽。

KCP模式使用UDP作为底层传输。在frp中使用KCP：

1. 在frps中启用KCP：

  ```toml
  # frps.toml
  bindPort = 7000
  # 为KCP指定一个UDP端口。
  kcpBindPort = 7000
  ```

  `kcpBindPort`数字可以与`bindPort`相同，因为`bindPort`字段指定TCP端口。

2. 配置`frpc.toml`使用KCP连接到frps：

  ```toml
  # frpc.toml
  serverAddr = "x.x.x.x"
  # 与frps.toml中的'kcpBindPort'相同
  serverPort = 7000
  transport.protocol = "kcp"
  ```

### 支持QUIC协议

QUIC是一种基于UDP构建的新型多路复用传输。

在frp中使用QUIC：

1. 在frps中启用QUIC：

  ```toml
  # frps.toml
  bindPort = 7000
  # 为QUIC指定一个UDP端口。
  quicBindPort = 7000
  ```

  `quicBindPort`数字可以与`bindPort`相同，因为`bindPort`字段指定TCP端口。

2. 配置`frpc.toml`使用QUIC连接到frps：

  ```toml
  # frpc.toml
  serverAddr = "x.x.x.x"
  # 与frps.toml中的'quicBindPort'相同
  serverPort = 7000
  transport.protocol = "quic"
  ```

### 连接池

默认情况下，frps会根据用户请求创建到后端服务的新frpc连接。通过连接池，frps会保持一定数量的预建立连接，减少建立连接所需的时间。

此功能适用于大量短连接的场景。

1. 在`frps.toml`中配置每个代理可以使用的池计数限制：

  ```toml
  # frps.toml
  transport.maxPoolCount = 5
  ```

2. 启用并指定连接池数量：

  ```toml
  # frpc.toml
  transport.poolCount = 1
  ```

### 负载均衡

负载均衡通过`group`支持。

此功能目前仅适用于类型`tcp`、`http`、`tcpmux`。

```toml
# frpc.toml

[[proxies]]
name = "test1"
type = "tcp"
localPort = 8080
remotePort = 80
loadBalancer.group = "web"
loadBalancer.groupKey = "123"

[[proxies]]
name = "test2"
type = "tcp"
localPort = 8081
remotePort = 80
loadBalancer.group = "web"
loadBalancer.groupKey = "123"
```

`loadBalancer.groupKey`用于认证。

到端口80的连接将被随机分发到同一组中的代理。

对于类型`tcp`，同一组中的`remotePort`应该相同。

对于类型`http`，`customDomains`、`subdomain`、`locations`应该相同。

### 服务健康检查

健康检查功能可以帮助您通过负载均衡实现高可用性。

添加`healthCheck.type = "tcp"`或`healthCheck.type = "http"`来启用健康检查。

使用健康检查类型**tcp**，服务端口将被ping（TCPing）：

```toml
# frpc.toml

[[proxies]]
name = "test1"
type = "tcp"
localPort = 22
remotePort = 6000
# 启用TCP健康检查
healthCheck.type = "tcp"
# TCPing超时秒数
healthCheck.timeoutSeconds = 3
# 如果健康检查连续失败3次，代理将从frps中删除
healthCheck.maxFailed = 3
# 每10秒进行一次健康检查
healthCheck.intervalSeconds = 10
```

使用健康检查类型**http**，将向服务发送HTTP请求，并期望HTTP 2xx OK响应：

```toml
# frpc.toml

[[proxies]]
name = "web"
type = "http"
localIP = "127.0.0.1"
localPort = 80
customDomains = ["test.example.com"]
# 启用HTTP健康检查
healthCheck.type = "http"
# frpc将发送GET请求到'/status'
# 并期望HTTP 2xx OK响应
healthCheck.path = "/status"
healthCheck.timeoutSeconds = 3
healthCheck.maxFailed = 3
healthCheck.intervalSeconds = 10
```

### 重写HTTP Host头

默认情况下，frp不会修改隧道HTTP请求，因为它是字节对字节的副本。

然而，对于Web服务器和HTTP请求，您的Web服务器可能依赖于`Host` HTTP头来确定要访问的网站。frp可以在转发HTTP请求时重写`Host`头，使用`hostHeaderRewrite`字段：

```toml
# frpc.toml

[[proxies]]
name = "web"
type = "http"
localPort = 80
customDomains = ["test.example.com"]
hostHeaderRewrite = "dev.example.com"
```

当HTTP请求到达实际的Web服务器时，它将具有被重写为`Host: dev.example.com`的`Host`头，尽管来自浏览器的请求可能具有`Host: test.example.com`。

### 设置其他HTTP头

类似于`Host`，您可以使用代理类型`http`覆盖其他HTTP请求和响应头。

```toml
# frpc.toml

[[proxies]]
name = "web"
type = "http"
localPort = 80
customDomains = ["test.example.com"]
hostHeaderRewrite = "dev.example.com"
requestHeaders.set.x-from-where = "frp"
responseHeaders.set.foo = "bar"
```

在此示例中，它将在HTTP请求中设置头`x-from-where: frp`，在HTTP响应中设置`foo: bar`。

### 获取真实IP

#### HTTP X-Forwarded-For

此功能适用于`http`代理或启用了`https2http`和`https2https`插件的代理。

您可以从HTTP请求头`X-Forwarded-For`获取用户的真实IP。

#### Proxy Protocol

frp支持Proxy Protocol将用户的真实IP发送到本地服务。

以下是https服务的示例：

```toml
# frpc.toml

[[proxies]]
name = "web"
type = "https"
localPort = 443
customDomains = ["test.example.com"]

# 现在支持v1和v2
transport.proxyProtocolVersion = "v2"
```

您可以在nginx中启用Proxy Protocol支持，以在HTTP头`X-Real-IP`中暴露用户的真实IP，然后在Web服务中读取`X-Real-IP`头获取真实IP。

### 为Web服务要求HTTP基本认证（密码）

任何能够猜测您的隧道URL的人都可以访问您的本地Web服务器，除非您用密码保护它。

这会在所有请求上强制执行HTTP基本认证，使用frpc配置文件中指定的用户名和密码。

它只能在代理类型为http时启用。

```toml
# frpc.toml

[[proxies]]
name = "web"
type = "http"
localPort = 80
customDomains = ["test.example.com"]
httpUser = "abc"
httpPassword = "abc"
```

在浏览器中访问`http://test.example.com`，现在您会被提示输入用户名和密码。

### 自定义子域名

当多人共享一个frps服务器时，对http和https类型使用`subdomain`配置很方便。

```toml
# frps.toml
subDomainHost = "frps.com"
```

将`*.frps.com`解析到frps服务器的IP。这通常称为通配符DNS记录。

```toml
# frpc.toml

[[proxies]]
name = "web"
type = "http"
localPort = 80
subdomain = "test"
```

现在您可以在`test.frps.com`上访问您的Web服务。

请注意，如果`subdomainHost`不为空，`customDomains`不应是`subdomainHost`的子域名。

### URL路由

frp支持通过URL路由将HTTP请求转发到不同的后端Web服务。

`locations`指定用于路由的URL前缀。frps首先搜索由文字字符串给出的最具体的前缀位置，而不管列出的顺序如何。

```toml
# frpc.toml

[[proxies]]
name = "web01"
type = "http"
localPort = 80
customDomains = ["web.example.com"]
locations = ["/"]

[[proxies]]
name = "web02"
type = "http"
localPort = 81
customDomains = ["web.example.com"]
locations = ["/news", "/about"]
```

URL前缀为`/news`或`/about`的HTTP请求将被转发到**web02**，其他请求将被转发到**web01**。

### TCP端口多路复用

frp支持在frps的单个端口上接收定向到不同代理的TCP套接字，类似于`vhostHTTPPort`和`vhostHTTPSPort`。

目前唯一支持的TCP端口多路复用方法是`httpconnect` - HTTP CONNECT隧道。

当在frps中设置`tcpmuxHTTPConnectPort`为0以外的任何值时，frps将在此端口上监听HTTP CONNECT请求。

HTTP CONNECT请求的主机将用于匹配frps中的代理。代理主机可以通过在`tcpmux`代理下配置`customDomains`和/或`subdomain`来在frpc中配置，当`multiplexer = "httpconnect"`时。

例如：

```toml
# frps.toml
bindPort = 7000
tcpmuxHTTPConnectPort = 1337
```

```toml
# frpc.toml
serverAddr = "x.x.x.x"
serverPort = 7000

[[proxies]]
name = "proxy1"
type = "tcpmux"
multiplexer = "httpconnect"
customDomains = ["test1"]
localPort = 80

[[proxies]]
name = "proxy2"
type = "tcpmux"
multiplexer = "httpconnect"
customDomains = ["test2"]
localPort = 8080
```

在上面的配置中，frps可以在端口1337上通过HTTP CONNECT头联系，例如：

```
CONNECT test1 HTTP/1.1\r\n\r\n
```
连接将被路由到`proxy1`。

### 通过PROXY连接到frps

如果您设置OS环境变量`HTTP_PROXY`，或者在frpc.toml文件中设置`transport.proxyURL`，frpc可以通过代理连接到frps。

它仅在协议为tcp时工作。

```toml
# frpc.toml
serverAddr = "x.x.x.x"
serverPort = 7000
transport.proxyURL = "http://user:pwd@192.168.1.128:8080"
```

### 端口范围映射

*在v0.56.0中添加*

我们可以使用Go模板的范围语法结合内置的`parseNumberRangePair`函数来实现端口范围映射。

以下示例运行时将创建8个代理，名称为`test-6000, test-6001 ... test-6007`，每个映射远程端口到本地端口。

```
{{- range $_, $v := parseNumberRangePair "6000-6006,6007" "6000-6006,6007" }}
[[proxies]]
name = "tcp-{{ $v.First }}"
type = "tcp"
localPort = {{ $v.First }}
remotePort = {{ $v.Second }}
{{- end }}
```

### 客户端插件

默认情况下，frpc仅将请求转发到本地TCP或UDP端口。

插件用于提供丰富的功能。有内置插件如`unix_domain_socket`、`http_proxy`、`socks5`、`static_file`、`http2https`、`https2http`、`https2https`，您可以看到[示例用法](#example-usage)。

使用插件**http_proxy**：

```toml
# frpc.toml

[[proxies]]
name = "http_proxy"
type = "tcp"
remotePort = 6000
[proxies.plugin]
type = "http_proxy"
httpUser = "abc"
httpPassword = "abc"
```

`httpUser`和`httpPassword`是`http_proxy`插件中使用的配置参数。

### 服务器管理插件

阅读[文档](/doc/server_plugin.md)。

在[gofrp/plugin](https://github.com/gofrp/plugin)中找到更多插件。

### SSH隧道网关

*在v0.53.0中添加*

frp支持在frps端监听SSH端口，并通过SSH -R协议实现TCP协议代理，无需依赖frpc。

```toml
# frps.toml
sshTunnelGateway.bindPort = 2200
```

运行`./frps -c frps.toml`时，当前工作目录中会自动创建一个名为`.autogen_ssh_key`的私钥文件。这个生成的私钥文件将被frps中的SSH服务器使用。

执行命令

```bash
ssh -R :80:127.0.0.1:8080 v0@{frp address} -p 2200 tcp --proxy_name "test-tcp" --remote_port 9090
```

在frps上设置一个代理，将本地8080服务转发到端口9090。

```bash
frp (via SSH) (Ctrl+C to quit)

User:
ProxyName: test-tcp
Type: tcp
RemoteAddress: :9090
```

这相当于：

```bash
frpc tcp --proxy_name "test-tcp" --local_ip 127.0.0.1 --local_port 8080 --remote_port 9090
```

有关更多信息，请参考此[文档](/doc/ssh_tunnel_gateway.md)。

### 虚拟网络（VirtualNet）

*在v0.62.0中添加的Alpha特性*

VirtualNet特性使frp能够通过TUN接口在客户端和访问者之间创建和管理虚拟网络连接。这允许机器之间的IP级路由，将frp从简单的端口转发扩展到支持完整的网络连接。

有关配置和使用的详细信息，请参阅[VirtualNet文档](/doc/virtual_net.md)。

## 特性开关

frp支持特性开关来启用或禁用实验性功能。这允许用户在功能被认为稳定之前尝试新功能。

### 可用的特性开关

| 名称 | 阶段 | 默认 | 描述 |
|------|-------|---------|-------------|
| VirtualNet | ALPHA | false | frp的虚拟网络功能 |

### 启用特性开关

要启用实验性功能，请将特性开关添加到您的配置中：

```toml
featureGates = { VirtualNet = true }
```

### 特性生命周期

特性通常经历三个阶段：
1. **ALPHA**：默认禁用，可能不稳定
2. **BETA**：可能默认启用，更稳定但仍在发展中
3. **GA（一般可用）**：默认启用，可用于生产环境

## 相关项目

* [gofrp/plugin](https://github.com/gofrp/plugin) - frp插件的存储库，包含基于frp扩展机制实现的各种插件，满足不同场景的定制需求。
* [gofrp/tiny-frpc](https://github.com/gofrp/tiny-frpc) - frp客户端的轻量级版本（最小约3.5MB），使用ssh协议实现，支持一些最常用的功能，适用于资源有限的设备。

## 贡献

有兴趣参与吗？我们很乐意帮助您！

* 查看我们的[issues列表](https://github.com/purpose168/frp/issues)，并考虑向**dev分支**发送Pull Request。
* 如果您想添加新功能，请先创建一个issue来描述新功能，以及实现方法。一旦提案被接受，请创建新功能的实现并将其作为pull request提交。
* 抱歉，我的英语不好。欢迎对本文档进行改进，甚至是一些拼写修正。
* 如果您有好的想法，请发送电子邮件至fatedier@gmail.com。

**注意：我们希望您在[issues](https://github.com/purpose168/frp/issues)中给出您的建议，这样有相同问题的其他人可以快速搜索到，我们也不需要重复回答。**

## 捐赠

如果frp对您有很大帮助，您可以通过以下方式支持我们：

### GitHub赞助商

通过[Github Sponsors](https://github.com/sponsors/fatedier)支持我们。

您可以将公司的徽标放置在这个项目的README文件上。

### PayPal

通过[PayPal](https://www.paypal.me/fatedier)向我的账户**fatedier@gmail.com**捐赠资金。