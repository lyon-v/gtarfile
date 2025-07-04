# Installation Guide

This guide provides detailed instructions for installing and setting up the GTarFile library.

## System Requirements

### Go Version Requirements
- Go 1.19 or higher
- Operating Systems: Linux, macOS, Windows

### Dependency Requirements
- Standard Go library (no external dependencies required)

## Installation Methods

### 1. Using go get (Recommended)

This is the simplest and most commonly used installation method:

```bash
# Install the latest version
go get github.com/yourusername/gtarfile

# Install a specific version
go get github.com/yourusername/gtarfile@v1.0.0
```

### 2. Using go mod

If you're using Go modules in your project:

```bash
# Initialize go mod (if not already done)
go mod init your-project-name

# Add dependency
go get github.com/yourusername/gtarfile

# Update go.mod and go.sum
go mod tidy
```

### 3. Manual Installation

Clone the repository and install manually:

```bash
# Clone repository
git clone https://github.com/yourusername/gtarfile.git

# Enter directory
cd gtarfile

# Install
go install
```

## Verify Installation

Create a simple test file to verify the installation was successful:

```go
// test_installation.go
package main

import (
    "fmt"
    "log"
    "gtarfile/tarfile"
)

func main() {
    // Try creating a TarFile object
    tf, err := tarfile.Open("test.tar", "w", nil, 4096)
    if err != nil {
        log.Fatal("Installation failed:", err)
    }
    defer tf.Close()
    
    fmt.Println("GTarFile installation successful!")
}
```

Run the test:

```bash
go run test_installation.go
```

If you see "GTarFile installation successful!", the installation was successful.

## Development Environment Setup

### Recommended IDE Configuration

#### VS Code
Install the following extensions:
- Go extension (official)
- Go Test Explorer
- GitLens

#### GoLand/IntelliJ IDEA
- Enable Go plugin
- Configure Go SDK path

### Project Setup

If you want to contribute to the project or modify the source code:

```bash
# Clone repository
git clone https://github.com/yourusername/gtarfile.git
cd gtarfile

# Install development dependencies
go mod download

# Run tests
go test ./...

# Build
go build ./...
```

## Configuration Options

### Environment Variables

```bash
# Set Go module proxy (optional)
export GOPROXY=https://proxy.golang.org,direct

# Set Go module checksum database (optional)
export GOSUMDB=sum.golang.org

# Enable Go module mode
export GO111MODULE=on
```

### Build Tags

The library supports some optional build tags:

```bash
# Build with debug information
go build -tags debug

# Build for production (optimized)
go build -tags production -ldflags "-s -w"
```

## Troubleshooting

### Common Issues

#### 1. Go Version Too Old
```
Error: go version go1.18 is not supported
```
**Solution**: Update Go to version 1.19 or higher.

#### 2. Module Not Found
```
Error: cannot find module github.com/yourusername/gtarfile
```
**Solution**: Check if the module path is correct, or try using a proxy:
```bash
export GOPROXY=https://proxy.golang.org,direct
go clean -modcache
go get github.com/yourusername/gtarfile
```

#### 3. Permission Issues
```
Error: permission denied
```
**Solution**: Ensure you have write permissions to the GOPATH directory, or use Go modules.

#### 4. Network Connection Issues
```
Error: dial tcp: i/o timeout
```
**Solution**: Check network connection, or use a proxy:
```bash
export GOPROXY=https://goproxy.cn,direct  # China users
```

### Getting Help

If you encounter other issues:

1. Check the [Issues](https://github.com/yourusername/gtarfile/issues) page
2. Search for existing solutions
3. Create a new Issue with detailed error information

## Performance Optimization

### Build Options

For production environments, use the following build command for optimal performance:

```bash
go build -ldflags "-s -w" -gcflags "-B"
```

### Runtime Configuration

```go
// Set appropriate buffer size for large files
tf, err := tarfile.Open("large.tar", "r", nil, 65536) // 64KB buffer
```

## Next Steps

After successful installation, please check:
- [Basic Tutorial](./basic-tutorial-en.md) - Learn basic usage
- [Advanced Usage](./advanced-usage-en.md) - Learn advanced features 