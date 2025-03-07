## Overview

gtarfile is a Golang library inspired by Python's tarfile module. It provides functionality to read, write, and manipulate tar archives, built on top of the archiver/tar package. This project aims to replicate the core features of Python's tarfile.TarFile and tarinfo.TarInfo classes, offering a lightweight and efficient alternative for tar file handling in Go.

## Features

- Create, read, and extract tar archives.
- Support for TarFile-like operations (open, extract, add files, etc.).
- Implementation of TarInfo for metadata handling (file name, size, permissions, etc.).
- Built with simplicity and performance in mind.

## Installation

To use gtarfile in your project, ensure you have Go installed, then run:

bash

CollapseWrapCopy

```
go get github.com/lyon-v/gtarfile
```

## Usage

Here’s a basic example of how to use gtarfile:

go

CollapseWrapCopy

```
package main

import (
    "fmt"
    "github.com/lyon-v/gtarfile"
)

func main() {
    // Open an existing tar file
    tf, err := gtarfile.Open("example.tar", "r")
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    defer tf.Close()

    // List all files in the tar archive
    for _, info := range tf.GetMembers() {
        fmt.Printf("File: %s, Size: %d bytes\n", info.Name, info.Size)
    }

    // Extract all files
    err = tf.ExtractAll("./output")
    if err != nil {
        fmt.Println("Error extracting:", err)
    }
}
```

## Development Status

This project is under active development. Current goals include:

1. Full implementation of TarFile class with read/write support.
2. Complete TarInfo class for detailed file metadata.
3. Support for compression (e.g., gzip) in future releases.

## Contributing

Contributions are welcome! Feel free to submit issues or pull requests to the repository.

## License

This project is licensed under the [MIT License](LICENSE). See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built using the archiver/tar package from the Go ecosystem.
- Inspired by Python's tarfile module.