package path

import (
	"errors"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/viocha/go-pathlib/common"
	"github.com/viocha/go-pathlib/purepath"
	"github.com/viocha/go-pathlib/purepath/ntpath"
)

type MergeMode int

const (
	MergeModeError   MergeMode = iota // 返回错误
	MergeModeSkip                     // 跳过冲突
	MergeModeReplace                  // 替换冲突
)

var (
	WalkSkip = errors.New("walk skip") // Walk函数中跳过当前目录的向下遍历
	WalkStop = errors.New("walk stop") // Walk函数中终止遍历
)

type IPath interface {
	IBasePath

	ToURL() (string, error)         // 转换成文件URL，返回 file:// URL 格式，支持UNC路径和本地路径
	ToAbs() (IPath, error)          // 转换成绝对路径
	ReadLink() (IPath, error)       // 读取符号链接的目标路径，目标路径不存在也会返回，如果是相对链接的路径，也会原样返回
	ReadLinkPath() (IPath, error)   // 读取符号链接的目标路径，如果是相对路径，会和链接的路径进行拼接
	Resolve() (IPath, error)        // 转换成绝对路径，并解析符号链接
	ToPurePath() purepath.IPurePath // 转换成纯路径对象

	// 查询状态
	Stat() (os.FileInfo, error)    // 跟随符号链接的状态查询
	Lstat() (os.FileInfo, error)   // 不跟随符号链接的状态查询
	Exists(follow ...bool) bool    // 默认会跟随符号链接，所以可能出现符号链接存在但是目标文件不存在的情况
	IsFile(follow ...bool) bool    // 默认会跟随符号链接
	IsDir(follow ...bool) bool     // 默认会跟随符号链接
	IsLink() bool                  // 是否是符号链接，不会跟随目标文件的符号链接
	SameFile(otherPath IPath) bool // 是否是同一个文件，会跟随符号链接

	// 文件读写
	Open(mode ...int) (*os.File, error)          // 打开文件，默认只读模式，不存在会返回错误
	OpenWrite(append ...bool) (*os.File, error)  // 打开文件用于写入，不存在会自动创建，默认覆盖并清空已有内容，可以指定追加模式
	Read(encoding ...string) (string, error)     //  读取文本内容，默认使用UTF-8编码。TODO:实现其他编码
	ReadBytes() ([]byte, error)                  // 读取字节切片
	Write(text string, encoding ...string) error // 写入文本内容，默认使用UTF-8编码，会自动创建父路径。TODO：实现其他编码
	WriteBytes([]byte) error                     // 写入字节切片，会自动创建父路径

	// 创建和删除
	Create(parents ...bool) error                // 创建或清空文件，默认会创建父路径
	Mkdir(parents ...bool) error                 // 创建文件夹，默认会创建父路径
	Symlink(target IPath, parents ...bool) error // 创建符号链接，默认会创建父路径
	EnsureFile() error                           // 确保文件存在，如果存在但不是文件则返回错误
	EnsureDir() error                            // 确保目录存在，如果存在但不是目录则返回错误
	Remove(recursive ...bool) error              // 删除文件或目录，默认递归删除

	// 重命名、移动、复制。都不跟随符号链接，操作符号链接本身
	Rename(newName string, replace ...bool) (IPath, error) // 使用Move方法实现
	Move(dst IPath, replace ...bool) error                 // 支持文件，符号链接，文件夹，不支持合并文件夹，需要删除整个目标文件夹
	Copy(dst IPath, replace ...bool) error                 // 支持文件，符号链接，文件夹，支持递归复制文件夹，但需要删除整个目标文件夹
	CopyMerge(dst IPath, mergeMode ...MergeMode) error     // 使用合并方式，递归复制文件夹，支持跳过冲突或覆盖文件，默认返回错误
	MoveMerge(dst IPath, mergeMode ...MergeMode) error     // 使用合并方式，递归移动文件夹，支持跳过冲突或覆盖文件，默认返回错误

	// 目录读取和遍历
	ReadDir() ([]IPath, error)
	// 使用通配符的读取所有匹配的路径，支持**，默认不跟随符号链接，不跳过循环的链接，而是返回错误
	Glob(pattern string, globOptions ...GlobOptions) ([]IPath, error)
	// 自顶向下遍历目录，fn返回nil表示继续遍历，WalkSkip表示跳过当前目录的向下遍历，WalkStop表示终止遍历
	Walk(fn func(path IPath, err error) error, follow ...bool) error

	// panic版本的方法
	MustToURL() string
	MustToAbs() IPath
	MustReadLink() IPath
	MustReadLinkPath() IPath
	MustResolve() IPath

	MustStat() os.FileInfo
	MustLStat() os.FileInfo
	MustOpen(mode ...int) *os.File
	MustOpenWrite(append ...bool) *os.File
	MustRead() string
	MustReadBytes() []byte

	MustRename(newName string, replace ...bool) IPath
	MustReadDir() []IPath
	MustGlob(pattern string, globOptions ...GlobOptions) []IPath
}

var (
	ErrParseURL = errors.New("failed to parse URL")
)

func New(segments ...string) IPath {
	if runtime.GOOS == "windows" {
		return NewWindowsPath(segments...)
	}
	panic("path package is not implemented for this OS")
}

func FromPurePath(purePath purepath.IPurePath) IPath {
	if purePath == nil { // 如果传入的纯路径是 nil，返回 nil
		return nil
	}
	return New(purePath.String())
}

// 从文件URL创建 Path
func FromURL(fileUrl string) (IPath, error) {
	path, err := URLToPath(fileUrl)
	if err != nil {
		return nil, err
	}
	return New(path), nil
}

func MustFromURL(fileUrl string) IPath {
	path, err := URLToPath(fileUrl)
	if err != nil {
		panic(err)
	}
	return New(path)
}

func URLToPath(fileUrl string) (string, error) {
	// 解析成文件路径
	u, err := url.Parse(fileUrl)
	if err != nil {
		return "", common.WrapSub(err, ErrParseURL, "failed to parse URL: %q", fileUrl)
	}

	if u.Scheme != "file" {
		return "", common.WrapMsg(ErrParseURL, "not a file URL: %s", fileUrl)
	}
	host := u.Host
	path := u.Path
	if runtime.GOOS == "windows" {
		if len(path) >= 3 && path[2] == ':' { // 包含盘符
			// 去除前导的斜杠
			path = strings.TrimPrefix(path, "/")
		} else { // 判断是否是UNC路径
			// 连接host
			path = "//" + host + path
		}
		path = ntpath.Clean(path)
	}
	return path, nil
}

func MustURLToPath(fileUrl string) string {
	path, err := URLToPath(fileUrl)
	if err != nil {
		panic(err)
	}
	return path
}
