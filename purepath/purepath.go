package purepath

import (
	"runtime"
)

// IBasePurePath 包含所有可以直接继承的方法，不依赖于特定的接口和类型
type IBasePurePath interface {
	String() string
	ToPosix() string
	Parts() []string

	Root() string
	Drive() string
	Anchor() string
	Name() string
	Stem() string
	Suffix() string
	Suffixes() []string

	Validate() error

	IsAbs() bool

	FullMatch(pattern string, caseSensitive ...bool) bool
	Match(pattern string, caseSensitive ...bool) bool
}

// IPurePath 纯路径接口，提供不实际访问文件系统的路径处理操作
// 创建实例时不会检查路径合法性，但是修改路径的组件时会进行合法性检查，并返回错误
// 可以通过 Validate() 方法手动检查整个路径合法性
// ToValid() 方法会修复所有不合法的组件名称，除了anchor部分
type IPurePath interface {
	IBasePurePath

	Parents() []IPurePath
	Parent() IPurePath
	Join(segments ...string) IPurePath

	WithAnchor(anchor string) (IPurePath, error)
	WithParent(parent IPurePath) (IPurePath, error)
	WithName(name string) (IPurePath, error)
	WithStem(stem string) (IPurePath, error)
	WithSuffix(suffix string) (IPurePath, error)

	ToValid() IPurePath
	IsRelTo(other IPurePath, walkUp ...bool) bool
	RelTo(other IPurePath, walkUp ...bool) (IPurePath, error)
	RelToFile(other IPurePath, walkUp ...bool) (IPurePath, error)

	// 出现错误直接 panic 的版本
	MustWithAnchor(anchor string) IPurePath
	MustWithName(name string) IPurePath
	MustWithParent(parent IPurePath) IPurePath
	MustWithStem(stem string) IPurePath
	MustWithSuffix(suffix string) IPurePath
	MustRelTo(other IPurePath, walkUp ...bool) IPurePath
	MustRelToFile(other IPurePath, walkUp ...bool) IPurePath
}

func New(segments ...string) IPurePath {
	if runtime.GOOS == "windows" {
		return NewPureWindowsPath(segments...)
	}
	panic("purepath package is not implemented for this OS")
}
