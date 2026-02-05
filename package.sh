#!/bin/sh
set -e  # 遇到错误立即退出脚本

# 编译当前版本的frp
make
if [ $? -ne 0 ]; then
    echo "make error"  # 编译失败时输出错误信息
    exit 1
fi

# 获取当前编译的frp版本号
frp_version=`./bin/frps --version`
echo "build version: $frp_version"  # 输出版本号

# 执行交叉编译，生成多平台二进制文件
make -f ./Makefile.cross-compiles

# 清理并创建packages目录，用于存放打包后的文件
rm -rf ./release/packages
mkdir -p ./release/packages

# 定义需要打包的操作系统、架构和额外参数
os_all='linux windows darwin freebsd openbsd android'
arch_all='386 amd64 arm arm64 mips64 mips64le mips mipsle riscv64 loong64'
extra_all='_ hf'  # _表示默认，hf表示硬浮点

# 进入release目录开始打包
cd ./release

# 遍历所有操作系统、架构和额外参数组合
for os in $os_all; do
    for arch in $arch_all; do
        for extra in $extra_all; do
            suffix="${os}_${arch}"
            if [ "x${extra}" != x"_" ]; then
                suffix="${os}_${arch}_${extra}"  # 添加额外参数到后缀
            fi
            frp_dir_name="frp_${frp_version}_${suffix}"  # 生成目录名
            frp_path="./packages/frp_${frp_version}_${suffix}"  # 生成路径

            # 处理Windows平台
            if [ "x${os}" = x"windows" ]; then
                # 检查二进制文件是否存在
                if [ ! -f "./frpc_${os}_${arch}.exe" ]; then
                    continue
                fi
                if [ ! -f "./frps_${os}_${arch}.exe" ]; then
                    continue
                fi
                # 创建目录并移动二进制文件
                mkdir ${frp_path}
                mv ./frpc_${os}_${arch}.exe ${frp_path}/frpc.exe
                mv ./frps_${os}_${arch}.exe ${frp_path}/frps.exe
            else
                # 处理非Windows平台
                if [ ! -f "./frpc_${suffix}" ]; then
                    continue
                fi
                if [ ! -f "./frps_${suffix}" ]; then
                    continue
                fi
                # 创建目录并移动二进制文件
                mkdir ${frp_path}
                mv ./frpc_${suffix} ${frp_path}/frpc
                mv ./frps_${suffix} ${frp_path}/frps
            fi  
            # 复制LICENSE和配置文件
            cp ../LICENSE ${frp_path}
            cp -f ../conf/frpc.toml ${frp_path}
            cp -f ../conf/frps.toml ${frp_path}

            # 打包
            cd ./packages
            if [ "x${os}" = x"windows" ]; then
                zip -rq ${frp_dir_name}.zip ${frp_dir_name}  # Windows平台使用zip打包
            else
                tar -zcf ${frp_dir_name}.tar.gz ${frp_dir_name}  # 非Windows平台使用tar.gz打包
            fi  
            cd ..
            rm -rf ${frp_path}  # 删除临时目录
        done
    done
done

# 返回上一级目录
cd -
