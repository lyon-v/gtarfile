# GTarFile - Go TAR File Processing Library

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.19-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()

A high-performance, thread-safe Go TAR file processing library that fully mimics the API and functionality of Python's tarfile module.

## ✨ Features

- 🔒 **Thread-Safe** - Complete concurrency protection with safe multi-goroutine access
- 📦 **Feature Complete** - Supports all core TAR file operations: create, read, extract
- 🛡️ **Type Safe** - Strict type checking and error handling
- ⚡ **High Performance** - Optimized file I/O operations and memory management
- 📏 **Standards Compliant** - Fully compliant with POSIX TAR format standards
- 🔧 **Easy to Use** - Clean and intuitive API design
- 🗜️ **Compression Support** - Supports gzip, bzip2, xz compression formats

## 🎯 Use Cases

### Backup and Archiving
- System file backups
- Log file archiving
- Database backup compression

### Software Distribution
- Application packaging
- Dependency library distribution
- Container image building

### Data Transfer
- Batch file transfers
- Network file synchronization
- Cloud storage upload/download

### Development Tools
- Build system integration
- CI/CD pipelines
- Automated deployment scripts

## 🚀 Quick Start

### Installation

```bash
go get github.com/yourusername/gtarfile
```

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "gtarfile/tarfile"
)

func main() {
    // Create TAR file
    tf, err := tarfile.Open("archive.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal(err)
    }
    defer tf.Close()

    // Add file to archive
    err = tf.Add("myfile.txt", "", false, nil)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("TAR file created successfully!")
}
```

## 📚 Core Features

### 1. TAR File Creation
- Create new TAR archive files
- Add files and directories to archives
- Support recursive directory structure addition
- Custom file filters

### 2. TAR File Reading
- Read existing TAR files
- Iterate through archive members
- Get file metadata information
- Streaming read support

### 3. TAR File Extraction
- Extract individual files
- Batch extract all files
- Preserve file permissions and timestamps
- Support symbolic links and hard links

### 4. Compression Format Support
- `.tar` - Uncompressed TAR files
- `.tar.gz` / `.tgz` - Gzip compression
- `.tar.bz2` - Bzip2 compression  
- `.tar.xz` - XZ compression

### 5. Advanced Features
- PAX extended header support
- GNU TAR format compatibility
- Sparse file handling
- Large file support (>8GB)

## 🔧 API Reference

### Main Types

```go
// TarFile - TAR file operation object
type TarFile struct {
    // ... private fields
}

// TarInfo - TAR file member information
type TarInfo struct {
    Name     string    // File name
    Size     int64     // File size
    Mode     int64     // Permission mode
    Mtime    time.Time // Modification time
    Type     string    // File type
    // ... other fields
}
```

### Main Methods

```go
// Open TAR file
func Open(name, mode string, fileobj io.ReadWriteSeeker, bufsize int) (*TarFile, error)

// Add file to archive
func (tf *TarFile) Add(name, arcname string, recursive bool, filter func(*TarInfo) (*TarInfo, error)) error

// Get all members
func (tf *TarFile) GetMembers() ([]*TarInfo, error)

// Extract file
func (tf *TarFile) Extract(member *TarInfo, path string) error

// Extract all files
func (tf *TarFile) ExtractAll(path string) error
```

## 📖 Detailed Documentation

For more detailed usage examples and API documentation, please check the [docs](./docs/) directory:

- [Installation Guide](./docs/installation-en.md)
- [Basic Tutorial](./docs/basic-tutorial-en.md)
- [Advanced Usage](./docs/advanced-usage-en.md)
- [API Reference](./docs/api-reference-en.md)
- [Performance Optimization](./docs/performance-en.md)

## 🤝 Contributing

We welcome Issues and Pull Requests to help improve this project!

1. Fork this repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

This project uses the MIT License. See the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by Python's [tarfile](https://docs.python.org/3/library/tarfile.html) module
- Thanks to the Go language community for support and contributions

---

**If this project helps you, please give it a ⭐️ to show your support!** 