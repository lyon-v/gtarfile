# 基础教程

本教程将通过实例展示GTarFile库的基本用法。

## 目录

- [创建TAR文件](#创建tar文件)
- [读取TAR文件](#读取tar文件)
- [提取TAR文件](#提取tar文件)
- [压缩格式支持](#压缩格式支持)
- [错误处理](#错误处理)

## 创建TAR文件

### 1. 创建简单的TAR文件

```go
package main

import (
    "log"
    "strings"
    "gtarfile/tarfile"
)

func main() {
    // 创建新的TAR文件
    tf, err := tarfile.Open("example.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 创建文件信息
    ti := tarfile.NewTarInfo("hello.txt")
    ti.Size = 13

    // 添加文件内容
    content := strings.NewReader("Hello, World!")
    err = tf.AddFile(ti, content)
    if err != nil {
        log.Fatal(err)
    }

    log.Println("TAR文件创建成功!")
}
```

### 2. 添加多个文件

```go
func createMultipleFiles() {
    tf, err := tarfile.Open("multi.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 添加文本文件
    ti1 := tarfile.NewTarInfo("file1.txt")
    ti1.Size = 10
    tf.AddFile(ti1, strings.NewReader("Content 1!"))

    // 添加另一个文件
    ti2 := tarfile.NewTarInfo("file2.txt")
    ti2.Size = 10
    tf.AddFile(ti2, strings.NewReader("Content 2!"))

    log.Println("多文件TAR创建成功!")
}
```

### 3. 从现有文件创建归档

```go
func archiveExistingFiles() {
    tf, err := tarfile.Open("archive.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 添加现有文件（假设文件存在）
    err = tf.Add("myfile.txt", "", false, nil)
    if err != nil {
        log.Printf("警告: 无法添加文件: %v", err)
    }

    // 递归添加目录
    err = tf.Add("mydir", "", true, nil)
    if err != nil {
        log.Printf("警告: 无法添加目录: %v", err)
    }
}
```

## 读取TAR文件

### 1. 列出TAR文件内容

```go
func listTarContents() {
    // 打开TAR文件进行读取
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 获取所有成员
    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal(err)
    }

    // 打印文件信息
    fmt.Println("TAR文件内容:")
    for _, member := range members {
        fmt.Printf("- %s (大小: %d 字节, 类型: %s)\n", 
            member.Name, member.Size, member.Type)
    }
}
```

### 2. 逐个读取文件

```go
func readFilesByFile() {
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 逐个读取文件
    for {
        ti, err := tf.Next()
        if err != nil {
            break
        }
        if ti == nil {
            break
        }

        fmt.Printf("找到文件: %s\n", ti.Name)
        
        // 这里可以处理文件内容
        // 注意：如果要读取文件内容，需要使用ExFileObject
    }
}
```

### 3. 获取特定文件

```go
func getSpecificFile() {
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 获取特定文件信息
    member, err := tf.GetMember("hello.txt")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("文件: %s, 大小: %d\n", member.Name, member.Size)
}
```

## 提取TAR文件

### 1. 提取所有文件

```go
func extractAll() {
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 提取到指定目录
    err = tf.ExtractAll("extracted")
    if err != nil {
        log.Fatal(err)
    }

    log.Println("所有文件提取完成!")
}
```

### 2. 提取单个文件

```go
func extractSingleFile() {
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 获取要提取的文件
    member, err := tf.GetMember("hello.txt")
    if err != nil {
        log.Fatal(err)
    }

    // 提取到指定路径
    err = tf.Extract(member, "output")
    if err != nil {
        log.Fatal(err)
    }

    log.Println("文件提取完成!")
}
```

### 3. 使用便利方法提取

```go
func extractWithConvenience() {
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 直接通过文件名提取
    err = tf.ExtractTo("hello.txt", "output")
    if err != nil {
        log.Fatal(err)
    }

    log.Println("便利提取完成!")
}
```

## 压缩格式支持

### 1. 创建压缩TAR文件

```go
func createCompressedTar() {
    // 创建gzip压缩的TAR文件
    tf, err := tarfile.Open("archive.tar.gz", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 添加文件（方法相同）
    ti := tarfile.NewTarInfo("compressed.txt")
    ti.Size = 15
    content := strings.NewReader("Compressed data")
    tf.AddFile(ti, content)

    log.Println("压缩TAR文件创建成功!")
}
```

### 2. 读取压缩TAR文件

```go
func readCompressedTar() {
    // 自动检测压缩格式
    tf, err := tarfile.Open("archive.tar.gz", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 读取方法相同
    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal(err)
    }

    for _, member := range members {
        fmt.Printf("压缩文件中的: %s\n", member.Name)
    }
}
```

## 错误处理

### 1. 常见错误类型

```go
func handleErrors() {
    tf, err := tarfile.Open("nonexistent.tar", "r", nil, 4096)
    if err != nil {
        // 检查错误类型
        switch err.(type) {
        case *tarfile.TarError:
            log.Printf("TAR错误: %v", err)
        case *tarfile.ReadError:
            log.Printf("读取错误: %v", err)
        default:
            log.Printf("其他错误: %v", err)
        }
        return
    }
    defer tf.Close()
}
```

### 2. 优雅的错误处理

```go
func gracefulErrorHandling() {
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Printf("无法打开TAR文件: %v", err)
        return
    }
    defer func() {
        if err := tf.Close(); err != nil {
            log.Printf("关闭文件时出错: %v", err)
        }
    }()

    members, err := tf.GetMembers()
    if err != nil {
        log.Printf("获取成员列表失败: %v", err)
        return
    }

    log.Printf("成功读取%d个文件", len(members))
}
```

## 完整示例

这是一个展示完整工作流程的示例：

```go
package main

import (
    "fmt"
    "log"
    "os"
    "strings"
    "gtarfile/tarfile"
)

func main() {
    // 第1步：创建TAR文件
    createSampleTar()
    
    // 第2步：读取TAR文件
    readSampleTar()
    
    // 第3步：提取TAR文件
    extractSampleTar()
    
    // 第4步：清理
    cleanup()
}

func createSampleTar() {
    tf, err := tarfile.Open("sample.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 添加多个文件
    files := map[string]string{
        "readme.txt": "这是一个示例README文件",
        "config.json": `{"version": "1.0", "debug": true}`,
        "data.csv": "名称,年龄\n张三,25\n李四,30",
    }

    for name, content := range files {
        ti := tarfile.NewTarInfo(name)
        ti.Size = int64(len(content))
        err = tf.AddFile(ti, strings.NewReader(content))
        if err != nil {
            log.Fatal(err)
        }
    }

    fmt.Println("✅ 示例TAR文件创建完成")
}

func readSampleTar() {
    tf, err := tarfile.Open("sample.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\n📁 TAR文件内容:")
    for _, member := range members {
        fmt.Printf("  - %s (%d 字节)\n", member.Name, member.Size)
    }
}

func extractSampleTar() {
    tf, err := tarfile.Open("sample.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    err = tf.ExtractAll("sample_output")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\n📤 文件提取完成到 sample_output/ 目录")
}

func cleanup() {
    os.Remove("sample.tar")
    os.RemoveAll("sample_output")
    fmt.Println("\n🧹 清理完成")
}
```

## 下一步

现在您已经掌握了基本用法，可以查看：
- [高级用法](./advanced-usage.md) - 学习更多高级特性