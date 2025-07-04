# Basic Tutorial

This tutorial will guide you through the basic usage of the GTarFile library, covering the most common TAR file operations.

## Prerequisites

- Go 1.19 or higher installed
- GTarFile library installed (see [Installation Guide](./installation-en.md))
- Basic knowledge of Go programming

## 1. Creating TAR Files

### Simple File Creation

```go
package main

import (
    "fmt"
    "log"
    "gtarfile/tarfile"
)

func createBasicTar() {
    // Create a new TAR file
    tf, err := tarfile.Open("example.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal("Failed to create TAR file:", err)
    }
    defer tf.Close()

    // Add a single file
    err = tf.Add("README.md", "", false, nil)
    if err != nil {
        log.Fatal("Failed to add file:", err)
    }

    fmt.Println("TAR file created successfully!")
}
```

### Adding Multiple Files

```go
func createMultiFileTar() {
    tf, err := tarfile.Open("multi.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // List of files to add
    files := []string{
        "file1.txt",
        "file2.txt", 
        "config.json",
    }

    for _, file := range files {
        err = tf.Add(file, "", false, nil)
        if err != nil {
            log.Printf("Warning: Failed to add %s: %v", file, err)
            continue
        }
        fmt.Printf("Added: %s\n", file)
    }
}
```

### Adding Directories Recursively

```go
func createDirectoryTar() {
    tf, err := tarfile.Open("directory.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // Add entire directory recursively
    err = tf.Add("src/", "", true, nil)
    if err != nil {
        log.Fatal("Failed to add directory:", err)
    }

    fmt.Println("Directory added successfully!")
}
```

## 2. Reading TAR Files

### Listing Archive Contents

```go
func listTarContents() {
    // Open existing TAR file for reading
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal("Failed to open TAR file:", err)
    }
    defer tf.Close()

    // Get all members
    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal("Failed to get members:", err)
    }

    fmt.Println("TAR file contents:")
    fmt.Println("==================")
    for _, member := range members {
        fmt.Printf("Name: %s\n", member.Name)
        fmt.Printf("Size: %d bytes\n", member.Size)
        fmt.Printf("Mode: %o\n", member.Mode)
        fmt.Printf("Modified: %s\n", member.Mtime.Format("2006-01-02 15:04:05"))
        fmt.Printf("Type: %s\n", member.Type)
        fmt.Println("------------------")
    }
}
```

### Iterating Through Members

```go
func iterateTarMembers() {
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // Iterate through each member
    for {
        member, err := tf.Next()
        if err != nil {
            if err.Error() == "EOF" {
                break // End of archive
            }
            log.Fatal("Error reading member:", err)
        }

        fmt.Printf("Processing: %s\n", member.Name)
        
        // Process member based on type
        switch member.Type {
        case "0": // Regular file
            fmt.Printf("  Regular file, size: %d\n", member.Size)
        case "5": // Directory
            fmt.Printf("  Directory\n")
        case "2": // Symbolic link
            fmt.Printf("  Symbolic link\n")
        default:
            fmt.Printf("  Other type: %s\n", member.Type)
        }
    }
}
```

## 3. Extracting Files

### Extract Single File

```go
func extractSingleFile() {
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // Find specific file
    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal(err)
    }

    for _, member := range members {
        if member.Name == "README.md" {
            // Extract to current directory
            err = tf.Extract(member, ".")
            if err != nil {
                log.Fatal("Failed to extract file:", err)
            }
            fmt.Printf("Extracted: %s\n", member.Name)
            break
        }
    }
}
```

### Extract All Files

```go
func extractAllFiles() {
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // Extract all files to specified directory
    err = tf.ExtractAll("./extracted/")
    if err != nil {
        log.Fatal("Failed to extract all files:", err)
    }

    fmt.Println("All files extracted successfully!")
}
```

### Extract with Custom Path

```go
func extractToCustomPath() {
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal(err)
    }

    // Extract each file to custom location
    for _, member := range members {
        customPath := fmt.Sprintf("./backup/%s", member.Name)
        err = tf.ExtractTo(member, customPath)
        if err != nil {
            log.Printf("Warning: Failed to extract %s: %v", member.Name, err)
            continue
        }
        fmt.Printf("Extracted %s to %s\n", member.Name, customPath)
    }
}
```

## 4. Working with Compressed TAR Files

### Creating Compressed Archives

```go
func createCompressedTar() {
    // Create gzip compressed TAR file
    tf, err := tarfile.Open("archive.tar.gz", "w:gz", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    err = tf.Add("largefile.dat", "", false, nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Compressed TAR file created!")
}
```

### Reading Compressed Archives

```go
func readCompressedTar() {
    // Open gzip compressed TAR file
    tf, err := tarfile.Open("archive.tar.gz", "r:gz", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Compressed archive contains %d files\n", len(members))
}
```

## 5. Error Handling

### Robust Error Handling

```go
func robustTarHandling() {
    tf, err := tarfile.Open("example.tar", "r", nil, 4096)
    if err != nil {
        log.Fatal("Cannot open TAR file:", err)
    }
    defer func() {
        if err := tf.Close(); err != nil {
            log.Printf("Warning: Failed to close TAR file: %v", err)
        }
    }()

    members, err := tf.GetMembers()
    if err != nil {
        log.Fatal("Cannot read TAR contents:", err)
    }

    for _, member := range members {
        err := tf.Extract(member, "./output/")
        if err != nil {
            // Log error but continue with other files
            log.Printf("Failed to extract %s: %v", member.Name, err)
            continue
        }
        fmt.Printf("Successfully extracted: %s\n", member.Name)
    }
}
```

## Best Practices

1. **Always close TAR files**: Use `defer tf.Close()` to ensure proper cleanup
2. **Handle errors appropriately**: Don't ignore errors, log them or handle gracefully
3. **Use appropriate buffer sizes**: Larger buffers for large files (64KB+)
4. **Check file permissions**: Ensure you have read/write permissions for target files
5. **Validate input paths**: Always validate and sanitize file paths before operations

## Common Pitfalls

1. **Not closing files**: Always call `Close()` to avoid resource leaks
2. **Ignoring errors**: Check and handle all errors properly
3. **Buffer size too small**: Use appropriate buffer sizes for performance
4. **Path traversal**: Be careful with archive member paths to avoid security issues

## Next Steps

Now that you've mastered the basics, you can explore:
- [Advanced Usage](./advanced-usage-en.md) - Learn advanced features
