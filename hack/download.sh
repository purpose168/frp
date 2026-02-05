#!/bin/sh

OS="$(go env GOOS)"
ARCH="$(go env GOARCH)"

if [ "${TARGET_OS}" ]; then
  OS="${TARGET_OS}"
fi
if [ "${TARGET_ARCH}" ]; then
  ARCH="${TARGET_ARCH}"
fi

# 通过版本号确定最新版本，忽略 alpha、beta 和 rc 版本。
if [ "${FRP_VERSION}" = "" ] ; then
  FRP_VERSION="$(curl -sL https://github.com/purpose168/frp/releases | \
                  grep -o 'releases/tag/v[0-9]*.[0-9]*.[0-9]*"' | sort -V | \
                  tail -1 | awk -F'/' '{ print $3}')"
  FRP_VERSION="${FRP_VERSION%?}"
  FRP_VERSION="${FRP_VERSION#?}"
fi

if [ "${FRP_VERSION}" = "" ] ; then
  printf "无法获取最新的 frp 版本。请设置 FRP_VERSION 环境变量并重新运行。例如：export FRP_VERSION=1.0.0"
  exit 1;
fi

SUFFIX=".tar.gz"
if [ "${OS}" = "windows" ] ; then
  SUFFIX=".zip"
fi
NAME="frp_${FRP_VERSION}_${OS}_${ARCH}${SUFFIX}"
DIR_NAME="frp_${FRP_VERSION}_${OS}_${ARCH}"
URL="https://github.com/purpose168/frp/releases/download/v${FRP_VERSION}/${NAME}"

download_and_extract() {
  printf "正在从 %s 下载 %s ...\n" "${URL}" "$NAME"
  if ! curl -o /dev/null -sIf "${URL}"; then
    printf "\n未找到 %s，请指定有效的 FRP_VERSION\n" "${URL}"
    exit 1
  fi
  curl -fsLO "${URL}"
  filename=$NAME

  if [ "${OS}" = "windows" ]; then
    unzip "${filename}"
  else
    tar -xzf "${filename}"
  fi
  rm "${filename}"

  if [ "${TARGET_DIRNAME}" ]; then
    mv "${DIR_NAME}" "${TARGET_DIRNAME}"
    DIR_NAME="${TARGET_DIRNAME}"
  fi
}

download_and_extract

printf ""
printf "\nfrp %s 下载完成！\n" "$FRP_VERSION"
printf "\n"
printf "frp 已成功下载到您系统上的 %s 文件夹中。\n" "$DIR_NAME"
printf "\n"
