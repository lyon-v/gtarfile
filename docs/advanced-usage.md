# 高级用法

本文档介绍GTarFile库的高级特性和复杂使用场景。

## 目录

- [并发处理](#并发处理)
- [自定义过滤器](#自定义过滤器)
- [流式处理](#流式处理)
- [PAX扩展头](#pax扩展头)
- [稀疏文件处理](#稀疏文件处理)
- [错误恢复策略](#错误恢复策略)
- [内存优化](#内存优化)

## 并发处理

GTarFile库是线程安全的，支持并发操作。

### 1. 并发读取

```go
package main

import (
    "fmt"
    "log"
    "sync"
    "gtarfile/tarfile"
)

func concurrentRead() {
    tf, err := tarfile.Open("large.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    var wg sync.WaitGroup
    
    // 启动多个goroutine并发读取
    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            // 每个goroutine可以安全地调用GetMembers
            members, err := tf.GetMembers()
            if err != nil {
                log.Printf("Goroutine %d: 错误 %v", id, err)
                return
            }
            
            fmt.Printf("Goroutine %d: 找到 %d 个文件\n", id, len(members))
        }(i)
    }
    
    wg.Wait()
}
```

### 2. 并发提取

```go
func concurrentExtract() {
    tf, err := tarfile.Open("archive.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal(err)
    }

    var wg sync.WaitGroup
    semaphore := make(chan struct{}, 3) // 限制并发数为3

    for _, member := range members {
        wg.Add(1)
        go func(m *tarfile.TarInfo) {
            defer wg.Done()
            semaphore <- struct{}{} // 获取信号量
            defer func() { <-semaphore }() // 释放信号量

            err := tf.Extract(m, "concurrent_output")
            if err != nil {
                log.Printf("提取 %s 失败: %v", m.Name, err)
            } else {
                fmt.Printf("✅ 提取完成: %s\n", m.Name)
            }
        }(member)
    }

    wg.Wait()
}
```

## 自定义过滤器

使用过滤器可以控制哪些文件被添加到归档中。

### 1. 基本过滤器

```go
func basicFilter() {
    tf, err := tarfile.Open("filtered.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 只包含.txt文件的过滤器
    textFilter := func(ti *tarfile.TarInfo) (*tarfile.TarInfo, error) {
        if strings.HasSuffix(ti.Name, ".txt") {
            return ti, nil // 保留文件
        }
        return nil, nil // 跳过文件
    }

    // 递归添加目录，应用过滤器
    err = tf.Add("source_dir", "", true, textFilter)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. 复杂过滤器

```go
func complexFilter() {
    tf, err := tarfile.Open("complex_filtered.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 复杂过滤逻辑
    complexFilter := func(ti *tarfile.TarInfo) (*tarfile.TarInfo, error) {
        // 跳过隐藏文件
        if strings.HasPrefix(filepath.Base(ti.Name), ".") {
            return nil, nil
        }

        // 跳过大文件（>10MB）
        if ti.Size > 10*1024*1024 {
            log.Printf("跳过大文件: %s (%d bytes)", ti.Name, ti.Size)
            return nil, nil
        }

        // 修改权限
        if ti.IsReg() {
            ti.Mode = 0644
        }

        // 重命名文件
        if strings.HasSuffix(ti.Name, ".log") {
            ti.Name = strings.Replace(ti.Name, ".log", ".txt", 1)
        }

        return ti, nil
    }

    err = tf.Add("source_dir", "", true, complexFilter)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 3. 安全过滤器

```go
func securityFilter() {
    tf, err := tarfile.Open("secure.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 安全过滤器：防止路径穿越攻击
    secureFilter := func(ti *tarfile.TarInfo) (*tarfile.TarInfo, error) {
        // 检查路径穿越
        if strings.Contains(ti.Name, "..") {
            return nil, fmt.Errorf("检测到路径穿越: %s", ti.Name)
        }

        // 规范化路径
        ti.Name = filepath.Clean(ti.Name)
        ti.Name = strings.TrimPrefix(ti.Name, "/")

        // 限制路径长度
        if len(ti.Name) > 255 {
            return nil, fmt.Errorf("路径过长: %s", ti.Name)
        }

        return ti, nil
    }

    err = tf.Add("source_dir", "", true, secureFilter)
    if err != nil {
        log.Fatal(err)
    }
}
```

## 流式处理

对于大文件或内存受限的环境，可以使用流式处理。

### 1. 流式创建

```go
func streamingCreate() {
    tf, err := tarfile.Open("streaming.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 模拟大数据流
    data := generateLargeData() // 假设这个函数生成大量数据
    
    ti := tarfile.NewTarInfo("large_data.bin")
    ti.Size = int64(len(data))
    
    reader := bytes.NewReader(data)
    err = tf.AddFile(ti, reader)
    if err != nil {
        log.Fatal(err)
    }
}

func generateLargeData() []byte {
    // 生成1MB的测试数据
    data := make([]byte, 1024*1024)
    for i := range data {
        data[i] = byte(i % 256)
    }
    return data
}
```

### 2. 流式读取

```go
func streamingRead() {
    tf, err := tarfile.Open("streaming.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 逐个处理文件，不加载所有到内存
    for {
        ti, err := tf.Next()
        if err != nil {
            break
        }
        if ti == nil {
            break
        }

        fmt.Printf("处理文件: %s (大小: %d)\n", ti.Name, ti.Size)
        
        // 可以在这里处理文件内容，而不需要全部加载到内存
        processFileStream(tf, ti)
    }
}

func processFileStream(tf *tarfile.TarFile, ti *tarfile.TarInfo) {
    // 创建文件对象来读取内容
    fileObj := tf.fileObject(tf, ti)
    
    // 分块读取
    buffer := make([]byte, 8192)
    totalRead := int64(0)
    
    for totalRead < ti.Size {
        n, err := fileObj.Read(buffer)
        if err != nil && err != io.EOF {
            log.Printf("读取错误: %v", err)
            break
        }
        
        totalRead += int64(n)
        
        // 在这里处理数据块
        processDataChunk(buffer[:n])
        
        if err == io.EOF {
            break
        }
    }
}

func processDataChunk(data []byte) {
    // 处理数据块的逻辑
    fmt.Printf("处理了 %d 字节数据\n", len(data))
}
```

## PAX扩展头

PAX格式支持扩展属性和长文件名。

### 1. 使用PAX头

```go
func usePaxHeaders() {
    tf, err := tarfile.Open("pax.tar", "w", nil, 4096,
        tarfile.WithFormat(tarfile.PAX_FORMAT))
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    ti := tarfile.NewTarInfo("example.txt")
    ti.Size = 13
    
    // 添加自定义PAX属性
    ti.PaxHeaders["custom.author"] = "GTarFile User"
    ti.PaxHeaders["custom.description"] = "示例文件"
    ti.PaxHeaders["custom.created"] = time.Now().Format(time.RFC3339)

    content := strings.NewReader("Hello, World!")
    err = tf.AddFile(ti, content)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. 读取PAX头

```go
func readPaxHeaders() {
    tf, err := tarfile.Open("pax.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal(err)
    }

    for _, member := range members {
        fmt.Printf("文件: %s\n", member.Name)
        
        // 打印PAX扩展属性
        for key, value := range member.PaxHeaders {
            fmt.Printf("  %s: %s\n", key, value)
        }
    }
}
```

## 稀疏文件处理

GTarFile支持稀疏文件的高效存储。

### 1. 创建稀疏文件

```go
func createSparseFile() {
    tf, err := tarfile.Open("sparse.tar", "w", nil, 4096,
        tarfile.WithFormat(tarfile.GNU_FORMAT))
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    ti := tarfile.NewTarInfo("sparse.bin")
    ti.Size = 1024 * 1024 // 1MB文件
    
    // 定义稀疏区域：只有前1KB和最后1KB有数据
    ti.Sparse = [][2]int64{
        {0, 1024},           // 开始位置：0，大小：1KB
        {1023 * 1024, 1024}, // 开始位置：1023KB，大小：1KB
    }

    // 创建稀疏数据
    data := make([]byte, 2048) // 只需要2KB实际数据
    for i := range data {
        data[i] = byte(i % 256)
    }

    err = tf.AddFile(ti, bytes.NewReader(data))
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. 检测稀疏文件

```go
func detectSparseFiles() {
    tf, err := tarfile.Open("sparse.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal(err)
    }

    for _, member := range members {
        if member.IsSparse() {
            fmt.Printf("稀疏文件: %s\n", member.Name)
            fmt.Printf("  逻辑大小: %d\n", member.Size)
            
            realSize := int64(0)
            for _, sparse := range member.Sparse {
                realSize += sparse[1] // 累加实际数据大小
            }
            fmt.Printf("  实际大小: %d\n", realSize)
            fmt.Printf("  压缩比: %.2f%%\n", 
                float64(realSize)/float64(member.Size)*100)
        }
    }
}
```

## 错误恢复策略

实现健壮的错误处理和恢复机制。

### 1. 重试机制

```go
func retryOperation() {
    maxRetries := 3
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        err := performTarOperation()
        if err == nil {
            break // 成功
        }
        
        log.Printf("尝试 %d 失败: %v", attempt+1, err)
        
        if attempt < maxRetries-1 {
            // 等待后重试
            time.Sleep(time.Second * time.Duration(attempt+1))
        } else {
            log.Fatal("所有重试都失败了")
        }
    }
}

func performTarOperation() error {
    tf, err := tarfile.Open("unstable.tar", "r", nil, 4096)
    if err != nil {
        return fmt.Errorf("打开文件失败: %w", err)
    }
    defer tf.Close()

    _, err = tf.GetMembers()
    return err
}
```

### 2. 部分失败处理

```go
func partialFailureHandling() {
    tf, err := tarfile.Open("partial.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    files := []string{"file1.txt", "file2.txt", "nonexistent.txt"}
    successCount := 0
    
    for _, filename := range files {
        err := tf.Add(filename, "", false, nil)
        if err != nil {
            log.Printf("警告: 无法添加文件 %s: %v", filename, err)
        } else {
            successCount++
            log.Printf("成功添加: %s", filename)
        }
    }
    
    log.Printf("总计: %d/%d 文件成功添加", successCount, len(files))
}
```

## 内存优化

针对大文件和内存受限环境的优化策略。

### 1. 缓冲区管理

```go
func optimizedBuffering() {
    // 为大文件使用较大的缓冲区
    bufferSize := 64 * 1024 // 64KB
    
    tf, err := tarfile.Open("optimized.tar", "w", nil, bufferSize)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 设置调试级别来监控性能
    tf.SetDebug(1)
    
    // 添加大文件
    err = tf.Add("large_file.bin", "", false, nil)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. 内存监控

```go
func memoryMonitoring() {
    var m runtime.MemStats
    
    runtime.ReadMemStats(&m)
    fmt.Printf("开始内存使用: %d KB\n", m.Alloc/1024)
    
    tf, err := tarfile.Open("memory_test.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // 添加多个文件
    for i := 0; i < 100; i++ {
        ti := tarfile.NewTarInfo(fmt.Sprintf("file_%d.txt", i))
        ti.Size = 1024
        
        data := make([]byte, 1024)
        err = tf.AddFile(ti, bytes.NewReader(data))
        if err != nil {
            log.Fatal(err)
        }
        
        if i%10 == 0 {
            runtime.ReadMemStats(&m)
            fmt.Printf("处理 %d 文件后内存: %d KB\n", i, m.Alloc/1024)
        }
    }
    
    runtime.ReadMemStats(&m)
    fmt.Printf("完成后内存使用: %d KB\n", m.Alloc/1024)
}
```

### 3. 资源清理

```go
func resourceCleanup() {
    tf, err := tarfile.Open("cleanup.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    
    // 使用defer确保资源清理
    defer func() {
        if err := tf.Close(); err != nil {
            log.Printf("关闭TAR文件时出错: %v", err)
        }
        
        // 强制垃圾回收
        runtime.GC()
    }()

    // 处理文件...
}
```
