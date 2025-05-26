package path

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/viocha/go-pathlib/common"
)

func createTestFileTree(base string) error {
	/*
		./file
				├── empty
				├── dir
				│   ├── sub
				│   │   ├── x.md
				│   │   └── y.md
				│   ├── a.md
				│   └── b.md
				├── ldir (目录符号链接 -> dir)
				├── lnodir (目录符号链接 -> 不存在的文件 d:/tmp/nodir)
				├── f.md
				├── lnof.md (文件符号链接 -> 不存在的文件 d:/tmp/nofile)
				└── lf.md (文件符号链接 -> f.md)
	*/

	// 清理已经存在的file目录
	if err := cleanupTestFileTree(base); err != nil {
		return fmt.Errorf("failed to clean up existing test directory: %w", err)
	}

	// 创建目录结构
	paths := []string{
		filepath.Join(base, "empty"),
		filepath.Join(base, "dir", "sub"),
	}
	for _, p := range paths {
		if err := os.MkdirAll(p, os.ModePerm); err != nil {
			return err
		}
	}

	// 创建文件
	files := map[string]string{
		filepath.Join(base, "f.md"):               "f.md content",
		filepath.Join(base, "dir", "a.md"):        "a.md content",
		filepath.Join(base, "dir", "b.md"):        "b.md content",
		filepath.Join(base, "dir", "sub", "x.md"): "x.md content",
		filepath.Join(base, "dir", "sub", "y.md"): "y.md content",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), os.ModePerm); err != nil {
			return err
		}
	}

	// 创建文件符号链接 lf -> f.md
	fPath := filepath.Join("f.md") // 相对于链接的路径，而不是工作目录
	// fPath, _ = filepath.Abs(fPath) // 指向绝对路径
	lfPath := filepath.Join(base, "lf.md")
	if err := os.Symlink(fPath, lfPath); err != nil {
		return err
	}

	// 创建目录符号链接 ldir -> dir
	dirPath := filepath.Join("dir") // 相对于链接的路径，而不是工作目录
	// dirPath, _ = filepath.Abs(dirPath) // 指向绝对路径
	ldirPath := filepath.Join(base, "ldir")
	if err := os.Symlink(dirPath, ldirPath); err != nil {
		return err
	}

	// 创建不存在的目录符号链接 lnodir -> d:/tmp/nodir
	lnodirPath := filepath.Join(base, "lnodir")
	lnodirTarget := `d:\tmp\nodir`
	if err := os.Symlink(lnodirTarget, lnodirPath); err != nil {
		return err
	}

	// 创建不存在的文件符号链接 lnof.md -> d:/tmp/nofile
	lnofPath := filepath.Join(base, "lnof.md")
	lnofTarget := `d:\tmp\nofile`
	if err := os.Symlink(lnofTarget, lnofPath); err != nil {
		return err
	}

	return nil
}

func cleanupTestFileTree(base string) error {
	return os.RemoveAll(base)
}

func TestMain(m *testing.M) {
	base := "./file"

	// 创建用于测试的文件树
	if err := createTestFileTree(base); err != nil {
		if err := cleanupTestFileTree(base); err != nil {
			fmt.Printf("创建失败后，清理测试文件树也失败: %v\n", err)
		}
		panic(fmt.Sprintf("创建测试文件树失败: %v", err))
	}

	// 执行测试
	code := m.Run()

	// 清理测试文件
	if err := cleanupTestFileTree(base); err != nil {
		panic(fmt.Sprintf("清理测试文件树失败: %v", err))
	}

	os.Exit(code)
}

func TestWindowsPath_MustToAbs(t *testing.T) {
	testcases := []struct {
		path     string
		expected string
	}{
		{`./file/f.md`, `D:\IT\go\projects\go-pathlib\file\f.md`},
		{`./file/lf.md`, `D:\IT\go\projects\go-pathlib\file\lf.md`},
	}
	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			absPath := p.MustToAbs()
			if absPath.String() != tc.expected {
				t.Errorf("Expected absolute path %s, got %s", tc.expected, absPath.String())
			}
		})
	}
}

func TestWindowsPath_MustToURL(t *testing.T) {
	testcases := []struct {
		path     string
		expected string
	}{
		{`./file/f.md`, `file:///D:/IT/go/projects/go-pathlib/file/f.md`},
		{`//wsl$/Ubuntu/home/`, `file://wsl$/Ubuntu/home`},
		{`//wsl$/Ubuntu/`, `file://wsl$/Ubuntu/`},
	}
	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			url := p.MustToURL()
			if url != tc.expected {
				t.Errorf("Expected absolute path %s, got %s", tc.expected, url)
			}
		})
	}
}

func TestWindowsPath_MustReadLink(t *testing.T) {
	testcases := []struct {
		path     string
		expected string
	}{
		{`./file/ldir`, `file\dir`},
		{`./file/lf.md`, `file\f.md`},
	}
	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			target := p.MustReadLink()
			if target.String() != tc.expected {
				t.Errorf("Expected target path %s, got %s", tc.expected, target.String())
			}
		})
	}
}

func TestWindowsPath_MustResolve(t *testing.T) {
	testcases := []struct {
		path     string
		expected string
	}{
		{`./file/f.md`, `D:\IT\go\projects\go-pathlib\file\f.md`},
		{`./file/lf.md`, `D:\IT\go\projects\go-pathlib\file\f.md`},
	}
	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			resolved := p.MustResolve()
			if resolved.String() != tc.expected {
				t.Errorf("Expected resolved path %s, got %s", tc.expected, resolved.String())
			}
		})
	}
}

func TestWindowsPath_Exists(t *testing.T) {
	testcases := []struct {
		path   string
		follow bool
		exists bool
	}{
		{`./file/nonexist`, false, false},
		{`./file/lnof.md`, false, true},
		{`./file/lnof.md`, true, false},
		{`./file/lnodir`, true, false},
		{`./file/lnodir`, false, true},

		{`./file/f.md`, false, true},
		{`./file/lf.md`, false, true},
		{`./file/lf.md`, true, true},
		{`./file/dir`, false, true},
		{`./file/ldir`, false, true},
		{`./file/ldir`, true, true},
		{`./file/empty`, false, true},
	}
	for _, tc := range testcases {
		t.Run(fmt.Sprintf("Path: %s, Follow: %t", tc.path, tc.follow), func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			exists := p.Exists(tc.follow)
			if exists != tc.exists {
				t.Errorf("Expected Exists to be %t, got %t", tc.exists, exists)
			}
		})
	}
}

func TestWindowsPath_IsDir(t *testing.T) {
	testcases := []struct {
		path   string
		follow bool
		isDir  bool
	}{
		{`./file/dir`, false, true},
		{`./file/empty`, false, true},
		{`./file/nonexist`, false, false},
		{`./file/ldir`, false, false},
		{`./file/ldir`, true, true},
	}
	for _, tc := range testcases {
		t.Run(fmt.Sprintf("Path: %s, Follow: %t", tc.path, tc.follow), func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			isDir := p.IsDir(tc.follow)
			if isDir != tc.isDir {
				t.Errorf("Expected IsDir to be %t, got %t", tc.isDir, isDir)
			}
		})
	}
}

func TestWindowsPath_IsFile(t *testing.T) {
	testcases := []struct {
		path   string
		follow bool
		isFile bool
	}{
		{`./file/nonexist`, false, false},
		{`./file/f.md`, false, true},
		{`./file/lf.md`, false, false},
		{`./file/lf.md`, true, true},
		{`./file/dir`, false, false},
		{`./file/ldir`, false, false},
		{`./file/ldir`, true, false},
		{`./file/empty`, false, false},
	}
	for _, tc := range testcases {
		t.Run(fmt.Sprintf("Path: %s, Follow: %t", tc.path, tc.follow), func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			isFile := p.IsFile(tc.follow)
			if isFile != tc.isFile {
				t.Errorf("Expected IsFile to be %t, got %t", tc.isFile, isFile)
			}
		})
	}
}

func TestWindowsPath_IsSymlink(t *testing.T) {
	testcases := []struct {
		path      string
		isSymlink bool
	}{
		{`./file/nonexist`, false},
		{`./file/ldir`, true},
		{`./file/lf.md`, true},
		{`./file/dir`, false},
		{`./file/f.md`, false},
	}
	for _, tc := range testcases {
		t.Run(fmt.Sprintf("Path: %s", tc.path), func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			isSymlink := p.IsLink()
			if isSymlink != tc.isSymlink {
				t.Errorf("Expected IsSymlink to be %t, got %t", tc.isSymlink, isSymlink)
			}
		})
	}
}

func TestWindowsPath_SameFile(t *testing.T) {
	testcases := []struct {
		path1 string
		path2 string
		same  bool
	}{
		{`./file/f.md`, `./file/lf.md`, true},
		{`./file/ldir`, `./file/dir`, true},
		{`./file/dir/a.md`, `./file/dir/b.md`, false},
		{`./file/dir/a.md`, `./file/ldir/a.md`, true},
		{`./file/dir/a.md`, `./file/ldir/b.md`, false},
	}
	for _, tc := range testcases {
		t.Run(fmt.Sprintf("Path1: %s, Path2: %s", tc.path1, tc.path2), func(t *testing.T) {
			p1 := NewWindowsPath(tc.path1)
			p2 := NewWindowsPath(tc.path2)
			same := p1.SameFile(p2)
			if same != tc.same {
				t.Errorf("Expected SameFile to be %t, got %t", tc.same, same)
			}
		})
	}
}

func TestWindowsPath_MustOpen(t *testing.T) {
	testcases := []struct {
		path     string
		expected string
	}{
		{`./file/f.md`, `f.md content`},
		{`./file/lf.md`, `f.md content`},
	}
	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			file := p.MustOpen()
			defer func() {
				if err := file.Close(); err != nil {
					t.Fatalf("Failed to close file: %v", err)
				}
			}()

			content := make([]byte, 1024)
			n, err := file.Read(content)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
				return
			}

			if string(content[:n]) != tc.expected {
				t.Errorf("Expected file content %s, got %s", tc.expected, string(content[:n]))
			}
		})
	}
}

func TestWindowsPath_MustRead(t *testing.T) {
	testcases := []struct {
		path     string
		expected string
	}{
		{`./file/f.md`, `f.md content`},
		{`./file/lf.md`, `f.md content`},
	}
	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			content := p.MustRead()
			if content != tc.expected {
				t.Errorf("Expected file content %s, got %s", tc.expected, content)
			}
		})
	}
}

func TestWindowsPath_Write(t *testing.T) {
	testcases := []struct {
		path     string
		content  string
		expected string
	}{
		{`./file/f.md`, `new content`, `new content`},
		{`./file/lf.md`, `new content`, `new content`},
	}
	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			_ = p.Write(tc.content)
			content := p.MustRead()
			if content != tc.expected {
				t.Errorf("Expected file content %s, got %s", tc.expected, content)
			}
			_ = p.Write("f.md content") // 恢复原内容
		})
	}
}

func TestWindowsPath_MustReadBytes(t *testing.T) {
	testcases := []struct {
		path     string
		expected []byte
	}{
		{`./file/f.md`, []byte(`f.md content`)},
		{`./file/lf.md`, []byte(`f.md content`)},
	}
	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			content := p.MustReadBytes()
			if string(content) != string(tc.expected) {
				t.Errorf("Expected file content %s, got %s", string(tc.expected), string(content))
			}
		})
	}
}

func TestWindowsPath_WriteBytes(t *testing.T) {
	testcases := []struct {
		path     string
		content  []byte
		expected []byte
	}{
		{`./file/f.md`, []byte(`new byte content`), []byte(`new byte content`)},
		{`./file/lf.md`, []byte(`new byte content`), []byte(`new byte content`)},
	}
	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			_ = p.WriteBytes(tc.content)
			content := p.MustReadBytes()
			if string(content) != string(tc.expected) {
				t.Errorf("Expected file content %s, got %s", string(tc.expected), string(content))
			}
			_ = p.WriteBytes([]byte("f.md content")) // 恢复原内容
		})
	}
}

func TestWindowsPath_Create(t *testing.T) {
	testcases := []struct {
		path    string
		parents bool
	}{
		{`./file/newfile.md`, false},
		{`./file/newdir/newfile.md`, true},
	}
	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			err := p.Create(tc.parents)
			if err != nil {
				t.Fatalf("Failed to create path: %v", err)
			}
			_ = os.RemoveAll(p.String()) // 清理创建的文件或目录
		})
	}
	_ = os.RemoveAll("./file/newdir") // 清理创建的目录
}

func TestWindowsPath_Mkdir(t *testing.T) {
	testcases := []struct {
		path    string
		parents bool
	}{
		{`./file/newdir`, false},
		{`./file/newdir/subdir`, true},
	}
	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			err := p.Mkdir(tc.parents)
			if err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}
			_ = os.RemoveAll(p.String()) // 清理创建的目录
		})
	}
}

func TestWindowsPath_Symlink(t *testing.T) {
	testcases := []struct {
		path    string
		parents bool
		target  string
	}{
		{`./file/link-to-file`, false, `./f.md`},
		{`./file/newdir1234/link-to-file`, true, `./f.md`},
		{`./file/link-to-dir`, false, `./dir`},
		{`./file/newdir4567/link-to-dir`, true, `./dir`},
	}
	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			targetPath := NewWindowsPath(tc.target)
			err := p.Symlink(targetPath, tc.parents)
			if err != nil {
				t.Fatalf("Failed to create symlink: %v", err)
			}
			_ = os.Remove(p.String()) // 清理创建的符号链接
		})
	}
}

func TestWindowsPath_Remove(t *testing.T) {
	testcases := []struct {
		path      string
		recursive bool
	}{
		{`./file/newfile.md`, false},       // 删除空目录
		{`./file/newdir/newfile.md`, true}, // 递归删除目录
	}
	// 创建测试文件和目录
	allErrors := errors.Join(
		os.MkdirAll(`./file/newdir`, os.ModePerm),
		os.WriteFile(`./file/newfile.md`, []byte("test content"), os.ModePerm),
		os.WriteFile(`./file/newdir/newfile.md`, []byte("test content"), os.ModePerm),
	)
	if allErrors != nil {
		t.Fatalf("Failed to create test files: %v", allErrors)
	}

	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			err := p.Remove(tc.recursive)
			if err != nil {
				t.Fatalf("Failed to remove path: %v", err)
			}
			if p.Exists(false) { // 确认路径已被删除
				t.Errorf("Path %s still exists after removal", p.String())
			}
		})
	}
}

func TestWindowsPath_RemoveLink(t *testing.T) {
	testcases := []struct {
		path      string
		recursive bool
	}{
		{`./file/file-link.md`, false}, // 删除符号链接文件
		{`./file/dir-link`, false},     // 删除符号链接目录
	}
	// 创建测试符号链接
	_ = NewWindowsPath(`./file/file-link.md`).Symlink(NewWindowsPath(`./f.md`))
	_ = NewWindowsPath(`./file/dir-link`).Symlink(NewWindowsPath(`./dir`))

	for _, tc := range testcases {
		t.Run(tc.path, func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			truePath := p.MustResolve()
			err := p.Remove(tc.recursive)
			if err != nil {
				t.Fatalf("Failed to remove path: %v", err)
			}
			if p.Exists(false) { // 确认路径已被删除
				t.Errorf("Path %s still exists after removal", p.String())
			}
			if !truePath.Exists(false) { // 确认真实路径仍然存在
				t.Errorf("True path %s does not exist after removing symlink %s", truePath.String(), p.String())
			}
		})
	}
}

func TestWindowsPath_MustRename(t *testing.T) {
	testcases := []struct {
		path    string
		newName string
	}{
		{`./file/f.md`, `new_f.md`},
		{`./file/lf.md`, `new_lffff.md`},
		{`./file/dir`, `new_dir`},
	}
	for _, tc := range testcases {
		t.Run(fmt.Sprintf("Path: %s, NewName: %s", tc.path, tc.newName), func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			newPath := p.MustRename(tc.newName)
			fmt.Printf("%q => %q\n", p, newPath)
			if !newPath.Exists(false) {
				t.Errorf("New path %s does not exist after renaming", newPath.String())
			}
			if p.Exists(false) {
				t.Errorf("Original path %s still exists after renaming", p.String())
			}
			err := newPath.Move(p) // 恢复原路径
			if err != nil {
				t.Fatalf("Failed to restore original path: %v", err)
			}
		})
	}
}

func TestWindowsPath_Move(t *testing.T) {
	testcases := []struct {
		path    string
		newPath string
	}{
		{`./file/f.md`, `./file/new_fff.md`},
		{`./file/lf.md`, `./file/new_lfff.md`},
		{`./file/dir`, `./file/new_dirrrr`},
	}
	for _, tc := range testcases {
		t.Run(fmt.Sprintf("Path: %s, NewPath: %s", tc.path, tc.newPath), func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			newP := NewWindowsPath(tc.newPath)
			if !p.Exists(false) {
				t.Fatalf("Original path %s does not exist", p.String())
			}
			err := p.Move(newP, false)
			if err != nil {
				t.Fatalf("Failed to move path: %v", err)
			}
			if !newP.Exists(false) {
				t.Errorf("New path %s does not exist after moving", newP.String())
			}
			if p.Exists(false) {
				t.Errorf("Original path %s still exists after moving", p.String())
			}
			_ = newP.Move(p) // 恢复原路径
		})
	}
}

func TestWindowsPath_MoveDir(t *testing.T) {
	testcases := []struct {
		path    string
		newPath string
	}{
		{`./file/dir`, `./file/new_dir`},
	}
	for _, tc := range testcases {
		t.Run(fmt.Sprintf("Path: %s, NewPath: %s", tc.path, tc.newPath), func(t *testing.T) {
			p := NewWindowsPath(tc.path)
			newP := NewWindowsPath(tc.newPath)
			err := p.Move(newP, false)
			if err != nil {
				t.Fatalf("Failed to move path: %v", err)
			}
			if !newP.Exists(false) {
				t.Errorf("New path %s does not exist after moving", newP.String())
			}
			if !newP.Join("a.md").Exists(false) {
				t.Errorf("New path %s does not contain expected file after moving", newP.String())
			}
			if p.Exists(false) {
				t.Errorf("Original path %s still exists after moving", p.String())
			}
			_ = os.Rename(newP.String(), p.String()) // 恢复原路径
		})
	}
}

func TestWindowsPath_Copy(t *testing.T) {
	src := NewWindowsPath(`./file/dir`)
	dst := NewWindowsPath(`./file/copy_dir`)
	err := src.Copy(dst)
	if err != nil {
		t.Fatalf("Failed to copy directory: %v", err)
	}
	if !dst.Join("a.md").Exists() {
		t.Errorf("Copied directory %s does not contain expected file", dst.String())
	}
	if dst.Join("b.md").MustRead() != "b.md content" {
		t.Errorf("Copied directory %s does not contain expected file content", dst.String())
	}
	if !src.Join("a.md").Exists() {
		t.Errorf("Source directory %s does not contain expected file after copy", src.String())
	}
	if src.Join("b.md").MustRead() != "b.md content" {
		t.Errorf("Source directory %s does not contain expected file content", dst.String())
	}
}

func TestWindowsPath_Glob(t *testing.T) {
	// 创建一个指向上级目录的符号链接
	err := NewWindowsPath("./file/link-to-parent").Symlink(NewWindowsPath("."), true)
	if err != nil {
		t.Fatalf("Failed to create symlink ./file/link-to-parent : %v", err)
	}
	t.Run("Glob without follow symlinks", func(t *testing.T) {
		p := NewWindowsPath(`./file/dir`)
		matches := p.MustGlob("**/*.md")
		t.Logf("Matched files: %v", matches)
		matchesStr := make([]string, len(matches))
		for i, match := range matches {
			matchesStr[i] = match.String()
		}
		if !slices.Contains(matchesStr, "file\\dir\\sub\\y.md") {
			t.Errorf("Expected to find file 'file/dir/sub/y.md' in matches, but it was not found")
		}
	})
	t.Run("Glob with follow symlinks", func(t *testing.T) {
		p := NewWindowsPath(`./file/link-to-parent`)
		matches := p.MustGlob("**/*.md", GlobOptions{Follow: true, SkipOnCycle: true})
		t.Logf("Matched files: %v", matches)
	})
}

func TestWindowsPath_Walk(t *testing.T) {
	p := NewWindowsPath(`./file`)
	t.Run("Walk without follow symlinks", func(t *testing.T) {
		err := p.Walk(func(path IPath, err error) error {
			t.Logf("Visited path: %s", path.String())
			return err
		})
		if err != nil {
			t.Fatalf("Failed to walk path: %v", err)
		}
	})
	t.Run("Walk with follow symlinks", func(t *testing.T) {
		// 创建一个指向上级目录的符号链接
		err := NewWindowsPath("./file/parent_link").Symlink(NewWindowsPath("."), true)
		if err != nil {
			t.Fatalf("Failed to create symlink ./file/parent_link :\n\t %v", err)
		}
		err = p.Walk(func(path IPath, err error) error {
			if err != nil { // 会出现读取链接失败的错误，目标不存在
				t.Logf("Error walking path %s:\n\t %v", path.String(), err)
				return nil // 继续遍历
			}
			t.Logf("Visited path: %s", path.String())
			if path.Name() == "sub" {
				t.Logf("Skipping sub directory: %s", path.String())
				return WalkSkip // 跳过子目录的向下遍历
			}
			return err
		}, true)
		if err != nil {
			t.Fatalf("Failed to walk path: %v", err)
		}
	})
	t.Run("Walk with stop", func(t *testing.T) {
		err := p.Walk(func(path IPath, err error) error {
			if err != nil { // 会出现读取链接失败的错误，目标不存在
				t.Log(common.WrapMsg(err, "error walking path: %q", path))
				return nil // 继续遍历
			}
			t.Logf("Visited path: %q", path)
			if path.Name() == "sub" {
				t.Logf("Stopping walk at sub directory: %q", path)
				return WalkStop
			}
			return err
		}, true)
		if !errors.Is(err, WalkStop) {
			t.Fatalf("Failed to walk path: %v", err)
		}
	})
}
