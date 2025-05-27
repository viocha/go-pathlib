package path

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/viocha/go-pathlib/common"
	"github.com/viocha/go-pathlib/purepath"
)

type WindowsPath struct {
	*BasePath
}

// 确保实现了 IPath 接口
var _ IPath = (*WindowsPath)(nil)

func NewWindowsPath(segments ...string) WindowsPath {
	return WindowsPath{
		BasePath: NewBasePath(segments...),
	}
}

var (
	ErrToURL      = errors.New("failed to convert to URL")
	ErrToAbs      = errors.New("failed to convert to absolute path")
	ErrReadLink   = errors.New("failed to read symlink")
	ErrResolve    = errors.New("failed to resolve path")
	ErrReadStat   = errors.New("failed to read file status")
	ErrReadLstat  = errors.New("failed to read file status without following symlink")
	ErrOpen       = errors.New("failed to open file")
	ErrCreate     = errors.New("failed to create file")
	ErrMkdir      = errors.New("failed to create directory")
	ErrSymlink    = errors.New("failed to create symlink")
	ErrRead       = errors.New("failed to read file")
	ErrWrite      = errors.New("failed to write file")
	ErrEnsureDir  = errors.New("failed to ensure directory exists")
	ErrEnsureFile = errors.New("failed to ensure file exists")
	ErrRemove     = errors.New("failed to remove file or directory")
	ErrRename     = errors.New("failed to rename file or directory")
	ErrMove       = errors.New("failed to move file or directory")
	ErrCopy       = errors.New("failed to copy file or directory")
	ErrCopyMerge  = errors.New("failed to copy-merge file or directory")
	ErrReadDir    = errors.New("failed to read directory")
)

func (p WindowsPath) ToPurePath() purepath.IPurePath {
	return purepath.New(p.String())
}

// 转换成URL
func (p WindowsPath) ToURL() (string, error) {
	absPath, err := p.ToAbs()
	if err != nil {
		return "", common.WrapSub(ErrToURL, err, "failed to convert to absolute path: %q", p)
	}
	anchor := absPath.Anchor()
	path := absPath.String()
	if strings.HasPrefix(anchor, `\\`) { // UNC 路径
		// 提取host 和 urlPath
		parts := strings.SplitN(strings.TrimPrefix(path, `\\`), `\`, 2)
		if len(parts) < 2 {
			return "", common.WrapMsg(ErrToURL, "invalid UNC path: %q", path)
		}
		host := parts[0]
		urlPath := parts[1]
		url := fmt.Sprintf("file://%s/%s", host, filepath.ToSlash(urlPath))
		return url, nil
	} else {
		// 普通绝对路径，包含盘符
		return fmt.Sprintf("file:///%s", filepath.ToSlash(path)), nil
	}
}

func (p WindowsPath) ToAbs() (IPath, error) {
	absPath, err := filepath.Abs(p.String())
	if err != nil {
		return nil, common.WrapSub(ErrToAbs, err, "failed to convert to absolute path: %q", p)
	}
	return NewWindowsPath(absPath), nil
}

// 读取符号链接的目标路径
func (p WindowsPath) ReadLink() (IPath, error) {
	target, err := os.Readlink(p.String())
	if err != nil {
		return nil, common.WrapSub(ErrReadLink, err, "failed to read symlink: %q", p)
	}
	// 返回一个新的 WindowsPath 实例
	return NewWindowsPath(target), nil
}

func (p WindowsPath) ReadLinkPath() (IPath, error) {
	// 读取符号链接的目标路径
	target, err := p.ReadLink()
	if err != nil {
		return nil, err
	}
	// 如果是相对路径，和链接路径进行拼接
	if !target.IsAbs() {
		target = p.Parent().Join(target.String())
	}
	return target, nil
}

// 转换成绝对路径，并解析符号链接，返回最终的目标路径
func (p WindowsPath) Resolve() (IPath, error) {
	absPath, err := p.ToAbs()
	if err != nil {
		return nil, errors.Join(ErrResolve, err)
	}
	// 解析符号链接
	resolvedPath, err := absPath.ReadLink()
	if err != nil {
		return nil, errors.Join(ErrResolve, err)
	}
	return resolvedPath, nil
}

// 判断路径是否存在，默认跟随符号链接
func (p WindowsPath) Exists(follow ...bool) bool {
	isFollow := true
	if len(follow) > 0 {
		isFollow = follow[0]
	}
	if isFollow {
		_, err := p.Stat()
		return err == nil
	} else {
		_, err := p.Lstat()
		return err == nil
	}
}

// 返回文件的状态信息，会跟随目标的符号链接
func (p WindowsPath) Stat() (os.FileInfo, error) {
	info, err := os.Stat(p.String())
	if err != nil {
		return nil, common.WrapSub(ErrReadStat, err, "failed to read file status: %q", p)
	}
	return info, nil
}

// 返回文件的状态信息，不会跟随目标的符号链接
func (p WindowsPath) Lstat() (os.FileInfo, error) {
	info, err := os.Lstat(p.String())
	if err != nil {
		return nil, common.WrapSub(ErrReadLstat, err, "failed to read file status without following symlink: %q", p)
	}
	return info, nil
}

// 如果 follow 为 true，则跟目标文件的符号链接，中间的符号链接始终会被跟随
func (p WindowsPath) IsFile(follow ...bool) bool {
	isFollow := common.ParseOptional(follow, true) // 默认跟随符号链接
	if isFollow {
		stat, err := p.Stat()
		return err == nil && stat.Mode().IsRegular()
	} else {
		stat, err := p.Lstat()
		return err == nil && stat.Mode().IsRegular()
	}
}

// 如果 follow 为 true，则跟目标文件的符号链接，中间的符号链接始终会被跟随
func (p WindowsPath) IsDir(follow ...bool) bool {
	isFollow := common.ParseOptional(follow, true) // 默认跟随符号链接
	if isFollow {
		stat, err := p.Stat()
		return err == nil && stat.IsDir()
	} else {
		stat, err := p.Lstat()
		return err == nil && stat.IsDir()
	}
}

// 判断路径是否是符号链接
func (p WindowsPath) IsLink() bool {
	info, err := p.Lstat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// 判断两个路径是否指向同一个文件
func (p WindowsPath) SameFile(otherPath IPath) bool {
	info1, err1 := p.Stat()
	info2, err2 := otherPath.Stat()
	if err1 != nil || err2 != nil {
		return false
	}
	return os.SameFile(info1, info2)
}

// 打开文件，默认只读方式打开，故不存在默认返回错误。
// 可以自行指定写入方式，比如附加，从开头覆写，不存在则创建，存在则清空等
func (p WindowsPath) Open(mode ...int) (*os.File, error) {
	openMode := common.ParseOptional(mode, os.O_RDONLY) // 默认只读方式打开
	file, err := os.OpenFile(p.String(), openMode, os.ModePerm)
	if err != nil {
		return nil, common.WrapSub(ErrOpen, err, "failed to open file %q with mode %d", p, openMode)
	}
	return file, nil
}

// 打开文件用于写入，文件不存在会自动创建，默认覆盖并清空已有内容，可以指定追加模式
func (p WindowsPath) OpenWrite(append ...bool) (*os.File, error) {
	isAppend := common.ParseOptional(append, false) // 默认不追加内容
	if !p.Exists() {                                // 如果文件不存在，直接创建新文件
		if err := p.Create(); err != nil {
			return nil, err
		}
	}
	mode := os.O_WRONLY | os.O_TRUNC // 默认覆盖并清空已有内容
	if isAppend {
		mode = os.O_WRONLY | os.O_APPEND // 追加模式
	}
	file, err := os.OpenFile(p.String(), mode, os.ModePerm)
	if err != nil {
		return nil, common.WrapSub(ErrOpen, err, "failed to open file %q for writing with append=%v", p, isAppend)
	}
	return file, nil
}

// 读取文件内容，返回字符串
func (p WindowsPath) Read(encoding ...string) (string, error) {
	if len(encoding) > 0 {
		panic("encoding is not implemented yet")
	}
	content, err := os.ReadFile(p.String())
	if err != nil {
		return "", common.WrapSub(ErrRead, err, "failed to read file: %q", p)
	}
	return string(content), nil
}

// 将字符串写入文件，会先清空文件内容
func (p WindowsPath) Write(text string, encoding ...string) error {
	if len(encoding) > 0 {
		panic("encoding is not implemented yet")
	}
	if err := p.Create(); err != nil {
		return err
	}
	err := os.WriteFile(p.String(), []byte(text), os.ModePerm)
	if err != nil {
		return common.WrapSub(ErrWrite, err, "failed to write file: %q", p)
	}
	return nil
}

// 读取文件内容为字节切片
func (p WindowsPath) ReadBytes() ([]byte, error) {
	content, err := os.ReadFile(p.String())
	if err != nil {
		return nil, common.WrapSub(ErrRead, err, "failed to read file: %q", p)
	}
	return content, nil
}

// 将字节切片写入文件
func (p WindowsPath) WriteBytes(data []byte) error {
	if err := p.Create(); err != nil {
		return err
	}
	err := os.WriteFile(p.String(), data, os.ModePerm)
	if err != nil {
		return common.WrapSub(ErrWrite, err, "failed to write file: %q", p)
	}
	return nil
}

// 创建文件，或者清空文件内容
func (p WindowsPath) Create(parents ...bool) error {
	createParents := common.ParseOptional(parents, true) // 默认创建父目录

	if createParents {
		if err := p.Parent().EnsureDir(); err != nil {
			return err
		}
	}

	file, err := os.Create(p.String())
	if err != nil {
		return common.WrapSub(ErrCreate, err, "failed to create file: %q", p)
	}
	defer closeFile(file)
	return nil
}

// 创建目录，默认会创建父目录
func (p WindowsPath) Mkdir(parents ...bool) error {
	createParents := common.ParseOptional(parents, true) // 默认创建父目录
	// 如果不创建父目录，直接创建当前目录
	if createParents {
		err := os.MkdirAll(p.String(), os.ModePerm)
		if err != nil {
			return common.WrapSub(ErrMkdir, err, "failed to create directory %q with parents=%v", p, createParents)
		}
		return nil
	}
	err := os.Mkdir(p.String(), os.ModePerm)
	if err != nil {
		return common.WrapSub(ErrMkdir, err, "failed to create directory %q without parents=%v", p, createParents)
	}
	return nil
}

// 创建符号链接
func (p WindowsPath) Symlink(target IPath, parents ...bool) error {
	createParents := common.ParseOptional(parents, true) // 默认创建父目录

	if createParents {
		if err := p.Parent().EnsureDir(); err != nil {
			return err
		}
	}

	err := os.Symlink(target.String(), p.String())
	if err != nil {
		return common.WrapSub(ErrSymlink, err, "failed to create symlink from %q to %q", p, target)
	}
	return nil
}

// 确保文件存在，如果不存在则创建
func (p WindowsPath) EnsureFile() error {
	if p.IsFile() {
		return nil // 文件已存在
	}
	if p.Exists() { // 如果路径存在但不是文件，返回错误
		return common.WrapMsg(ErrEnsureFile, "path exists but is not a file: %q", p)
	}
	return p.Create(true) // 创建文件并确保父目录存在
}

// 确保目录存在，如果不存在则创建
func (p WindowsPath) EnsureDir() error {
	if p.IsDir() {
		return nil // 目录已存在
	}
	if p.Exists() { // 如果路径存在但不是目录，返回错误
		return common.WrapMsg(ErrEnsureDir, "path exists but is not a directory: %q", p)
	}
	err := p.Mkdir(true) // 创建目录并确保父目录存在
	if err != nil {
		return common.WrapSub(ErrEnsureDir, err, "directory : %q", p)
	}
	return nil
}

// 删除文件或目录，默认递归删除
func (p WindowsPath) Remove(recursive ...bool) error {
	isRecursive := common.ParseOptional(recursive, true) // 默认递归删除

	if !p.Exists(false) { // 不跟随符号链接，os.Remove只会删除链接本身
		return nil // 如果路径不存在，直接返回 nil，静默成功
	}

	if isRecursive {
		err := os.RemoveAll(p.String()) // 递归删除
		if err != nil {
			return common.WrapSub(ErrRemove, err, "failed to remove path %q recursively", p)
		}
		return nil // 成功删除
	} else {
		err := os.Remove(p.String()) // 非递归删除文件或目录，如果是目录且非空会返回错误
		if err != nil {
			return common.WrapSub(ErrRemove, err, "failed to remove path %q non-recursively", p)
		}
		return nil // 成功删除
	}
}

// 重命名文件或目录
func (p WindowsPath) Rename(newName string, replace ...bool) (IPath, error) {
	newPath, err := p.WithName(newName)
	if err != nil {
		return nil, common.WrapSub(ErrRename, err, "failed to create new path with name %q from %q", newName, p)
	}
	if err := p.Move(newPath, replace...); err != nil {
		return nil, err
	}
	return newPath, nil
}

// 移动文件或目录到新路径
func (p WindowsPath) Move(dst IPath, replace ...bool) error {
	if p.SameFile(dst) { // 如果源路径和目标路径相同，直接返回
		return nil
	}
	// 确保目标路径的父目录存在，以及目标路径不存在
	if err := ensureMove(dst, replace...); err != nil {
		return err
	}
	if p.IsLink() { // 如果是符号链接，直接使用 os.Symlink 创建新的符号链接并删除旧的符号链接
		if err := CopySymlink(p, dst); err != nil {
			return err
		}
		if err := p.Remove(); err != nil {
			return err
		}
	} else if p.IsFile(false) || p.IsDir(false) { // 如果是文件或目录，直接使用 os.Rename 移动文件
		if err := os.Rename(p.String(), dst.String()); err != nil {
			return common.WrapSub(ErrMove, err, "failed to move path from %q to %q", p, dst)
		}
	}
	return nil
}

func (p WindowsPath) Copy(dst IPath, replace ...bool) error {
	if p.SameFile(dst) { // 如果源路径和目标路径相同，直接返回
		return nil
	}
	// 确保目标路径的父目录存在，以及目标路径不存在
	if err := ensureMove(dst, replace...); err != nil {
		return err
	}
	if p.IsLink() { // 先判断是否是符号链接
		if err := CopySymlink(p, dst); err != nil {
			return err
		}
	} else if p.IsFile(false) {
		if err := CopyFile(p, dst); err != nil {
			return err
		}
	} else if p.IsDir(false) {
		if err := CopyDir(p, dst); err != nil {
			return err
		}
	} else { // 不支持复制的类型
		return common.WrapMsg(ErrCopy, "unsupported path type for copy: %q", p)
	}
	return nil
}

// p 必须是一个目录，dst 必须不存在，或者是一个目录。
func (p WindowsPath) CopyMerge(dst IPath, mergeMode ...MergeMode) error {
	mode := common.ParseOptional(mergeMode, MergeModeError) // 默认返回错误
	if p.SameFile(dst) {                                    // 如果源路径和目标路径相同，直接返回
		return nil
	}
	if !p.IsDir(false) { // 如果源路径不是目录，返回错误
		return common.WrapMsg(ErrCopyMerge, "source path %q is not a directory", p)
	}
	if err := dst.EnsureDir(); err != nil { // 确保目标目录存在
		return err
	}
	return CopyMerge(p, dst, mode)
}

func (p WindowsPath) MoveMerge(dst IPath, mergeMode ...MergeMode) error {
	if err := p.CopyMerge(dst, mergeMode...); err != nil {
		return err
	}
	if err := p.Remove(); err != nil {
		return err
	}
	return nil
}

// 读取目录内容，返回路径列表
func (p WindowsPath) ReadDir() ([]IPath, error) {
	entries, err := os.ReadDir(p.String())
	if err != nil {
		return nil, common.WrapSub(ErrReadDir, err, "failed to read directory: %q", p)
	}
	var paths []IPath
	for _, entry := range entries {
		paths = append(paths, p.Join(entry.Name()))
	}
	return paths, nil
}

type GlobOptions struct {
	Follow      bool // 是否跟随符号链接，默认不跟随
	SkipOnCycle bool // 是否跳过循环链接，默认不跳过
}

func (p WindowsPath) Glob(pattern string, globOptions ...GlobOptions) ([]IPath, error) {
	options := common.ParseOptional(globOptions, GlobOptions{}) // 默认不跟随符号链接，不跳过循环链接
	var result []IPath
	err := p.Walk(func(path IPath, err error) error {
		if err != nil {
			if errors.Is(err, ErrWalkCycle) && options.SkipOnCycle { // 检测到循环链接，并允许跳过
				return nil
			}
			return err
		}
		if path.FullMatch(pattern) {
			result = append(result, path)
		}
		return nil
	}, options.Follow)

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p WindowsPath) Walk(fn func(path IPath, err error) error, follow ...bool) error {
	return Walk(p, fn, follow...)
}

// ======================== panic版本的方法 ========================

func (p WindowsPath) MustToURL() string {
	url, err := p.ToURL()
	if err != nil {
		panic(err)
	}
	return url
}

// MustToAbs 返回绝对路径，如果失败则 panic
func (p WindowsPath) MustToAbs() IPath {
	absPath, err := p.ToAbs()
	if err != nil {
		panic(err)
	}
	return absPath
}

// 返回符号链接的目标路径，如果失败则 panic
func (p WindowsPath) MustReadLink() IPath {
	target, err := p.ReadLink()
	if err != nil {
		panic(err)
	}
	return target
}

func (p WindowsPath) MustReadLinkPath() IPath {
	target, err := p.ReadLinkPath()
	if err != nil {
		panic(err)
	}
	return target
}

// 返回解析后的路径，如果失败则 panic
func (p WindowsPath) MustResolve() IPath {
	resolvedPath, err := p.Resolve()
	if err != nil {
		panic(err)
	}
	return resolvedPath
}

func (p WindowsPath) MustStat() os.FileInfo {
	stat, err := p.Stat()
	if err != nil {
		panic(err)
	}
	return stat
}

func (p WindowsPath) MustLStat() os.FileInfo {
	stat, err := p.Lstat()
	if err != nil {
		panic(err)
	}
	return stat
}

// 打开文件，如果失败则 panic
func (p WindowsPath) MustOpen(mode ...int) *os.File {
	file, err := p.Open(mode...)
	if err != nil {
		panic(err)
	}
	return file
}

// 打开文件用于写入，如果失败则 panic
func (p WindowsPath) MustOpenWrite(append ...bool) *os.File {
	file, err := p.OpenWrite(append...)
	if err != nil {
		panic(err)
	}
	return file
}

// 读取文件内容，如果失败则 panic
func (p WindowsPath) MustRead() string {
	content, err := p.Read()
	if err != nil {
		panic(err)
	}
	return content
}

// 读取文件内容为字节切片，如果失败则 panic
func (p WindowsPath) MustReadBytes() []byte {
	content, err := p.ReadBytes()
	if err != nil {
		panic(err)
	}
	return content
}

func (p WindowsPath) MustRename(newName string, replace ...bool) IPath {
	newPath, err := p.Rename(newName, replace...)
	if err != nil {
		panic(err)
	}
	return newPath
}

func (p WindowsPath) MustReadDir() []IPath {
	paths, err := p.ReadDir()
	if err != nil {
		panic(err)
	}
	return paths
}

func (p WindowsPath) MustGlob(pattern string, globOptions ...GlobOptions) []IPath {
	paths, err := p.Glob(pattern, globOptions...)
	if err != nil {
		panic(err)
	}
	return paths
}
