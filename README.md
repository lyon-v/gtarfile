# GTarFile - Go语言TAR文件处理库

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.19-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

一个高性能、线程安全的Go语言TAR文件处理库，完整模仿Python tarfile模块的API和功能。

## ✨ 特性

- 🔒 **线程安全** - 完整的并发保护，支持多goroutine安全访问
- 📦 **功能完整** - 支持创建、读取、提取TAR文件的所有核心功能
- 🛡️ **类型安全** - 严格的类型检查和错误处理
- ⚡ **高性能** - 优化的文件I/O操作和内存管理
- 📏 **标准兼容** - 完全符合POSIX TAR格式标准
- 🔧 **易于使用** - 简洁直观的API设计
- 🗜️ **压缩支持** - 支持gzip、bzip2、xz压缩格式

## 🎯 使用场景

### 备份和归档
- 系统文件备份
- 日志文件归档
- 数据库备份压缩

### 软件分发
- 应用程序打包
- 依赖库分发
- 容器镜像构建

### 数据传输
- 批量文件传输
- 网络文件同步
- 云存储上传下载

### 开发工具
- 构建系统集成
- CI/CD流水线
- 自动化部署脚本

## 🚀 快速开始

### 安装

```bash
go get github.com/yourusername/gtarfile
```

### 基本使用

```go
package main

import (
    "fmt"
    "log"
    "gtarfile/tarfile"
)

func main() {
    // 创建TAR文件
    tf, err := tarfile.Open("archive.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 添加文件到归档
    err = tf.Add("myfile.txt", "", false, nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("TAR文件创建成功!")
}
```

## 📚 核心功能

### 1. TAR文件创建
- 创建新的TAR归档文件
- 添加文件和目录到归档
- 支持递归添加目录结构
- 自定义文件过滤器

### 2. TAR文件读取
- 读取现有TAR文件
- 遍历归档成员
- 获取文件元数据信息
- 流式读取支持

### 3. TAR文件提取
- 提取单个文件
- 批量提取所有文件
- 保持文件权限和时间戳
- 支持符号链接和硬链接

### 4. 压缩格式支持
- `.tar` - 无压缩TAR文件
- `.tar.gz` / `.tgz` - Gzip压缩
- `.tar.bz2` - Bzip2压缩  
- `.tar.xz` - XZ压缩

### 5. 高级特性
- PAX扩展头支持
- GNU TAR格式兼容
- 稀疏文件处理
- 大文件支持（>8GB）

## 🔧 API参考

### 主要类型

```go
// TarFile - TAR文件操作对象
type TarFile struct {
    // ... 私有字段
}

// TarInfo - TAR文件成员信息
type TarInfo struct {
    Name     string    // 文件名
    Size     int64     // 文件大小
    Mode     int64     // 权限模式
    Mtime    time.Time // 修改时间
    Type     string    // 文件类型
    // ... 其他字段
}
```

### 主要方法

```go
// 打开TAR文件
func Open(name, mode string, fileobj io.ReadWriteSeeker, bufsize int) (*TarFile, error)

// 添加文件到归档
func (tf *TarFile) Add(name, arcname string, recursive bool, filter func(*TarInfo) (*TarInfo, error)) error

// 获取所有成员
func (tf *TarFile) GetMembers() ([]*TarInfo, error)

// 提取文件
func (tf *TarFile) Extract(member *TarInfo, path string) error

// 提取所有文件
func (tf *TarFile) ExtractAll(path string) error
```

## 📖 详细文档

更多详细使用案例和API文档请查看 [docs](./docs/) 目录：

- [安装指南](./docs/installation.md)
- [基础教程](./docs/basic-tutorial.md)
- [高级用法](./docs/advanced-usage.md)
- [API文档](./docs/api-reference.md)
- [性能优化](./docs/performance.md)

## 🤝 贡献

欢迎提交Issue和Pull Request来帮助改进这个项目！

1. Fork 这个仓库
2. 创建你的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交你的修改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启一个Pull Request

## 📄 许可证

这个项目使用 MIT 许可证。查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- 灵感来源于Python的 [tarfile](https://docs.python.org/3/library/tarfile.html) 模块
- 感谢Go语言社区的支持和贡献

---

**如果这个项目对您有帮助，请给个⭐️支持一下！**
