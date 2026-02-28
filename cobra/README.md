<div align="center">
<a href="https://cobra.dev">
<img width="512" height="535" alt="cobra-logo" src="https://github.com/user-attachments/assets/c8bf9aad-b5ae-41d3-8899-d83baec10af8" />
</a>
</div>

Cobra 是一个用于创建强大的现代 CLI 应用程序的库。

<a href="https://cobra.dev">访问 Cobra.dev 查看详细文档</a> 


Cobra 被许多 Go 项目使用，例如 [Kubernetes](https://kubernetes.io/)、
[Hugo](https://gohugo.io) 和 [GitHub CLI](https://github.com/cli/cli) 等。
[此列表](site/content/projects_using_cobra.md) 包含了更多使用 Cobra 的项目。

[![](https://img.shields.io/github/actions/workflow/status/spf13/cobra/test.yml?branch=main&longCache=true&label=Test&logo=github%20actions&logoColor=fff)](https://github.com/spf13/cobra/actions?query=workflow%3ATest)
[![Go Reference](https://pkg.go.dev/badge/github.com/spf13/cobra.svg)](https://pkg.go.dev/github.com/spf13/cobra)
[![Go Report Card](https://goreportcard.com/badge/github.com/spf13/cobra)](https://goreportcard.com/report/github.com/spf13/cobra)
[![Slack](https://img.shields.io/badge/Slack-cobra-brightgreen)](https://gophers.slack.com/archives/CD3LP1199)
<hr>
<div align="center" markdown="1">
   <sup>支持者：</sup>
   <br>
   <br>
   <a href="https://www.warp.dev/cobra">
      <img alt="Warp sponsorship" width="400" src="https://github.com/user-attachments/assets/ab8dd143-b0fd-4904-bdc5-dd7ecac94eae">
   </a>

### [Warp，面向开发者的 AI 终端](https://www.warp.dev/cobra)
[立即在 Warp 中试用 Cobra](https://www.warp.dev/cobra)<br>

</div>
<hr>

# 概述

Cobra 是一个库，提供了简单的接口来创建强大的现代 CLI
接口，类似于 git 和 go 工具。

Cobra 提供：
* 简单的基于子命令的 CLI：`app server`、`app fetch` 等
* 完全符合 POSIX 标准的标志（包括短版本和长版本）
* 嵌套子命令
* 全局、局部和级联标志
* 智能建议（`app srver`... 您是指 `app server` 吗？）
* 为命令和标志自动生成帮助
* 子命令的分组帮助
* 自动识别帮助标志 `-h`、`--help` 等
* 为您的应用程序自动生成 shell 自动补全（bash、zsh、fish、powershell）
* 为您的应用程序自动生成 man 手册页
* 命令别名，这样您可以在不破坏它们的情况下进行更改
* 定义您自己的帮助、用法等的灵活性
* 可选地与 [viper](https://github.com/spf13/viper) 无缝集成，用于 12-factor 应用程序

# 概念

Cobra 建立在命令、参数和标志的结构之上。

**命令** 代表操作，**参数** 是事物，**标志** 是这些操作的修饰符。

最好的应用程序在使用时读起来像句子，因此用户
直观地知道如何与它们交互。

要遵循的模式是
`APPNAME VERB NOUN --ADJECTIVE`
    或
`APPNAME COMMAND ARG --FLAG`。

一些很好的现实世界示例可以更好地说明这一点。

在下面的示例中，'server' 是一个命令，'port' 是一个标志：

    hugo server --port=1313

在这个命令中，我们告诉 Git 以裸模式克隆 URL。

    git clone URL --bare

## 命令

命令是应用程序的中心点。应用程序支持的每个交互
都将包含在一个命令中。一个命令可以
有子命令，并可选择执行一个操作。

在上面的示例中，'server' 是命令。

[更多关于 cobra.Command 的信息](https://pkg.go.dev/github.com/spf13/cobra#Command)

## 标志

标志是一种修改命令行为的方式。Cobra 支持
完全符合 POSIX 标准的标志以及 Go [flag 包](https://golang.org/pkg/flag/)。
Cobra 命令可以定义持久化到子命令的标志
以及仅对该命令可用的标志。

在上面的示例中，'port' 是标志。

标志功能由 [pflag
库](https://github.com/spf13/pflag) 提供，这是 flag 标准库的一个分支，
在添加 POSIX 合规性的同时保持相同的接口。

# 安装

使用 Cobra 很简单。首先，使用 `go get` 安装最新版本
的库。

```
go get -u github.com/spf13/cobra@latest
```

接下来，在您的应用程序中包含 Cobra：

```go
import "github.com/spf13/cobra"
```

# 用法

`cobra-cli` 是一个命令行程序，用于生成 cobra 应用程序和命令文件。
它将引导您的应用程序脚手架，以快速
开发基于 Cobra 的应用程序。这是将 Cobra 整合到您的应用程序中的最简单方法。

可以通过运行以下命令安装：

```
go install github.com/spf13/cobra-cli@latest
```

有关使用 Cobra-CLI 生成器的完整详细信息，请阅读 [Cobra 生成器 README](https://github.com/spf13/cobra-cli/blob/main/README.md)

有关使用 Cobra 库的完整详细信息，请阅读 [Cobra 用户指南](site/content/user_guide.md)。

# 许可证

Cobra 在 Apache 2.0 许可证下发布。参见 [LICENSE.txt](LICENSE.txt)
