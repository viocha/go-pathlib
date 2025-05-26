# go-pathlib

`go-pathlib` 是一个受 Python `pathlib` 启发的 Go 语言库，用于简化路径操作和文件系统交互。它提供了面向对象的接口，支持跨平台的路径处理，并封装了常见的文件操作。

## 特性

- **面向对象**：通过接口和结构体封装路径操作。
- **丰富的功能**：
  - 路径拼接、解析、转换。
  - 文件和目录的创建、删除、移动、复制。
  - 符号链接的处理。
  - 文件内容的读写。
  - 目录遍历和通配符匹配。
- **错误处理**：提供详细的错误信息，支持 panic 版本的方法。
- 目前仅实现了 Windows 平台的功能

## 安装

使用 `go get` 安装：

```bash
go get github.com/viocha/go-pathlib
```

## 快速开始

以下是一些常见用法的示例：

### 创建路径对象

```go
package main

import (
	"fmt"
	"github.com/viocha/go-pathlib/path"
)

func main() {
	p := path.New("example", "file.txt")
	fmt.Println(p.String()) // 输出: example\file.txt
}
```

### 路径操作

```go
p := path.New("example", "file.txt")

// 获取父路径
parent := p.Parent()
fmt.Println(parent.String()) // 输出: example

// 拼接路径
newPath := p.Join("subdir")
fmt.Println(newPath.String()) // 输出: example\file.txt\subdir
```

### 文件操作

```go
p := path.New("example", "file.txt")

// 确保文件存在
err := p.EnsureFile()
if err != nil {
	fmt.Println("Error:", err)
}

// 写入文件
err = p.Write("Hello, World!")
if err != nil {
	fmt.Println("Error:", err)
}

// 读取文件
content, err := p.Read()
if err != nil {
	fmt.Println("Error:", err)
}
fmt.Println(content) // 输出: Hello, World!
```

### 目录操作

```go
dir := path.New("example", "subdir")

// 确保目录存在
err := dir.EnsureDir()
if err != nil {
	fmt.Println("Error:", err)
}

// 遍历目录
paths, err := dir.ReadDir()
if err != nil {
	fmt.Println("Error:", err)
}
for _, p := range paths {
	fmt.Println(p.String())
}
```

### 符号链接

```go
link := path.New("example", "link")
target := path.New("example", "file.txt")

// 创建符号链接
err := link.Symlink(target)
if err != nil {
	fmt.Println("Error:", err)
}

// 读取符号链接
resolved, err := link.ReadLink()
if err != nil {
	fmt.Println("Error:", err)
}
fmt.Println(resolved.String()) // 输出: example\file.txt
```

## 贡献

欢迎贡献代码！请提交 Pull Request 或报告问题。

## 许可证

[MIT](LICENSE)
