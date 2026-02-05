# 将Go模块的bin目录添加到系统路径中
export PATH := $(PATH):`go env GOPATH`/bin
# 启用Go模块支持
export GO111MODULE=on
# 编译时的链接器参数：-s 去掉符号表，-w 去掉DWARF调试信息，减小二进制文件大小
LDFLAGS := -s -w

# 声明伪目标，防止与同名文件冲突
.PHONY: web frps-web frpc-web frps frpc

# 默认目标：检查环境变量 → 格式化代码 → 构建Web界面 → 编译二进制文件
all: env fmt web build

# 构建目标：编译frps和frpc两个二进制文件
build: frps frpc

# 检查Go环境版本
env:
	@go version

# 构建Web界面：同时构建frps和frpc的Web管理界面
web: frps-web frpc-web

# 构建frps的Web管理界面
frps-web:
	$(MAKE) -C web/frps build  # 进入web/frps目录执行make build

# 构建frpc的Web管理界面
frpc-web:
	$(MAKE) -C web/frpc build  # 进入web/frpc目录执行make build

# 使用Go官方工具格式化代码
fmt:
	go fmt ./...  # 格式化当前目录及所有子目录下的Go文件

# 使用gofumpt工具进行更严格的代码格式化
fmt-more:
	gofumpt -l -w .  # 检查并修复当前目录下所有Go文件的格式问题

# 使用gci工具整理Go代码的import顺序
gci:
	gci write -s standard -s default -s "prefix(github.com/fatedier/frp/)" ./  # 按照标准库、默认库、本项目库的顺序整理import

# 静态代码检查：先构建Web界面，再运行go vet检查代码
vet: web
	go vet ./...  # 对当前目录及所有子目录下的Go文件进行静态代码检查

# 编译frps（服务器端）二进制文件
frps:
	env CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -tags frps -o bin/frps ./cmd/frps
	# CGO_ENABLED=0 禁用CGO，生成纯Go静态编译的二进制文件
	# -trimpath 去除编译路径信息，减小二进制文件大小
	# -ldflags "$(LDFLAGS)" 传递链接器参数
	# -tags frps 仅编译带有frps标签的代码
	# -o bin/frps 指定输出文件路径
	# ./cmd/frps 主程序入口

# 编译frpc（客户端）二进制文件
frpc:
	env CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -tags frpc -o bin/frpc ./cmd/frpc
	# CGO_ENABLED=0 禁用CGO，生成纯Go静态编译的二进制文件
	# -trimpath 去除编译路径信息，减小二进制文件大小
	# -ldflags "$(LDFLAGS)" 传递链接器参数
	# -tags frpc 仅编译带有frpc标签的代码
	# -o bin/frpc 指定输出文件路径
	# ./cmd/frpc 主程序入口

# 运行所有测试：等同于运行gotest
test: gotest

# 运行单元测试：先构建Web界面，再运行各模块的单元测试
gotest: web
	go test -v --cover ./assets/...  # 测试assets模块并生成覆盖率报告
	go test -v --cover ./cmd/...     # 测试cmd模块并生成覆盖率报告
	go test -v --cover ./client/...  # 测试client模块并生成覆盖率报告
	go test -v --cover ./server/...  # 测试server模块并生成覆盖率报告
	go test -v --cover ./pkg/...     # 测试pkg模块并生成覆盖率报告

# 运行端到端测试
e2e:
	./hack/run-e2e.sh  # 执行端到端测试脚本

# 运行带trace日志的端到端测试
e2e-trace:
	DEBUG=true LOG_LEVEL=trace ./hack/run-e2e.sh  # 开启DEBUG模式和trace级日志，执行端到端测试

# 与上一版本的frpc进行兼容性测试
e2e-compatibility-last-frpc:
	if [ ! -d "./lastversion" ]; then \
		TARGET_DIRNAME=lastversion ./hack/download.sh; \
	fi  # 如果lastversion目录不存在，则下载上一版本的frp
	FRPC_PATH="`pwd`/lastversion/frpc" ./hack/run-e2e.sh  # 使用上一版本的frpc运行端到端测试
	rm -r ./lastversion  # 测试完成后删除lastversion目录

# 与上一版本的frps进行兼容性测试
e2e-compatibility-last-frps:
	if [ ! -d "./lastversion" ]; then \
		TARGET_DIRNAME=lastversion ./hack/download.sh; \
	fi  # 如果lastversion目录不存在，则下载上一版本的frp
	FRPS_PATH="`pwd`/lastversion/frps" ./hack/run-e2e.sh  # 使用上一版本的frps运行端到端测试
	rm -r ./lastversion  # 测试完成后删除lastversion目录

# 运行所有测试：静态代码检查 → 单元测试 → 端到端测试
alltest: vet gotest e2e
	
# 清理构建产物
clean:
	rm -f ./bin/frpc  # 删除客户端二进制文件
	rm -f ./bin/frps  # 删除服务器端二进制文件
	rm -rf ./lastversion  # 删除lastversion目录
