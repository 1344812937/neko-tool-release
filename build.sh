#!/bin/bash
# build.sh

# 设置变量
APP_NAME="neko-tool"
VERSION=$(sed -n 's/^const AppVersion = "\([^"]*\)"$/\1/p' internal/service/site_info_service.go)

if [ -z "$VERSION" ]; then
	echo "无法从 internal/service/site_info_service.go 读取版本号"
	exit 1
fi

# 清理之前的构建
rm -rf build
mkdir -p build

echo "+ Linux！"
# Linux 版本
GOOS=linux GOARCH=amd64 go build -o "build/${APP_NAME}-linux-amd64-${VERSION}"

echo "+ Windows！"
# Windows 版本
GOOS=windows GOARCH=amd64 go build -o "build/${APP_NAME}-windows-amd64-${VERSION}.exe"

echo "+ macOS！"
# macOS 版本（可选）
GOOS=darwin GOARCH=amd64 go build -o "build/${APP_NAME}-darwin-amd64-${VERSION}"

echo "✅ 构建完成！"
ls -lh build/