package main

import (
	"fmt"
	"gtarfile/tarfile"
	"log"
)

func main() {

	// 打开 tar 文件，使用文件名、读模式 "r"，fileobj 为 nil，bufsize 为 4096
	tf, err := tarfile.Open("test/image.tar", "r", nil, 4096)
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
	for _, member := range members {
		fmt.Println(member.Name)
	}
}
