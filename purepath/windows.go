package purepath

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/viocha/go-pathlib/common"
	nt "github.com/viocha/go-pathlib/purepath/ntpath"
)

// PureWindowsPath Windows文件系统路径的纯路径实现
type PureWindowsPath struct {
	path string
}

var _ IPurePath = (*PureWindowsPath)(nil) // 确保实现了IPurePath接口

// 创建新的Windows纯路径对象
func NewPureWindowsPath(segments ...string) *PureWindowsPath {
	if len(segments) == 0 {
		return &PureWindowsPath{path: "."}
	}
	if len(segments) == 1 && (segments[0] == "" || segments[0] == ".") {
		return &PureWindowsPath{path: "."}
	}

	path := nt.Clean(segments[0])
	for _, seg := range segments[1:] {
		seg = nt.Clean(seg)               // 规范化每个路径段
		if strings.HasPrefix(seg, `\\`) { // UNC绝对路径，覆盖前面的路径
			path = seg
		} else if regexp.MustCompile(`(?i)^[A-Z]:`).MatchString(seg) { // 以盘符开头的路径
			drive := seg[:2]                     // 提取盘符
			if len(seg) >= 3 && seg[2] == '\\' { // 有根路径\，为绝对路径
				path = seg // 直接覆盖
			} else { // 为相对路径，需要考虑是否和前面的盘符相同
				if strings.HasPrefix(path, drive) { // 盘符相同，则附加
					path = filepath.Join(path, seg[2:]) // 去掉盘符后的路径
				} else { // 盘符不同，则覆盖
					path = seg // 直接覆盖
				}
			}
		} else if strings.HasPrefix(seg, `\`) { // 以\开头的绝对路径
			// 提取前面的盘符
			reg := regexp.MustCompile(`(?i)^[A-Z]:\\`)
			if reg.MatchString(path) {
				path = path[:2] + seg // 盘符+绝对路径
			} else {
				path = seg // 没有盘符，则直接覆盖
			}
		} else { // 常规的相对路径
			path = filepath.Join(path, seg) // 注意 .. 也会被解析，和python的不同
		}
	}

	path = nt.Clean(path) // 规范化路径
	return &PureWindowsPath{path: path}
}

// 返回路径的字符串表示
func (p *PureWindowsPath) String() string {
	return p.path
}

// 返回路径的所有组件
func (p *PureWindowsPath) Parts() []string {
	return nt.Parts(p.path)
}

// 如果存在则返回盘符 c: 或UNC共享名 \\server\share，其他情况返回空字符串
func (p *PureWindowsPath) Drive() string {
	return nt.Drive(p.path)
}

// 返回根路径标识符，UNC的根为 `\`，带盘符的绝对路径的根为 `\`，以\开头的路径根为 `\`，其他情况返回空字符串
func (p *PureWindowsPath) Root() string {
	return nt.Root(p.path)
}

// 返回驱动器和根的联合
func (p *PureWindowsPath) Anchor() string {
	return nt.Anchor(p.path)
}

// 返回此路径的逻辑父路径，如果没有父路径，则返回当前路径
func (p *PureWindowsPath) Parent() IPurePath {
	anchor, parts := p.Anchor(), p.Parts()
	if len(parts) == 0 {
		return NewPureWindowsPath(".")
	}
	if len(parts) == 1 {
		if anchor != "" {
			return NewPureWindowsPath(parts[0]) // 如果等于anchor，保留anchor不变
		} else {
			return NewPureWindowsPath(".") // 没有anchor的情况，返回当前路径.
		}
	}
	// 移除最后一个部分
	newParts := parts[:len(parts)-1]
	var newPath string
	if anchor == "" { // 常规的相对路径
		newPath = strings.Join(newParts, `\`) // 相对路径，直接连接
	} else { // 有anchor的情况，anchor中包含了\分隔符
		newPath = parts[0] + strings.Join(newParts[1:], `\`) // parts[0]已经携带根路径，无需再和\连接
	}
	return NewPureWindowsPath(newPath)
}

// 返回此路径的所有逻辑祖先路径，可能为空数组
func (p *PureWindowsPath) Parents() []IPurePath {
	if p.String() == "." || p.String() == p.Anchor() { // 如果是'.'，或者
		return nil
	}
	result := []IPurePath{}
	for cur := p.Parent(); ; cur = cur.Parent() {
		result = append(result, cur)
		if cur.String() == "." || cur.String() == cur.Anchor() {
			break // 到达根路径或当前路径时停止
		}
	}
	return result
}

// 返回最后一个路径组件
func (p *PureWindowsPath) Name() string {
	parts := p.Parts()
	if len(parts) == 0 {
		return ""
	}
	if p.Anchor() != "" {
		if len(parts) == 1 {
			return ""
		} else {
			return parts[len(parts)-1] // 如果除了根有其他部分，返回最后一个部分
		}
	}
	return parts[len(parts)-1] // 相对路径，返回最后一个部分
}

// 返回文件扩展名
func (p *PureWindowsPath) Suffix() string {
	suffixes := p.Suffixes()
	if len(suffixes) == 0 {
		return ""
	}
	return suffixes[len(suffixes)-1] // 返回最后一个扩展名
}

// 返回所有文件扩展名
func (p *PureWindowsPath) Suffixes() []string {
	name := p.Name()
	if name == "" {
		return nil
	}

	// 分割文件名和扩展名
	nameParts := strings.Split(name, ".")
	if len(nameParts) <= 1 {
		return nil // 没有扩展名
	}

	// 返回所有扩展名，去掉第一个部分
	suffixes := nameParts[1:]
	for i, suffix := range suffixes {
		suffixes[i] = "." + suffix // 添加点前缀
	}
	return suffixes
}

// 返回去除扩展名的文件名
func (p *PureWindowsPath) Stem() string {
	name := p.Name()
	suffix := p.Suffix()
	return strings.TrimSuffix(name, suffix) // 去掉最后一个扩展名
}

// 设置新的anchor，返回新的路径对象，anchor支持的格式和Anchor方法返回的一样，可以是 \\server\share\ c: c:\ \ ""
func (p *PureWindowsPath) WithAnchor(anchor string) (IPurePath, error) {
	if err := nt.ValidateAnchor(anchor); err != nil {
		return nil, err
	}

	parts := p.Parts()
	curAnchor := p.Anchor()
	if curAnchor == "" {
		parts = slices.Insert(parts, 0, anchor) // 如果当前路径没有anchor，则直接插入
	} else {
		parts[0] = anchor // 如果当前路径有anchor，则替换第一个部分
	}

	return NewPureWindowsPath(parts...), nil
}

// 返回修改name后的新路径
func (p *PureWindowsPath) WithName(name string) (IPurePath, error) {
	curName := p.Name()
	if curName == "" {
		return nil, common.WrapMsg(nt.ErrNoName, "cannot set name %q on path without name", name)
	}
	if err := nt.ValidateName(name); err != nil {
		return nil, err
	}
	return NewPureWindowsPath(p.path, "..", name), nil
}

// 返回修改父路径后的新路径，parent必须是一个有效的IPurePath对象
func (p *PureWindowsPath) WithParent(parent IPurePath) (IPurePath, error) {
	name := p.Name()
	if name == "" {
		return nil, common.WrapMsg(nt.ErrNoName, "cannot set parent on path without name")
	}
	return parent.Join(name), nil
}

// 返回修改stem后的新路径
func (p *PureWindowsPath) WithStem(stem string) (IPurePath, error) {
	name := p.Name()
	if name == "" {
		return nil, common.WrapMsg(nt.ErrNoName, "cannot set stem %q on empty name", stem)
	}
	suffix := p.Suffix()
	newName := stem + suffix // 保留原有扩展名

	if err := nt.ValidateName(newName); err != nil {
		return nil, err
	}
	return NewPureWindowsPath(p.path, "..", newName), nil
}

// 返回修改suffix后的新路径，必须存在文件名，否则返回ErrNoName错误，suffix必须以点开头，否则返回ErrInvalidSuffix错误
func (p *PureWindowsPath) WithSuffix(suffix string) (IPurePath, error) {
	name := p.Name()
	if name == "" {
		return nil, common.WrapMsg(nt.ErrNoName, "cannot set suffix %q on empty name", suffix)
	}
	if !strings.HasPrefix(suffix, ".") {
		return nil, common.WrapMsg(nt.ErrInvalidSuffix, "suffix %q must start with a dot", suffix)
	}

	stem, suffixes := p.Stem(), p.Suffixes()
	firstDotPart := strings.SplitN(stem, ".", 2)[0] // 获取第一个点前的部分
	if len(suffixes) == 0 {
		suffixes = []string{suffix} // 如果没有扩展名，则直接添加
	} else {
		suffixes[len(suffixes)-1] = suffix // 替换最后一个扩展名
	}
	newName := firstDotPart + strings.Join(suffixes, "")

	if err := nt.ValidateName(newName); err != nil {
		return nil, err
	}
	return NewPureWindowsPath(p.path, "..", newName), nil
}

// 返回使用正斜杠的路径字符串
func (p *PureWindowsPath) ToPosix() string {
	return strings.ReplaceAll(p.path, `\`, `/`)
}

// 返回此路径是否为绝对路径
func (p *PureWindowsPath) IsAbs() bool {
	return p.Drive() != "" && p.Root() != "" // 有盘符和根路径，才是绝对路径
}

// 返回此路径是否相对于other路径，walkUp参数表示是否允许向上遍历
func (p *PureWindowsPath) IsRelTo(other IPurePath, walkUp ...bool) bool {
	isWalkUp := common.ParseOptional(walkUp, true) // 默认允许向上遍历，和python不同
	// 忽略大小写比较
	pPath := strings.ToLower(p.String())
	otherPath := strings.ToLower(other.String())
	pAnchor := strings.ToLower(p.Anchor())
	otherAnchor := strings.ToLower(other.Anchor())
	if pAnchor != "" {
		if isWalkUp {
			return pAnchor == otherAnchor // 允许向上遍历到根路径
		}
		return strings.HasPrefix(pPath, otherPath) && pAnchor == otherAnchor
	} else {
		if otherAnchor != "" {
			return false
		}
		return otherPath == "." || strings.HasPrefix(pPath, otherPath) // 无根路径，必须是前缀
	}
}

func (p *PureWindowsPath) Validate() error {
	return nt.ValidatePath(p.path)
}

// 让所有名称都变得合法，不会改变anchor部分。如果anchor部分不合法，则结果路径会偏差很大
func (p *PureWindowsPath) ToValid() IPurePath {
	anchor, parts := p.Anchor(), p.Parts()
	if anchor != "" {
		parts = parts[1:] // 去掉anchor部分
	}
	for i, part := range parts {
		parts[i] = nt.ToValidName(part) // 让每个部分的名称合法
	}
	parts = slices.Insert(parts, 0, anchor) // 将anchor部分重新插入到最前面
	return NewPureWindowsPath(parts...)     // 重新组合路径
}

// 将路径与给定的路径段组合
func (p *PureWindowsPath) Join(segments ...string) IPurePath {
	segments = slices.Insert(segments, 0, p.path) // 将当前路径作为第一个元素
	return NewPureWindowsPath(segments...)
}

// 将此路径与pattern完全匹配
func (p *PureWindowsPath) FullMatch(pattern string, caseSensitive ...bool) bool {
	cs := common.ParseOptional(caseSensitive, false) // 默认不区分大小写

	path := p.String()
	if !cs { // 如果不区分大小写，则转换为小写
		path = strings.ToLower(path)
		pattern = strings.ToLower(pattern)
	}

	// 规范化模式串格式
	pattern = nt.Clean(pattern)
	// 支持**语法的匹配方式
	return nt.Match(pattern, path)
}

// 将此路径与pattern匹配，如果pattern是相对路径，则从右侧开始匹配
func (p *PureWindowsPath) Match(pattern string, caseSensitive ...bool) bool {
	cs := common.ParseOptional(caseSensitive, false) // 默认不区分大小写

	path := p.String()
	if !cs { // 如果不区分大小写，则转换为小写
		path = strings.ToLower(path)
		pattern = strings.ToLower(pattern)
	}

	patternPath := NewPureWindowsPath(pattern)
	if patternPath.IsAbs() || patternPath.Drive() != "" { // 如果模式是绝对路径或包含盘符
		return p.FullMatch(pattern, caseSensitive...) // 使用全匹配
	}
	// 相对路径从右侧开始匹配
	parts := p.Parts()
	patternParts := patternPath.Parts()
	if len(parts) < len(patternParts) {
		return false // 如果当前路径部分少于模式部分，无法匹配
	}
	for i := len(patternParts) - 1; i >= 0; i-- {
		matched, _ := filepath.Match(patternParts[i], parts[len(parts)-1-i])
		if !matched {
			return false // 如果任意部分不匹配，则返回false
		}
	}
	return true // 如果所有模式部分都匹配，则返回true
}

// 计算此路径相对于other的版本
func (p *PureWindowsPath) RelTo(other IPurePath, walkUp ...bool) (IPurePath, error) {
	isWalkUp := common.ParseOptional(walkUp, true) // 默认允许向上遍历
	if !p.IsRelTo(other, isWalkUp) {
		return nil, common.WrapMsg(nt.ErrNotRelative, "path %q is not relative to %q", p, other)
	}
	parts, otherParts := p.Parts(), other.Parts()
	// 找到公共前缀长度
	commonLength := 0
	for i := 0; i < len(otherParts); i++ {
		if !strings.EqualFold(parts[i], otherParts[i]) {
			break
		}
		commonLength++
	}
	if !isWalkUp {
		return NewPureWindowsPath(parts[commonLength:]...), nil
	}
	// 计算向上遍历的次数
	upLevels := len(otherParts) - commonLength
	upSegments := make([]string, upLevels)
	for i := 0; i < upLevels; i++ {
		upSegments[i] = ".." // 向上遍历的部分
	}
	newParts := append(upSegments, parts[commonLength:]...) // 合并向上遍历的部分和剩余部分
	return NewPureWindowsPath(newParts...), nil
}

// 基于目标文件的相对路径，会先获取目标文件的父路径，然后计算相对路径
func (p *PureWindowsPath) RelToFile(other IPurePath, walkUp ...bool) (IPurePath, error) {
	return p.RelTo(other.Parent(), walkUp...)
}

func (p *PureWindowsPath) MustWithAnchor(anchor string) IPurePath {
	path, err := p.WithAnchor(anchor)
	if err != nil {
		panic(err)
	}
	return path
}

func (p *PureWindowsPath) MustWithName(name string) IPurePath {
	path, err := p.WithName(name)
	if err != nil {
		panic(err)
	}
	return path
}

func (p *PureWindowsPath) MustWithStem(stem string) IPurePath {
	path, err := p.WithStem(stem)
	if err != nil {
		panic(err)
	}
	return path
}

func (p *PureWindowsPath) MustWithSuffix(suffix string) IPurePath {
	path, err := p.WithSuffix(suffix)
	if err != nil {
		panic(err)
	}
	return path
}

func (p *PureWindowsPath) MustWithParent(parent IPurePath) IPurePath {
	path, err := p.WithParent(parent)
	if err != nil {
		panic(err)
	}
	return path
}

func (p *PureWindowsPath) MustRelTo(other IPurePath, walkUp ...bool) IPurePath {
	path, err := p.RelTo(other, walkUp...)
	if err != nil {
		panic(err)
	}
	return path
}

func (p *PureWindowsPath) MustRelToFile(other IPurePath, walkUp ...bool) IPurePath {
	path, err := p.RelToFile(other, walkUp...)
	if err != nil {
		panic(err)
	}
	return path
}
