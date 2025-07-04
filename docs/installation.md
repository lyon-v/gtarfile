# 安装指南

本指南将帮助您安装和配置GTarFile库。

## 系统要求

- Go版本 >= 1.19
- 支持的操作系统：Linux、macOS、Windows
- 推荐内存：>=512MB

## 安装方式

### 方式一：使用go get（推荐）

```bash
go get github.com/yourusername/gtarfile
```

### 方式二：从源码安装

1. 克隆仓库：
```bash
git clone https://github.com/yourusername/gtarfile.git
cd gtarfile
```

2. 构建项目：
```bash
go build .
```

3. 运行测试：
```bash
go test ./...
```

### 方式三：使用Go Modules

在您的项目中的`go.mod`文件中添加：

```go
module yourproject

go 1.19

require (
	github.com/yourusername/gtarfile v1.0.0
)
```

然后运行：
```bash
go mod download
```

## 依赖项

GTarFile依赖以下第三方库：

- `github.com/ulikunitz/xz` - XZ压缩支持
- `golang.org/x/sys` - 系统调用支持

这些依赖会在安装时自动下载。

## 验证安装

创建一个简单的测试文件来验证安装：

```go
// test_installation.go
package main

import (
	"fmt"
	"log"
	"gtarfile/tarfile"
)

func main() {
	// 创建一个测试TAR文件
	tf, err := tarfile.Open("test.tar", "w", nil, 4096)
	if err != nil {
		log.Fatal("安装验证失败:", err)
	}
	defer tf.Close()
	
	fmt.Println("✅ GTarFile安装成功！")
}
```

运行验证：
```bash
go run test_installation.go
```

如果看到"✅ GTarFile安装成功！"消息，说明安装成功。

## 常见问题

### Q: 安装时提示"模块未找到"
A: 确保您的Go版本>=1.19，并且启用了Go Modules：
```bash
go env GO111MODULE
# 应该返回 "on" 或 "auto"
```

### Q: 编译错误"package not found"
A: 确保import路径正确，如果是本地开发，请使用相对路径。

### Q: XZ压缩功能不可用
A: 确保网络连接正常，第三方依赖可以正常下载。

## 开发环境配置

### IDE配置
推荐使用以下IDE进行开发：
- GoLand
- VS Code with Go extension
- Vim with vim-go

### 代码格式化
项目使用标准Go格式化工具：
```bash
go fmt ./...
```

### 代码检查
使用golint检查代码质量：
```bash
golint ./...
```

## 下一步

安装完成后，请查看：
- [基础教程](./basic-tutorial.md) - 学习基本用法