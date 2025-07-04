package main

import (
	"fmt"
	"gtarfile/tarfile"
	"log"
	"os"
	"strings"
)

func main() {
	// 演示如何创建一个简单的tar文件
	createExampleTar()

	// 演示如何读取tar文件
	readExampleTar()

	// 演示如何提取tar文件
	extractExampleTar()
}

func createExampleTar() {
	// 创建一个新的tar文件用于写入
	tf, err := tarfile.Open("example.tar", "w", nil, 4096)
	if err != nil {
		log.Fatalf("创建 tar 文件失败: %v", err)
	}
	defer tf.Close()

	// 添加一个示例文件
	content := "Hello, World! This is a test file."
	ti := tarfile.NewTarInfo("test.txt")
	ti.Size = int64(len(content))

	err = tf.AddFile(ti, strings.NewReader(content))
	if err != nil {
		log.Fatalf("添加文件失败: %v", err)
	}

	fmt.Println("成功创建 example.tar")
}

func readExampleTar() {
	// 检查文件是否存在
	if _, err := os.Stat("example.tar"); os.IsNotExist(err) {
		log.Printf("example.tar 不存在，跳过读取操作")
		return
	}

	// 打开 tar 文件，使用文件名、读模式 "r"，fileobj 为 nil，bufsize 为 4096
	tf, err := tarfile.Open("example.tar", "r", nil, 4096)
	if err != nil {
		log.Fatalf("打开 tar 文件失败: %v", err)
	}
	defer tf.Close()

	// 获取 tar 文件成员
	members, err := tf.GetMembers()
	if err != nil {
		log.Fatalf("获取 tar 成员失败: %v", err)
	}

	// 遍历打印每个成员信息
	fmt.Println("Tar 文件成员:")
	for _, member := range members {
		fmt.Printf("- %s (大小: %d 字节)\n", member.Name, member.Size)
	}
}

func extractExampleTar() {
	// 检查文件是否存在
	if _, err := os.Stat("example.tar"); os.IsNotExist(err) {
		log.Printf("example.tar 不存在，跳过提取操作")
		return
	}

	// 打开tar文件用于提取
	tf, err := tarfile.Open("example.tar", "r", nil, 4096)
	if err != nil {
		log.Fatalf("打开 tar 文件失败: %v", err)
	}
	defer tf.Close()

	// 创建提取目录
	extractDir := "extracted"
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		log.Fatalf("创建提取目录失败: %v", err)
	}

	// 提取所有文件
	err = tf.ExtractAll(extractDir)
	if err != nil {
		log.Fatalf("提取文件失败: %v", err)
	}

	fmt.Println("成功提取到", extractDir, "目录")

	// 验证提取的文件
	extractedFile := extractDir + "/test.txt"
	if content, err := os.ReadFile(extractedFile); err == nil {
		fmt.Printf("提取的文件内容: %s\n", string(content))
	} else {
		log.Printf("读取提取文件失败: %v", err)
	}
}
