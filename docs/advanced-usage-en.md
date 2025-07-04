# Advanced Usage

This guide covers advanced features and usage patterns of the GTarFile library for complex scenarios.

## Table of Contents

1. [Concurrent Processing](#concurrent-processing)
2. [Custom File Filters](#custom-file-filters)
3. [Streaming Operations](#streaming-operations)
4. [PAX Extended Headers](#pax-extended-headers)
5. [Compression Handling](#compression-handling)
6. [Error Recovery](#error-recovery)
7. [Performance Optimization](#performance-optimization)
8. [Security Considerations](#security-considerations)

## Concurrent Processing

### Thread-Safe Operations

The GTarFile library is designed to be thread-safe. Here's how to use it in concurrent scenarios:

```go
package main

import (
    "fmt"
    "log"
    "sync"
    "gtarfile/tarfile"
)

func concurrentArchiveCreation() {
    var wg sync.WaitGroup
    const numWorkers = 3

    // Create multiple archives concurrently
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            filename := fmt.Sprintf("archive_%d.tar", workerID)
            tf, err := tarfile.Open(filename, "w", nil, 4096)
            if err != nil {
                log.Printf("Worker %d failed to create archive: %v", workerID, err)
                return
            }
            defer tf.Close()

            // Add files specific to this worker
            for j := 0; j < 5; j++ {
                testFile := fmt.Sprintf("test_file_%d_%d.txt", workerID, j)
                err = tf.Add(testFile, "", false, nil)
                if err != nil {
                    log.Printf("Worker %d failed to add file %s: %v", workerID, testFile, err)
                }
            }
            
            fmt.Printf("Worker %d completed archive %s\n", workerID, filename)
        }(i)
    }

    wg.Wait()
    fmt.Println("All archives created successfully!")
}
```

### Concurrent Reading

```go
func concurrentReading() {
    var wg sync.WaitGroup
    archives := []string{"archive1.tar", "archive2.tar", "archive3.tar"}

    for _, archive := range archives {
        wg.Add(1)
        go func(archiveName string) {
            defer wg.Done()
            
            tf, err := tarfile.Open(archiveName, "r", nil, 4096)
            if err != nil {
                log.Printf("Failed to open %s: %v", archiveName, err)
                return
            }
            defer tf.Close()

            members, err := tf.GetMembers()
            if err != nil {
                log.Printf("Failed to get members from %s: %v", archiveName, err)
                return
            }

            fmt.Printf("Archive %s contains %d files\n", archiveName, len(members))
        }(archive)
    }

    wg.Wait()
}
```

## Custom File Filters

### Basic File Filtering

```go
func createFilteredArchive() {
    tf, err := tarfile.Open("filtered.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // Custom filter function
    filter := func(tarinfo *tarfile.TarInfo) (*tarfile.TarInfo, error) {
        // Skip hidden files
        if strings.HasPrefix(tarinfo.Name, ".") {
            return nil, nil // Skip this file
        }
        
        // Skip files larger than 10MB
        if tarinfo.Size > 10*1024*1024 {
            fmt.Printf("Skipping large file: %s (%d bytes)\n", tarinfo.Name, tarinfo.Size)
            return nil, nil
        }
        
        // Rename files with specific pattern
        if strings.HasSuffix(tarinfo.Name, ".tmp") {
            tarinfo.Name = strings.TrimSuffix(tarinfo.Name, ".tmp") + ".backup"
        }
        
        return tarinfo, nil
    }

    // Add directory with filter
    err = tf.Add("src/", "", true, filter)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Advanced Filtering with Metadata

```go
import (
    "path/filepath"
    "strings"
    "time"
)

func advancedFiltering() {
    tf, err := tarfile.Open("advanced_filtered.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    filter := func(tarinfo *tarfile.TarInfo) (*tarfile.TarInfo, error) {
        // Skip files older than 30 days
        if time.Since(tarinfo.Mtime) > 30*24*time.Hour {
            return nil, nil
        }
        
        // Only include specific file types
        ext := filepath.Ext(tarinfo.Name)
        allowedExt := []string{".go", ".txt", ".md", ".json"}
        
        allowed := false
        for _, allowed_ext := range allowedExt {
            if ext == allowed_ext {
                allowed = true
                break
            }
        }
        
        if !allowed {
            return nil, nil
        }
        
        // Flatten directory structure for certain files
        if strings.Contains(tarinfo.Name, "config/") {
            tarinfo.Name = filepath.Base(tarinfo.Name)
        }
        
        return tarinfo, nil
    }

    err = tf.Add("project/", "", true, filter)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Streaming Operations

### Streaming Archive Creation

```go
import (
    "bytes"
    "io"
)

func streamingCreation() {
    // Create archive in memory
    var buf bytes.Buffer
    
    tf, err := tarfile.Open("", "w", &buf, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // Add files to the stream
    files := []string{"file1.txt", "file2.txt", "config.json"}
    for _, file := range files {
        err = tf.Add(file, "", false, nil)
        if err != nil {
            log.Printf("Failed to add %s: %v", file, err)
        }
    }

    // Now buf contains the TAR archive data
    fmt.Printf("Archive size: %d bytes\n", buf.Len())
    
    // You can write this to a file, network connection, etc.
    err = os.WriteFile("streamed.tar", buf.Bytes(), 0644)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Streaming Extraction

```go
func streamingExtraction() {
    // Read archive from memory
    data, err := os.ReadFile("example.tar")
    if err != nil {
        log.Fatal(err)
    }
    
    buf := bytes.NewReader(data)
    tf, err := tarfile.Open("", "r", buf, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // Stream processing
    for {
        member, err := tf.Next()
        if err != nil {
            if err.Error() == "EOF" {
                break
            }
            log.Fatal(err)
        }

        fmt.Printf("Processing: %s\n", member.Name)
        
        // Custom processing for each file
        if strings.HasSuffix(member.Name, ".txt") {
            // Process text files differently
            err = tf.Extract(member, "text_files/")
        } else {
            // Process other files
            err = tf.Extract(member, "other_files/")
        }
        
        if err != nil {
            log.Printf("Failed to extract %s: %v", member.Name, err)
        }
    }
}
```

## PAX Extended Headers

### Working with Extended Attributes

```go
func paxExtendedHeaders() {
    tf, err := tarfile.Open("pax_extended.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // Create a file with extended attributes
    member := &tarfile.TarInfo{
        Name: "extended_file.txt",
        Size: 1024,
        Mode: 0644,
        Mtime: time.Now(),
        Type: "0", // Regular file
        Pax: map[string]string{
            "custom.field1": "custom_value_1",
            "custom.field2": "custom_value_2",
            "comment": "This file has extended attributes",
        },
    }

    // Add file with extended headers
    err = tf.AddTarInfo(member)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("File with PAX extended headers added")
}
```

### Reading Extended Headers

```go
func readPaxHeaders() {
    tf, err := tarfile.Open("pax_extended.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal(err)
    }

    for _, member := range members {
        fmt.Printf("File: %s\n", member.Name)
        
        if member.Pax != nil && len(member.Pax) > 0 {
            fmt.Println("Extended attributes:")
            for key, value := range member.Pax {
                fmt.Printf("  %s: %s\n", key, value)
            }
        }
        fmt.Println("---")
    }
}
```

## Compression Handling

### Multiple Compression Formats

```go
func multipleCompressionFormats() {
    formats := map[string]string{
        "archive.tar.gz":  "w:gz",
        "archive.tar.bz2": "w:bz2", 
        "archive.tar.xz":  "w:xz",
    }

    for filename, mode := range formats {
        tf, err := tarfile.Open(filename, mode, nil, 4096)
        if err != nil {
            log.Printf("Failed to create %s: %v", filename, err)
            continue
        }

        err = tf.Add("README.md", "", false, nil)
        if err != nil {
            log.Printf("Failed to add file to %s: %v", filename, err)
        }

        tf.Close()
        fmt.Printf("Created compressed archive: %s\n", filename)
    }
}
```

### Compression Level Control

```go
func compressionWithLevels() {
    // Custom compression settings would need to be implemented
    // in the underlying compression libraries
    
    tf, err := tarfile.Open("high_compression.tar.gz", "w:gz", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // Add large files that benefit from compression
    largeFiles := []string{"large_data.csv", "logs.txt", "database_dump.sql"}
    
    for _, file := range largeFiles {
        err = tf.Add(file, "", false, nil)
        if err != nil {
            log.Printf("Failed to add %s: %v", file, err)
        }
    }
}
```

## Error Recovery

### Robust Error Handling

```go
func robustArchiveProcessing() {
    tf, err := tarfile.Open("potentially_corrupt.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    var successCount, errorCount int

    for {
        member, err := tf.Next()
        if err != nil {
            if err.Error() == "EOF" {
                break
            }
            
            errorCount++
            log.Printf("Error reading member: %v", err)
            
            // Try to recover by skipping corrupted entry
            continue
        }

        // Attempt to extract with error recovery
        err = tf.Extract(member, "recovered/")
        if err != nil {
            errorCount++
            log.Printf("Failed to extract %s: %v", member.Name, err)
            
            // Try alternative extraction methods
            err = alternativeExtraction(tf, member)
            if err != nil {
                log.Printf("Alternative extraction also failed for %s", member.Name)
                continue
            }
        }
        
        successCount++
    }

    fmt.Printf("Processing completed: %d successful, %d errors\n", successCount, errorCount)
}

func alternativeExtraction(tf *tarfile.TarFile, member *tarfile.TarInfo) error {
    // Implement alternative extraction logic
    // For example, extract with different permissions or to different location
    return tf.ExtractTo(member, fmt.Sprintf("alternative/%s", member.Name))
}
```

## Performance Optimization

### Buffer Size Optimization

```go
func optimizedBuffering() {
    // Different buffer sizes for different file types
    configs := map[string]int{
        "small_files.tar":  4096,   // 4KB for small files
        "medium_files.tar": 32768,  // 32KB for medium files  
        "large_files.tar":  131072, // 128KB for large files
    }

    for filename, bufsize := range configs {
        tf, err := tarfile.Open(filename, "w", nil, bufsize)
        if err != nil {
            log.Printf("Failed to create %s: %v", filename, err)
            continue
        }

        // Add appropriate files based on the archive type
        switch filename {
        case "small_files.tar":
            addSmallFiles(tf)
        case "medium_files.tar":
            addMediumFiles(tf)
        case "large_files.tar":
            addLargeFiles(tf)
        }

        tf.Close()
    }
}

func addSmallFiles(tf *tarfile.TarFile) {
    // Add files < 1MB
    files := []string{"config.json", "README.md", "LICENSE"}
    for _, file := range files {
        tf.Add(file, "", false, nil)
    }
}

func addMediumFiles(tf *tarfile.TarFile) {
    // Add files 1MB - 100MB
    files := []string{"binary", "images/photo.jpg", "docs/manual.pdf"}
    for _, file := range files {
        tf.Add(file, "", false, nil)
    }
}

func addLargeFiles(tf *tarfile.TarFile) {
    // Add files > 100MB
    files := []string{"database.db", "video.mp4", "large_dataset.csv"}
    for _, file := range files {
        tf.Add(file, "", false, nil)
    }
}
```

## Security Considerations

### Path Traversal Protection

```go
import (
    "path/filepath"
    "strings"
)

func secureExtraction() {
    tf, err := tarfile.Open("untrusted.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    extractDir := "./safe_extract/"
    os.MkdirAll(extractDir, 0755)

    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal(err)
    }

    for _, member := range members {
        // Validate and sanitize paths
        if isPathTraversal(member.Name) {
            log.Printf("Skipping potentially dangerous path: %s", member.Name)
            continue
        }

        // Clean the path
        cleanPath := filepath.Clean(member.Name)
        if strings.HasPrefix(cleanPath, "..") {
            log.Printf("Skipping path traversal attempt: %s", member.Name)
            continue
        }

        // Extract to safe location
        safePath := filepath.Join(extractDir, cleanPath)
        err = tf.ExtractTo(member, safePath)
        if err != nil {
            log.Printf("Failed to extract %s: %v", member.Name, err)
        }
    }
}

func isPathTraversal(path string) bool {
    // Check for common path traversal patterns
    dangerous := []string{
        "../",
        "..\\",
        "/etc/",
        "/usr/",
        "/var/",
        "\\Windows\\",
        "\\System32\\",
    }

    for _, pattern := range dangerous {
        if strings.Contains(path, pattern) {
            return true
        }
    }
    return false
}
```

### File Size Limits

```go
func extractWithLimits() {
    tf, err := tarfile.Open("large.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    const maxFileSize = 100 * 1024 * 1024 // 100MB limit
    const maxTotalSize = 1024 * 1024 * 1024 // 1GB total limit
    
    var totalExtracted int64

    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal(err)
    }

    for _, member := range members {
        // Check individual file size
        if member.Size > maxFileSize {
            log.Printf("Skipping oversized file: %s (%d bytes)", member.Name, member.Size)
            continue
        }

        // Check total extracted size
        if totalExtracted+member.Size > maxTotalSize {
            log.Printf("Reached total size limit, stopping extraction")
            break
        }

        err = tf.Extract(member, "./limited_extract/")
        if err != nil {
            log.Printf("Failed to extract %s: %v", member.Name, err)
            continue
        }

        totalExtracted += member.Size
        fmt.Printf("Extracted: %s (%d bytes total)\n", member.Name, totalExtracted)
    }
} 