package ntpath

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/viocha/go-pathlib/common"
)

var (
	ErrNoName        = errors.New("no name provided")
	ErrNotRelative   = errors.New("path is not relative to the other path")
	ErrInvalidName   = errors.New("invalid name provided")
	ErrInvalidAnchor = errors.New("invalid anchor provided")
	ErrInvalidSuffix = errors.New("invalid suffix provided")
)

var ReservedNames = []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8",
	"COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
var InvalidChars = `<>:/\|?*"`

// Parts 返回路径的各个部分，包括anchor部分，以及剩余的目录和名称部分。
// 会假定anchor的名称合法，然后使用路径分隔符分割得到目录和名称部分
func Parts(path string) []string {
	if path == "." {
		return nil
	}
	path = Clean(path) // 先清理路径，确保是标准格式

	// UNC路径
	regUNC := regexp.MustCompile(`^\\\\[^\\]+\\[^\\]+\\`)
	if regUNC.MatchString(path) {
		// 提取UNC路径的anchor部分
		anchor := regUNC.FindString(path)
		if len(path) == len(anchor) { // 如果只有根路径
			return []string{anchor}
		}
		// 去掉根部分后进行分割
		parts := strings.Split(path[len(anchor):], `\`)
		return append([]string{anchor}, parts...)
	}

	// 带盘符路径
	regDrive := regexp.MustCompile(`(?i)^[A-Z]:\\?`)
	if regDrive.MatchString(path) {
		// 提取盘符和冒号，以及可选的反斜杠
		anchor := regDrive.FindString(path)
		var parts []string
		if len(path) == len(anchor) { // 如果只有盘符
			return []string{anchor}
		}
		parts = strings.Split(path[len(anchor):], `\`)
		return append([]string{anchor}, parts...)
	}

	// 以反斜杠开头的路径
	if strings.HasPrefix(path, `\`) {
		if len(path) == 1 {
			return []string{`\`} // 只有根路径
		}
		// 去掉开头的反斜杠，并进行分割
		parts := strings.Split(path[1:], `\`)
		return append([]string{`\`}, parts...)
	}

	// 直接分割
	return strings.Split(path, `\`)
}

func Drive(path string) string {
	parts := Parts(path)
	if len(parts) == 0 {
		return ""
	}
	first := parts[0]
	if strings.HasPrefix(first, `\\`) { // UNC路径
		return strings.TrimSuffix(first, `\`)
	}
	if len(first) >= 2 && first[1] == ':' { // 驱动器盘符
		return strings.TrimSuffix(first, `\`)
	}
	return ""
}

func Root(path string) string {
	parts := Parts(path)
	if len(parts) == 0 {
		return ""
	}

	first := parts[0]
	if strings.HasPrefix(first, `\\`) { // UNC路径
		return `\`
	} else if len(first) >= 2 && first[1] == ':' && strings.HasSuffix(first, `\`) { // 带盘符且以\结尾
		return `\`
	} else if first == `\` { // 以\开头的路径
		return `\`
	}

	return ""
}

func Anchor(path string) string {
	return Drive(path) + Root(path) // parts[0]就是anchor，也等于drive + root
}

// 检查Windows路径名是否合法，不支持空名称
func ValidateName(name string) error {
	if len(name) == 0 {
		return common.WrapMsg(ErrNoName, "name cannot be empty")
	}
	if len(name) > 255 {
		return common.WrapMsg(ErrInvalidName, "name %q exceeds maximum length of 255 characters", name)
	}

	// 检查名称是否以点结尾
	if len(name) > 0 && name[len(name)-1] == '.' {
		return common.WrapMsg(ErrInvalidName, "name %q cannot end with a dot", name)
	}
	// 检查名称是否以空白字符结尾
	if regexp.MustCompile(`\s$`).MatchString(name) {
		return common.WrapMsg(ErrInvalidName, "name %q cannot end with a whitespace character", name)
	}
	// 是否包含换行符
	if strings.ContainsAny(name, "\n\r") {
		return common.WrapMsg(ErrInvalidName, "name %q cannot contain newline characters", name)
	}

	// 检查是否包含无效字符
	for _, char := range InvalidChars {
		if strings.Contains(name, string(char)) {
			return common.WrapMsg(ErrInvalidName, "name %q contains invalid character %q", name, string(char))
		}
	}

	// Windows保留名称检查
	for _, reservedName := range ReservedNames {
		if strings.EqualFold(name, reservedName) {
			return common.WrapMsg(ErrInvalidName, "name %q is a reserved name in Windows", name)
		}
	}
	return nil
}

func ValidateAnchor(anchor string) error {
	// 替换路径中的斜杠
	anchor = strings.ReplaceAll(anchor, `/`, `\`)

	if strings.HasPrefix(anchor, `\\`) { // UNC路径
		if !regexp.MustCompile(`^\\\\[^\\]+\\[^\\]+\\$`).MatchString(anchor) {
			return common.WrapMsg(ErrInvalidAnchor, "invalid UNC anchor %q", anchor)
		}
	} else if strings.Contains(anchor, ":") { // 驱动器盘符
		if !regexp.MustCompile(`(?i)^[A-Z]:\\?$`).MatchString(anchor) {
			return common.WrapMsg(ErrInvalidAnchor, "invalid drive anchor %q", anchor)
		}
	} else if anchor != `\` && anchor != "" { // 其他情况
		return common.WrapMsg(ErrInvalidAnchor, "invalid anchor %q", anchor)
	}
	return nil
}

// 检查路径是否合法，包括anchor和每个部分的名称
func ValidatePath(path string) error {
	path = Clean(path) // 先清理路径，确保是标准格式
	anchor, parts := Anchor(path), Parts(path)

	if anchor != "" {
		if err := ValidateAnchor(anchor); err != nil {
			return err
		}
		parts = parts[1:] // 去掉anchor部分
	}

	// 检查每个部分的名称是否合法
	for _, part := range parts {
		if err := ValidateName(part); err != nil {
			return err
		}
	}
	return nil
}

// 让名称合法
func ToValidName(name string) string {
	if len(name) == 0 {
		return "_"
	}
	if len(name) > 255 {
		name = name[:255] // 截断到255个字符
	}

	// 替换无效字符
	regInvalidChars := fmt.Sprintf("[%s]", regexp.QuoteMeta(InvalidChars))
	name = regexp.MustCompile(regInvalidChars).ReplaceAllString(name, "_")
	// 去掉末尾的点和空白字符
	name = regexp.MustCompile(`[\s.]+$`).ReplaceAllString(name, "")
	// 去掉换行符
	name = strings.NewReplacer("\n", " ", "\r", " ").Replace(name)
	// 如果是保留名称，则添加下划线
	for _, reserved := range ReservedNames {
		if strings.EqualFold(name, reserved) {
			name += "_"
			break
		}
	}
	if len(name) == 0 { // 可能名称被清理之后变成空字符串
		return "_"
	}
	return name
}

// 规范化路径，并不会确保名称合法
func Clean(path string) string {
	path = filepath.Clean(path)

	// 替换为Windows反斜杠
	path = strings.ReplaceAll(path, "/", `\`)

	// 确保UNC根路径以斜杠结尾
	if strings.HasPrefix(path, `\\`) {
		if strings.Count(path, `\`) == 3 && !strings.HasSuffix(path, `\`) { // 只有3个斜杠，则确保末尾有一个斜杠
			path += `\`
		}
	}
	// 移除盘符相对路径末尾的.号，防止出现 c:.
	if len(path) >= 2 && path[1] == ':' && strings.HasSuffix(path, `.`) {
		path = strings.TrimSuffix(path, `.`)
	}
	return path
}

func Match(pattern, path string) bool {
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	// 初始化记忆化缓存
	memo := make(map[[2]int]bool)
	var match func(int, int) bool
	match = func(patternIdx, pathIdx int) (res bool) {
		// 生成缓存键
		key := [2]int{patternIdx, pathIdx}
		// 检查缓存
		if v, ok := memo[key]; ok {
			return v
		}
		// 将结果存入缓存（通过defer确保所有return前执行）
		defer func() { memo[key] = res }()

		// 原有逻辑
		if patternIdx == len(patternParts) {
			return pathIdx == len(pathParts)
		}
		if patternParts[patternIdx] == "**" {
			for currentPathPos := pathIdx; currentPathPos <= len(pathParts); currentPathPos++ {
				if match(patternIdx+1, currentPathPos) {
					return true
				}
			}
			return false
		}
		if pathIdx >= len(pathParts) {
			return false
		}
		currentMatch, _ := filepath.Match(patternParts[patternIdx], pathParts[pathIdx])
		if !currentMatch {
			return false
		}
		return match(patternIdx+1, pathIdx+1)
	}
	return match(0, 0)
}
