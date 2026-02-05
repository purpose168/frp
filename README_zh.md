# frp

[![构建状态](https://circleci.com/gh/fatedier/frp.svg?style=shield)](https://circleci.com/gh/fatedier/frp)
[![GitHub 版本](https://img.shields.io/github/tag/fatedier/frp.svg?label=release)](https://github.com/purpose168/frp/releases)
[![Go 报告卡](https://goreportcard.com/badge/github.com/fatedier/frp)](https://goreportcard.com/report/github.com/fatedier/frp)
[![GitHub 下载统计](https://img.shields.io/github/downloads/purpose168/frp/total.svg?logo=github)](https://somsubhra.github.io/github-release-stats/?username=fatedier&repository=frp)

[英文文档](README.md) | [中文文档](README_zh.md)

## 赞助商

frp 是一个开源项目，其持续开发完全依赖于我们出色赞助商的支持。如果您想加入他们，请考虑[赞助 frp 的开发](https://github.com/sponsors/fatedier)。

<h3 align="center">金牌赞助商</h3>
<!--金牌赞助商开始-->
<p align="center">
  <a href="https://requestly.com/?utm_source=github&utm_medium=partnered&utm_campaign=frp" target="_blank">
    <img width="480px" src="https://github.com/user-attachments/assets/24670320-997d-4d62-9bca-955c59fe883d">
    <br>
    <b>Requestly - Postman 的免费开源替代品</b>
    <br>
    <sub>一站式平台，用于测试、模拟和拦截 API。</sub>
  </a>
</p>

<p align="center">
  <a href="https://jb.gg/frp" target="_blank">
    <img width="420px" src="https://raw.githubusercontent.com/purpose168/frp/dev/doc/pic/sponsor_jetbrains.jpg">
	<br>
	<b>为专业 Go 开发者打造的完整 IDE</b>
  </a>
</p>

<p align="center">
  <a href="https://github.com/beclab/Olares" target="_blank">
    <img width="420px" src="https://raw.githubusercontent.com/purpose168/frp/dev/doc/pic/sponsor_olares.jpeg">
	<br>
	<b>让您掌控一切的主权云</b>
	<br>
	<sub>一个开源的、自托管的公共云替代品，专为数据所有权和隐私而构建</sub>
  </a>
</p>
<div align="center">

## Recall.ai - 会议录制 API

如果您正在寻找会议录制 API，请考虑使用 [Recall.ai](https://www.recall.ai/?utm_source=github&utm_medium=sponsorship&utm_campaign=fatedier-frp)，

一个可以录制 Zoom、Google Meet、Microsoft Teams、面对面会议等的 API。

</div>
<!--金牌赞助商结束-->

## 什么是 frp？

frp 是一个快速的反向代理，允许您将位于 NAT 或防火墙后面的本地服务器暴露到互联网上。它目前支持 **TCP** 和 **UDP** 协议，以及 **HTTP** 和 **HTTPS** 协议，允许通过域名将请求转发到内部服务。

frp 还提供了 P2P 连接模式。

## 目录

<!-- vim-markdown-toc GFM -->

* [开发状态](#开发状态)
    * [关于 V2 版本](#关于-v2-版本)
* [架构](#架构)
* [示例用法](#示例用法)
    * [通过 SSH 访问局域网中的计算机](#通过-ssh-访问局域网中的计算机)
    * [多个 SSH 服务共享同一端口](#多个-ssh-服务共享同一端口)
    * [使用自定义域名访问局域网内的 Web 服务](#使用自定义域名访问局域网内的-web-服务)
    * [转发 DNS 查询请求](#转发-dns-查询请求)
    * [转发 Unix 域套接字](#转发-unix-域套接字)
    * [暴露一个简单的 HTTP 文件服务器](#暴露一个简单的-http-文件服务器)
    * [为本地 HTTP(S) 服务启用 HTTPS](#为本地-https-服务启用-https)
    * [私密暴露您的服务](#私密暴露您的服务)
    * [P2P 模式](#p2p-模式)
* [功能特性](#功能特性)
    * [配置文件](#配置文件)
    * [使用环境变量](#使用环境变量)
    * [将配置拆分为不同文件](#将配置拆分为不同文件)
    * [服务器仪表盘](#服务器仪表盘)
    * [客户端管理 UI](#客户端管理-ui)
    * [监控](#监控)
        * [Prometheus](#prometheus)
    * [客户端认证](#客户端认证)
        * [Token 认证](#token-认证)
        * [OIDC 认证](#oidc-认证)
    * [加密和压缩](#加密和压缩)
        * [TLS](#tls)
    * [热重载 frpc 配置](#热重载-frpc-配置)
    * [从客户端获取代理状态](#从客户端获取代理状态)
    * [仅允许服务器上的特定端口](#仅允许服务器上的特定端口)
    * [端口复用](#端口复用)
    * [带宽限制](#带宽限制)
        * [每个代理](#每个代理)
    * [TCP 流多路复用](#tcp-流多路复用)
    * [支持 KCP 协议](#支持-kcp-协议)
    * [支持 QUIC 协议](#支持-quic-协议)
    * [连接池](#连接池)
    * [负载均衡](#负载均衡)
    * [服务健康检查](#服务健康检查)
    * [重写 HTTP Host 头](#重写-http-host-头)
    * [设置其他 HTTP 头](#设置其他-http-头)
    * [获取真实 IP](#获取真实-ip)
        * [HTTP X-Forwarded-For](#http-x-forwarded-for)
        * [代理协议](#代理协议)
    * [为 Web 服务要求 HTTP 基本认证（密码）](#为-web-服务要求-http-基本认证密码)
    * [自定义子域名](#自定义子域名)
    * [URL 路由](#url-路由)
    * [TCP 端口多路复用](#tcp-端口多路复用)
    * [通过 PROXY 连接到 frps](#通过-proxy-连接到-frps)
    * [端口范围映射](#端口范围映射)
    * [客户端插件](#客户端插件)
    * [服务器管理插件](#服务器管理插件)
    * [SSH 隧道网关](#ssh-隧道网关)
    * [虚拟网络（VirtualNet）](#虚拟网络-virtualnet)
* [功能门控](#功能门控)
    * [可用功能门控](#可用功能门控)
    * [启用功能门控](#启用功能门控)
    * [功能生命周期](#功能生命周期)
* [相关项目](#相关项目)
* [贡献](#贡献)
* [捐赠](#捐赠)
    * [GitHub 赞助商](#github-赞助商)
    * [PayPal](#paypal)

<!-- vim-markdown-toc -->

## 开发状态

frp 目前正在开发中。您可以尝试 `master` 分支中的最新版本，或者使用 `dev` 分支访问当前正在开发的版本。

我们目前正在开发版本 2，并尝试进行一些代码重构和改进。但请注意，它将与版本 1 不兼容。

我们将在适当的时候从版本 0 过渡到版本 1，并且只接受 bug 修复和改进，而不是大型功能请求。

### 关于 V2 版本

V2 版本的复杂性和难度远远超出了预期。我只能利用碎片化的时间进行开发，而不断的中断严重影响了工作效率。鉴于这种情况，我们将继续优化和迭代当前版本，直到我们有更多的空闲时间来进行重大版本的全面升级。

V2 的概念基于我多年来在云原生领域的经验和思考，特别是在 K8s 和 ServiceMesh 方面。它的核心是一个现代化的四层和七层代理，类似于 envoy。这个代理本身具有高度的可扩展性，不仅能够实现内网穿透的功能，还适用于各种其他领域。基于这个高度可扩展的核心，我们的目标是实现 frp v1 的所有功能，同时解决以前无法实现或难以优雅实现的功能。此外，我们将保持高效的开发和迭代能力。

此外，我设想 frp 本身将成为一个高度可扩展的系统和平台，类似于我们可以基于 K8s 提供一系列扩展功能。在 K8s 中，我们可以根据企业需求进行定制开发，利用 CRD、控制器模式、webhook、CSI 和 CNI 等功能。在 frp v1 中，我们引入了服务器插件的概念，实现了一些基本的可扩展性。然而，它依赖于简单的 HTTP 协议，需要用户启动独立的进程并自行管理。这种方式远不够灵活和方便，而现实世界的需求差异很大。期望由少数人维护的非盈利开源项目满足每个人的需求是不现实的。

最后，我们认识到配置管理、权限验证、证书管理和 API 管理等模块的当前设计不够现代化。虽然我们可能会在 v1 版本中进行一些优化，但确保兼容性仍然是一个具有挑战性的问题，需要付出相当大的努力来解决。

我们衷心感谢您对 frp 的支持。

## 架构

![架构](/doc/pic/architecture.png)

## 示例用法

首先，从 [Release](https://github.com/purpose168/frp/releases) 页面下载适用于您的操作系统和架构的最新程序。

接下来，将 `frps` 二进制文件和服务器配置文件放在具有公网 IP 地址的服务器 A 上。

最后，将 `frpc` 二进制文件和客户端配置文件放在无法从公网直接访问的局域网中的服务器 B 上。

一些杀毒软件错误地将 frpc 标记为恶意软件并删除它。这是因为 frp 是一个能够创建反向代理的网络工具。杀毒软件有时会标记反向代理，因为它们能够绕过防火墙端口限制。如果您正在使用杀毒软件，您可能需要在杀毒软件设置中将 frpc 列入白名单/排除，以避免意外隔离/删除。有关更多详细信息，请参阅 [issue 3637](https://github.com/purpose168/frp/issues/3637)。

### 通过 SSH 访问局域网中的计算机

1. 修改服务器 A 上的 `frps.toml`，设置 frp 客户端连接的 `bindPort`：

  ```toml
  # frps.toml
  bindPort = 7000
  ```

2. 在服务器 A 上启动 `frps`：

  `./frps -c ./frps.toml`

3. 修改服务器 B 上的 `frpc.toml`，将 `serverAddr` 字段设置为您的 frps 服务器的公网 IP 地址：

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

请注意，`localPort`（客户端监听的端口）和 `remotePort`（服务器上暴露的端口）用于进出 frp 系统的流量，而 `serverPort` 用于 frps 和 frpc 之间的通信。

4. 在服务器 B 上启动 `frpc`：

  `./frpc -c ./frpc.toml`

5. 要通过服务器 A 从另一台机器 SSH 访问服务器 B（假设用户名是 `test`），请使用以下命令：

  `ssh -oPort=6000 test@x.x.x.x`