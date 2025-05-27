package purepath

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	nt "github.com/viocha/go-pathlib/purepath/ntpath"
)

func runTask[T any, R any](t *testing.T, testcases []struct {
	input  T
	output R
}, f func(T) R) {
	for _, tc := range testcases {
		t.Run(fmt.Sprintf("input: %v", tc.input), func(t *testing.T) {
			result := f(tc.input)
			if !reflect.DeepEqual(result, tc.output) {
				t.Errorf("Expected %v , got %v", tc.output, result)
			}
		})
	}
}

func TestNewPureWindowsPath(t *testing.T) {
	runTask(t, []struct {
		input  []string
		output string
	}{
		// 绝对路径覆盖前面的路径
		{[]string{"c:/Windows", "d:bar"}, "d:bar"},
		{[]string{"c:/Windows", "c:/Program Files"}, `c:\Program Files`},
		{[]string{"a/b", "//server/share/file"}, `\\server\share\file`},
		{[]string{"c:/hello", "/hi/abc"}, `c:\hi\abc`},
		// 带盘符的相对路径
		{[]string{"c:foo/bar", "c:abc"}, `c:foo\bar\abc`},
		// 处理空路径
		{[]string{}, `.`},
		{[]string{""}, `.`},
		// 处理.和..路径
		{[]string{`a/./b`}, `a\b`},
		{[]string{`c:.`}, `c:`},
		{[]string{`a/../b`}, `b`},
		{[]string{`a`, `../b`}, `b`},
		{[]string{`../b`}, `..\b`},
		// 斜杠去重，首尾斜杠移除
		{[]string{`./a//b\.`}, `a\b`},
		{[]string{`./a\\b/`}, `a\b`},
		{[]string{"c:/foo/"}, `c:\foo`},
		{[]string{"foo/bar/"}, `foo\bar`},
		{[]string{"//foo/bar/file/"}, `\\foo\bar\file`},
		// 锚点路径斜杠规范化
		{[]string{"c:/"}, `c:\`},
		{[]string{"c:"}, `c:`},
		{[]string{`//server/share/`}, `\\server\share\`},
		{[]string{`//server/share`}, `\\server\share\`},
	}, func(input []string) string {
		return NewPureWindowsPath(input...).String()
	})
}

func TestPureWindowsPath_Parts(t *testing.T) {
	runTask(t, []struct {
		input  string
		output []string
	}{
		{"", nil},
		{"//resource/share", []string{`\\resource\share\`}},
		{"c:/", []string{`c:\`}},
		{"/", []string{`\`}},
		{"/a", []string{`\`, "a"}},
		{"//resource/share/path", []string{`\\resource\share\`, "path"}},
		{"c:/foo", []string{`c:\`, "foo"}},
		{"c:foo/bar/", []string{`c:`, "foo", "bar"}},
		{"/foo/bar/", []string{`\`, "foo", "bar"}},
		{"foo/bar/", []string{"foo", "bar"}},
	}, func(input string) []string {
		return NewPureWindowsPath(input).Parts()
	})
}
func TestPureWindowsPath_Drive(t *testing.T) {
	runTask(t, []struct {
		input  string
		output string
	}{
		{"c:/foo/bar", "c:"},
		{"c:foo/bar", "c:"},
		{"//server/share/file", `\\server\share`},
		{"//server/share/", `\\server\share`},
		{"/foo/bar", ""},
		{"foo/bar", ""},
		{"", ""},
	}, func(input string) string {
		return NewPureWindowsPath(input).Drive()
	})
}

func TestPureWindowsPath_Root(t *testing.T) {
	runTask(t, []struct {
		input  string
		output string
	}{
		{"c:/foo/bar", `\`},
		{"c:foo/bar", ``},
		{"//server/share/file", `\`},
		{"//server/share", `\`},
		{"/foo/bar", `\`},
		{"foo/bar", ""},
		{"", ""},
	}, func(input string) string {
		return NewPureWindowsPath(input).Root()
	})
}

func TestPureWindowsPath_Anchor(t *testing.T) {
	runTask(t, []struct {
		input  string
		output string
	}{
		{"c:/foo/bar", `c:\`},
		{"c:foo/bar", `c:`},
		{"//server/share/file", `\\server\share\`},
		{"//server/share", `\\server\share\`},
		{"/foo/bar", `\`},
		{"foo/bar", ""},
		{"", ""},
	}, func(input string) string {
		return NewPureWindowsPath(input).Anchor()
	})
}

func TestPureWindowsPath_Parent(t *testing.T) {
	runTask(t, []struct {
		input  string
		output string
	}{
		{"c:/foo/bar", `c:\foo`},
		{"c:foo/bar", `c:foo`},
		{"//server/share/file", `\\server\share\`},
		{"//server/share/", `\\server\share\`},
		{"/foo/bar", `\foo`},
		{"foo/bar", "foo"},
		{"./file/newdir/newfile.md", `file\newdir`},
		{"", "."},
		{".", "."},
		{"a", "."},
	}, func(input string) string {
		return NewPureWindowsPath(input).Parent().String()
	})
}

func TestPureWindowsPath_Parents(t *testing.T) {
	runTask(t, []struct {
		input  string
		output []string
	}{
		{"c:/foo/bar", []string{`c:\foo`, `c:\`}},
		{"c:foo/bar", []string{`c:foo`, `c:`}},
		{"//server/share/file/foo", []string{`\\server\share\file`, `\\server\share\`}},
		{"//server/share/", nil},
		{"/foo/bar", []string{`\foo`, `\`}},
		{"foo/bar/baz", []string{`foo\bar`, "foo", "."}},
		{"", nil},
		{".", nil},
	}, func(input string) []string {
		parentsPath := NewPureWindowsPath(input).Parents()
		var result []string
		for _, p := range parentsPath {
			result = append(result, p.String())
		}
		return result
	})
}

func TestPureWindowsPath_MustWithParent(t *testing.T) {
	testcases := []struct {
		input  string
		parent string
		output string
	}{
		{"c:/foo/bar", "c:/foo/baz", `c:\foo\baz\bar`},
		{"./a/b", "/", `\b`},
	}
	for _, tc := range testcases {
		t.Run(tc.input+" with parent "+tc.parent, func(t *testing.T) {
			path := NewPureWindowsPath(tc.input)
			parentPath := NewPureWindowsPath(tc.parent)
			result := path.MustWithParent(parentPath)
			if result.String() != tc.output {
				t.Errorf("Expected %v, got %v", tc.output, result.String())
			}
		})
	}
}

func TestPureWindowsPath_Name(t *testing.T) {
	runTask(t, []struct {
		input  string
		output string
	}{
		{"c:/foo/bar", "bar"},
		{"c:foo/bar", "bar"},
		{"c:", ""},
		{"c:/", ""},
		{"//server/share/file", "file"},
		{"//server/share/", ""},
		{"/foo/bar", "bar"},
		{"/foo/", "foo"},
		{"foo/bar", "bar"},
		{".", ""},
		{"", ""},
		{"a", "a"},
	}, func(input string) string {
		return NewPureWindowsPath(input).Name()
	})
}

func TestPureWindowsPath_Suffix(t *testing.T) {
	runTask(t, []struct {
		input  string
		output string
	}{
		{"c:/foo/bar.out.txt", ".txt"},
		{"c:foo/bar.tmp.txt", ".txt"},
		{"c:/foo/bar", ""},
		{"c:foo/bar", ""},
		{"//server/share/file.txt", ".txt"},
		{"//server/share/file", ""},
		{"/foo/bar.txt", ".txt"},
		{"/foo/bar", ""},
		{"foo/bar.txt", ".txt"},
		{"foo/bar", ""},
		{".", ""},
		{"", ""},
	}, func(input string) string {
		return NewPureWindowsPath(input).Suffix()
	})
}

func TestPureWindowsPath_Suffixes(t *testing.T) {
	runTask(t, []struct {
		input  string
		output []string
	}{
		{"c:/foo/bar.out.txt", []string{".out", ".txt"}},
		{"c:foo/bar.tmp.txt", []string{".tmp", ".txt"}},
		{"c:/foo/bar", nil},
		{"c:foo/bar", nil},
		{"//server/share/file.txt", []string{".txt"}},
		{"//server/share/file", nil},
		{"/foo/bar.txt", []string{".txt"}},
		{"/foo/bar", nil},
		{"foo/bar.txt", []string{".txt"}},
		{"", nil},
		{".", nil},
	}, func(input string) []string {
		return NewPureWindowsPath(input).Suffixes()
	})
}

func TestPureWindowsPath_Stem(t *testing.T) {
	runTask(t, []struct {
		input  string
		output string
	}{
		{"c:/foo/bar.out.txt", "bar.out"},
		{"c:foo/bar.tmp.txt", "bar.tmp"},
		{"c:/foo/bar", "bar"},
		{"c:foo/bar.txt", "bar"},
		{"", ""},
		{".", ""},
	}, func(input string) string {
		return NewPureWindowsPath(input).Stem()
	})
}

func TestPureWindowsPath_AsPosix(t *testing.T) {
	runTask(t, []struct {
		input  string
		output string
	}{
		{"c:/foo/bar", "c:/foo/bar"},
		{"c:foo/bar", "c:foo/bar"},
		{"//server/share/file", "//server/share/file"},
		{"//server/share/", "//server/share/"},
		{"/foo/bar", "/foo/bar"},
		{"foo/bar", "foo/bar"},
		{".", "."},
		{"", "."},
	}, func(input string) string {
		return NewPureWindowsPath(input).ToPosix()
	})
}

func TestPureWindowsPath_IsAbsolute(t *testing.T) {
	runTask(t, []struct {
		input  string
		output bool
	}{
		{"c:/foo/bar", true},
		{"c:foo/bar", false},
		{"//server/share/file", true},
		{"//server/share/", true},
		{"/foo/bar", false},
		{"foo/bar", false},
		{".", false},
		{"", false},
	}, func(input string) bool {
		return NewPureWindowsPath(input).IsAbs()
	})
}

func TestPureWindowsPath_IsRelativeTo(t *testing.T) {
	testcases := []struct {
		input  string
		other  string
		output bool
	}{
		{"c:/foo/bar", "c:", false},
		{"c:foo/bar", "c:/", false},
		{"//server/share/file", "//Server/Share", true},
		{"//server/share/", "//server/share", true},
		{"/foo/bar", "/FOO", true},
		{"/foo/bar", "/", true},
		{"foo/bar", "FOO", true},
		{"a", ".", true},
		{"", ".", true},
	}

	for _, tc := range testcases {
		t.Run(tc.input+" relative to "+tc.other, func(t *testing.T) {
			path := NewPureWindowsPath(tc.input)
			otherPath := NewPureWindowsPath(tc.other)
			result := path.IsRelTo(otherPath)
			if result != tc.output {
				t.Errorf("Expected %v, got %v", tc.output, result)
			}
		})
	}
}

func TestPureWindowsPath_JoinPath(t *testing.T) {
	runTask(t, []struct {
		input  []string
		output string
	}{
		{[]string{"c:/foo", "c:bar"}, `c:\foo\bar`},
		{[]string{"c:foo", "c:bar"}, `c:foo\bar`},
		{[]string{"d:/Windows", "c:/"}, `c:\`},
		{[]string{"d:/Windows", "c:"}, `c:`},
		{[]string{"c:/", `//a/b/c`}, `\\a\b\c`},
		{[]string{"a/b", "c"}, `a\b\c`},
		{[]string{".", "a/b"}, `a\b`},
		{[]string{"a/b", "."}, `a\b`},
		{[]string{"", "."}, `.`},
		{[]string{".", ""}, `.`},
		{[]string{"c:", ""}, `c:`},
	}, func(input []string) string {
		return NewPureWindowsPath(input[0]).Join(input[1:]...).String()
	})
}

func TestPureWindowsPath_FullMatch(t *testing.T) {
	testcases := []struct {
		input         string
		pattern       string
		caseSensitive bool
		output        bool
	}{
		{"//foo/bar/file.txt", "/a/foo/bar/*.txt", false, false},
		{"//foo/bar/file.txt", "//**/*.txt", false, false},
		{"c:/foo/bar.txt", "c:/foo/bar.txt", false, true},
		{"c:/foo/bar.txt", "C:/FOO/BAR.TXT", false, true},
		{"c:/foo/bar.txt", "C:/FOO/BAR.TXT", true, false},
		{"c:/foo/bar.txt", "c:/foo/baz.txt", false, false},
		{"c:/foo/bar/baz.txt", "c:/**/*.txt", false, true},
		{"c:/foo/bar.txt", "c:/foo/*.txt", false, true},
		{"c:/foo/bar.txt", "c:/foo/*.*", false, true},
		{"c:/foo/bar.txt", "c:/**", false, true},
		{"foo", "./foo", false, true},
		{"foo/bar", "bar", false, false},
	}
	for _, tc := range testcases {
		t.Run(tc.input+" matches "+tc.pattern, func(t *testing.T) {
			path := NewPureWindowsPath(tc.input)
			result := path.FullMatch(tc.pattern, tc.caseSensitive)
			if result != tc.output {
				t.Errorf("Expected %v, got %v", tc.output, result)
			}
		})
	}
}

func TestPureWindowsPath_Match(t *testing.T) {
	testcases := []struct {
		input         string
		pattern       string
		caseSensitive bool
		output        bool
	}{
		{"c:/foo/bar.txt", "bar.txt", false, true},
		{"c:foo/bar.txt", "BAR.TXT", false, true},
		{"c:foo/bar", "c:bar", false, false},
		{"c:/foo/bar", "c:bar", false, false},
		{"/foo/bar", "./bar", false, true},
		{"/foo/bar", "a/../bar", false, true},
		{"/foo/bar", ".", false, true},
		{"/foo/BAR", "bar", true, false},
	}
	for _, tc := range testcases {
		t.Run(tc.input+" matches "+tc.pattern, func(t *testing.T) {
			path := NewPureWindowsPath(tc.input)
			result := path.Match(tc.pattern, tc.caseSensitive)
			if result != tc.output {
				t.Errorf("Expected %v, got %v", tc.output, result)
			}
		})
	}
}

func TestPureWindowsPath_WithSuffix(t *testing.T) {
	testcases := []struct {
		input  string
		suffix string
		output string
		err    error
	}{
		{"c:/foo/bar.txt", ".md", `c:\foo\bar.md`, nil},
		{"c:foo/bar.txt", ".md", `c:foo\bar.md`, nil},
		{"/foo/bar", ".js", `\foo\bar.js`, nil},
		{"foo/bar.ts", ".js", `foo\bar.js`, nil},
		{"foo/bar.out.ts", ".ts", `foo\bar.out.ts`, nil},
		{"foo/bar", "js", ``, nt.ErrInvalidSuffix},
		{"/", ".js", ``, nt.ErrNoName},
		{"/abc", ".j:s", ``, nt.ErrInvalidName},
	}
	for _, tc := range testcases {
		t.Run(tc.input+" with suffix "+tc.suffix, func(t *testing.T) {
			path := NewPureWindowsPath(tc.input)
			result, err := path.WithSuffix(tc.suffix)
			if err != nil && tc.err == nil {
				t.Errorf("Expected no error, got %v", err)
			} else if err == nil && tc.err != nil {
				t.Errorf("Expected error %v, got nil", tc.err)
			} else if err != nil && tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Errorf("Expected error %v, got %v", tc.err, err)
				}
			} else if result.String() != tc.output {
				t.Errorf("Expected %v, got %v", tc.output, result.String())
			}
		})
	}
}

func TestPureWindowsPath_WithStem(t *testing.T) {
	testcases := []struct {
		input  string
		stem   string
		output string
		err    error
	}{
		{"c:/foo/bar.txt", "baz", `c:\foo\baz.txt`, nil},
		{"c:foo/bar.txt", "baz", `c:foo\baz.txt`, nil},
		{"/foo/bar", "baz", `\foo\baz`, nil},
		{"foo/bar.ts", "baz", `foo\baz.ts`, nil},
		{"foo/bar.out.ts", "baz", `foo\baz.ts`, nil},
		{"/", "bar.js", ``, nt.ErrNoName},
		{"/abc", "b/ar.js", ``, nt.ErrInvalidName},
	}
	for _, tc := range testcases {
		t.Run(tc.input+" with stem "+tc.stem, func(t *testing.T) {
			path := NewPureWindowsPath(tc.input)
			result, err := path.WithStem(tc.stem)
			if err != nil && tc.err == nil {
				t.Errorf("Expected no error, got %v", err)
			} else if err == nil && tc.err != nil {
				t.Errorf("Expected error %v, got nil", tc.err)
			} else if err != nil && tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Errorf("Expected error %v, got %v", tc.err, err)
				}
			} else if result.String() != tc.output {
				t.Errorf("Expected %v, got %v", tc.output, result.String())
			}
		})
	}
}

func TestPureWindowsPath_WithName(t *testing.T) {
	testcases := []struct {
		input  string
		name   string
		output string
		err    error
	}{
		{"c:/foo/bar.txt", "baz.txt", `c:\foo\baz.txt`, nil},
		{"c:foo/bar.txt", "baz.txt", `c:foo\baz.txt`, nil},
		{"/foo/bar", "baz.js", `\foo\baz.js`, nil},
		{"foo/bar.ts", "baz.ts", `foo\baz.ts`, nil},
		{"foo/bar.out.ts", "baz.ts", `foo\baz.ts`, nil},
		{"/", "bar.js", ``, nt.ErrNoName},
		{"/abc", "b/ar.js", ``, nt.ErrInvalidName},
	}
	for _, tc := range testcases {
		t.Run(tc.input+" with name "+tc.name, func(t *testing.T) {
			path := NewPureWindowsPath(tc.input)
			result, err := path.WithName(tc.name)
			if err != nil && tc.err == nil {
				t.Errorf("Expected no error, got %v", err)
			} else if err == nil && tc.err != nil {
				t.Errorf("Expected error %v, got nil", tc.err)
			} else if err != nil && tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Errorf("Expected error %v, got %v", tc.err, err)
				}
			} else if result.String() != tc.output {
				t.Errorf("Expected %v, got %v", tc.output, result.String())
			}
		})
	}
}

func TestPureWindowsPath_RelativeTo(t *testing.T) {
	testcases := []struct {
		input  string
		other  string
		walkUp bool
		output string
		err    error
	}{
		{"c:/foo/bar.txt", "c:/foo", false, "bar.txt", nil},
		{"c:foo/bar.txt", "c:foo", false, "bar.txt", nil},
		{"//server/share/file.txt", "//server/share", false, "file.txt", nil},
		{"/a/b", "/", false, `a\b`, nil},
		{"a/b", ".", false, `a\b`, nil},
		{"/a/b", "", false, ``, nt.ErrNotRelative},
		{"a/b", "c", true, ``, nt.ErrNotRelative},
		{"/a/b", "/c", true, `..\a\b`, nil},
		{"c:/foo/bar.txt", "c:/a/b", true, `..\..\foo\bar.txt`, nil},
	}
	for _, tc := range testcases {
		t.Run(tc.input+" relative to "+tc.other, func(t *testing.T) {
			path := NewPureWindowsPath(tc.input)
			otherPath := NewPureWindowsPath(tc.other)
			result, err := path.RelTo(otherPath, tc.walkUp)
			if err != nil && tc.err == nil {
				t.Errorf("Expected no error, got %v", err)
			} else if err == nil && tc.err != nil {
				t.Errorf("Expected error %v, got nil", tc.err)
			} else if err != nil && tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Errorf("Expected error %v, got %v", tc.err, err)
				}
			} else if result.String() != tc.output {
				t.Errorf("Expected %v, got %v", tc.output, result.String())
			}
		})
	}
}

func TestPureWindowsPath_WithAnchor(t *testing.T) {
	testcases := []struct {
		input  string
		anchor string
		output string
		err    error
	}{
		{"c:/foo/bar.txt", "d:/", `d:\foo\bar.txt`, nil},
		{"c:/foo/bar.txt", "c:", `c:foo\bar.txt`, nil},
		{"c:/foo/bar.txt", "/", `\foo\bar.txt`, nil},
		{"c:/foo/bar.txt", "", `foo\bar.txt`, nil},
		{"c:/foo/bar.txt", "//server/share/", `\\server\share\foo\bar.txt`, nil},
		{"c:/foo/bar.txt", "//server/share", ``, nt.ErrInvalidAnchor},
	}
	for _, tc := range testcases {
		t.Run(tc.input+" with anchor "+tc.anchor, func(t *testing.T) {
			path := NewPureWindowsPath(tc.input)
			result, err := path.WithAnchor(tc.anchor)
			if result != nil {
				fmt.Println(result.String())
			} else {
				fmt.Println("result is nil")
			}
			if err != nil && tc.err == nil {
				t.Errorf("Expected no error, got %v", err)
			} else if err == nil && tc.err != nil {
				t.Errorf("Expected error %v, got nil", tc.err)
			} else if err != nil && tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Errorf("Expected error %v, got %v", tc.err, err)
				}
			} else if result.String() != tc.output {
				t.Errorf("Expected %v, got %v", tc.output, result.String())
			}
		})
	}
}

func TestPureWindowsPath_ValidatePath(t *testing.T) {
	testcases := []struct {
		input  string
		output error
	}{
		{"c:/foo/bar.txt", nil},
		{"c:foo/bar.txt", nil},
		{"//server/share/file.txt", nil},
		{"/foo/bar.txt", nil},
		{"foo/bar.txt", nil},
		{"c:/", nil},
		{"/", nil},
		{"c:", nil},
		{"//server/share/", nil},

		{"c:/foo/invalid:name.txt", nt.ErrInvalidName}, // 包含非法字符
		{"c:/foo/bar/na\rme", nt.ErrInvalidName},       // 包含回车符
		{"c:/foo/bar/na\rme\t", nt.ErrInvalidName},
		{"c:/foo/bar/con", nt.ErrInvalidName},
	}

	for _, tc := range testcases {
		t.Run(tc.input, func(t *testing.T) {
			path := NewPureWindowsPath(tc.input)
			err := path.Validate()
			if err != nil && tc.output == nil {
				t.Errorf("Expected no error, got %v", err)
			} else if err == nil && tc.output != nil {
				t.Errorf("Expected error %v, got nil", tc.output)
			} else if err != nil && tc.output != nil {
				if !errors.Is(err, tc.output) {
					t.Errorf("Expected error %v, got %v", tc.output, err)
				}
			}
		})
	}
}

func TestPureWindowsPath_ToValid(t *testing.T) {
	testcases := []struct {
		input  string
		output string
	}{
		{"c:/foo/bar\rtxt", `c:\foo\bar txt`},
		{"c:/foo/bar\ntxt", `c:\foo\bar txt`},
		{"c:/foo/bar\t", `c:\foo\bar`},
		{"c:/foo/bar:txt", `c:\foo\bar_txt`},
		{"c:/foo/bar?txt", `c:\foo\bar_txt`},
		{"c:/foo/bar\"txt", `c:\foo\bar_txt`},
	}

	for _, tc := range testcases {
		t.Run(tc.input, func(t *testing.T) {
			path := NewPureWindowsPath(tc.input)
			result := path.ToValid()
			if result.String() != tc.output {
				t.Errorf("Expected %v, got %v", tc.output, result)
			}
		})
	}
}
