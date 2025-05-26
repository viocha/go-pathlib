package path

import (
	"errors"
	"fmt"
	"os"

	"github.com/viocha/go-pathlib/common"
)

var (
	ErrTargetExists = errors.New("target path already exists")
	ErrWalkCycle    = errors.New("walk cycle detected")
	ErrCopyDir      = errors.New("copy directory error")
	ErrCopyFile     = errors.New("copy file error")
)

func shouldStopWalk(err error) bool {
	return !errors.Is(err, nil) && !errors.Is(err, WalkSkip)
}

func Walk(root IPath, fn func(path IPath, err error) error, follow ...bool) error {
	isFollow := common.ParseOptional(follow, false) // 默认不跟随符号链接

	visited := make(map[string]int) // 用于检测循环，三色标记法：1表示正在访问，2表示访问结束
	return walkRecursively(root, fn, isFollow, visited)
}

func walkRecursively(root IPath, fn func(path IPath, err error) error, follow bool, visited map[string]int) error {
	// 转换成绝对路径
	rootAbs, err := root.ToAbs()
	if err != nil {
		return fn(root, err)
	}
	pathKey := rootAbs.String()
	if visited[pathKey] == 1 { // 重复访问，检测到循环
		return fn(rootAbs, common.WrapMsg(ErrWalkCycle, "cycle detected at %q", rootAbs))
	}
	if visited[pathKey] == 2 { // 已经访问结束
		return nil // 已经访问过，直接返回
	}
	visited[pathKey] = 1 // 标记为正在访问
	defer func() {
		visited[pathKey] = 2 // 函数返回时，标记为访问结束
	}()

	// 尝试解析符号链接
	if follow && root.IsLink() { // 如果跟随符号链接
		target, err := root.Resolve()
		if err != nil { // 可能目标路径不存在，或者无法转换成绝对路径
			return fn(root, err)
		}
		if rootAbs.IsRelTo(target, false) { // 回到了上级目录
			return fn(rootAbs, common.WrapMsg(ErrWalkCycle, "symbolic link %q points to a parent directory %q",
				rootAbs, target))
		}
		root = target // 使用解析后的路径继续遍历
	}

	// 先处理根路径
	err = fn(root, nil)
	if shouldStopWalk(err) {
		return err
	}

	// 不是目录，则已经访问结束
	if !root.IsDir(false) {
		return nil
	}

	// 如果是目录，继续遍历子目录
	if errors.Is(err, WalkSkip) { // 跳过当前目录
		return nil
	}
	children, err := root.ReadDir()
	if err != nil {
		return fn(root, err) // 读取目录失败，直接返回错误
	}
	for _, child := range children {
		err := walkRecursively(child, fn, follow, visited) // 递归遍历子路径
		if shouldStopWalk(err) {
			return err
		}
	}

	return nil
}

// src和dst必须是一个目录
func CopyMerge(src, dst IPath, mode MergeMode) error {
	children, err := src.ReadDir()
	if err != nil {
		return err
	}

	for _, child := range children {
		name := child.Name()
		childDst := dst.Join(name) // 将child 复制到 childDst
		// 目标路径存在，尝试进行解决冲突
		if childDst.Exists(false) {
			if childDst.IsDir(false) { // 目标是目录
				// 不允许使用文件覆盖目录
				if !child.IsDir(false) {
					return common.WrapMsg(ErrTargetExists, "target path %q is a directory but source path %q is not a directory",
						childDst, child)
				}
				// 如果源路径是目录，递归合并
				if err := CopyMerge(child, childDst, mode); err != nil {
					return err
				}
			}
			switch mode {
			case MergeModeError: // 返回错误
				return common.WrapMsg(ErrTargetExists, "target path %q already exists, cannot merge", childDst)
			case MergeModeSkip: // 跳过冲突
				continue
			case MergeModeReplace: // 替换冲突
				if err := childDst.Remove(); err != nil { // 删除目标文件或链接
					return err
				}
			}
		}
		// 目标路径不存在，使用 Copy 方法进行复制
		if err := child.Copy(childDst); err != nil {
			return err
		}
	}
	return nil
}

// 递归复制整个文件夹，目标文件夹必须不存在，且会自动创建父目录
func CopyDir(src, dst IPath) error {
	// 确保目标路径的父目录存在，以及目标路径不存在
	if err := ensureMove(dst, false); err != nil {
		return err
	}
	// 读取源目录的子目录和文件
	children, err := src.ReadDir()
	if err != nil {
		return err
	}
	// 先创建目标目录，再拷贝子目录和文件
	if err := dst.Mkdir(); err != nil {
		return err
	}
	for _, child := range children {
		name := child.Name()
		childDst := dst.Join(name)
		if child.IsLink() {
			if err := CopySymlink(child, childDst); err != nil {
				return err
			}
		} else if child.IsFile(false) {
			if err := CopyFile(child, childDst); err != nil {
				return err
			}
		} else if child.IsDir(false) {
			if err := CopyDir(child, childDst); err != nil { // 递归复制子目录
				return err
			}
		} else {
			return common.WrapMsg(ErrCopyDir, "unsupported path type for copy: %q", child)
		}
	}
	return nil
}

// 目标路径必须不存在，且会自动创建父目录
func CopyFile(src, dst IPath) error {
	if err := ensureMove(dst, false); err != nil {
		return err
	}

	inputFile, err := src.Open()
	if err != nil {
		return common.WrapSub(ErrCopyFile, err, "failed to open source file: %q", src)
	}
	defer closeFile(inputFile)

	outputFile, err := dst.OpenWrite()
	if err != nil {
		return common.WrapSub(ErrCopyFile, err, "failed to open target file: %q", dst)
	}
	defer closeFile(outputFile)

	if _, err = inputFile.WriteTo(outputFile); err != nil {
		return common.WrapSub(ErrCopyFile, err, "failed to copy file content from %q to %q", src, dst)
	}
	return nil
}

// 目标路径必须不存在，且会自动创建父目录
func CopySymlink(src, dst IPath) error {
	if err := ensureMove(dst, false); err != nil {
		return err
	}
	target, err := src.ReadLink()
	if err != nil {
		return err
	}
	// 如果是相对路径，需要计算相对于链接的相对路径
	if !target.IsAbs() {
		target, err = target.RelToFile(dst) // 基于dst链接文件的相对路径
		if err != nil {
			return err
		}
	}
	err = dst.Symlink(target, false)
	if err != nil {
		return err
	}
	return nil
}

// 确保目标路径不存在，以及目标路径父文件夹存在。目标存在时默认返回错误，replace可以允许删除已经存在的路径
func ensureMove(dst IPath, replace ...bool) error {
	isReplace := common.ParseOptional(replace, false) // 默认不允许替换

	if isReplace { // 如果允许替换，先删除目标路径
		err := dst.Remove()
		if err != nil {
			return err
		}
	}
	if dst.Exists(false) { // 不允许覆盖已存在的路径
		return common.WrapMsg(ErrTargetExists, "target: %q", dst)
	}
	// 确保目标的父目录存在
	if err := dst.Parent().EnsureDir(); err != nil {
		return err
	}
	return nil
}

// 关闭文件，处理可能的错误
func closeFile(inputFile *os.File) {
	err := inputFile.Close()
	if err != nil {
		fmt.Println(common.WrapMsg(err, "failed to close file after opening: %q", inputFile.Name()))
	}
}
