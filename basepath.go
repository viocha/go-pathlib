package path

import (
	"github.com/viocha/go-pathlib/purepath"
)

// 拥有 IPurePath 的所有方法，但是将使用 IPurePath 的方法转换为使用 IPath 的方法
type IBasePath interface {
	purepath.IBasePurePath

	Parents() []IPath
	Parent() IPath
	Join(segments ...string) IPath

	WithAnchor(anchor string) (IPath, error)
	WithName(name string) (IPath, error)
	WithParent(parent IPath) (IPath, error)
	WithStem(stem string) (IPath, error)
	WithSuffix(suffix string) (IPath, error)

	ToValid() IPath

	IsRelTo(other IPath, walkUp ...bool) bool
	RelTo(other IPath, walkUp ...bool) (IPath, error)
	RelToFile(other IPath, walkUp ...bool) (IPath, error)

	MustWithAnchor(anchor string) IPath
	MustWithName(name string) IPath
	MustWithParent(parent IPath) IPath
	MustWithStem(stem string) IPath
	MustWithSuffix(suffix string) IPath
	MustRelTo(other IPath, walkUp ...bool) IPath
	MustRelToFile(other IPath, walkUp ...bool) IPath
}

// 将使用 IPurePath 的方法转换为使用 IPath 的方法
type BasePath struct {
	purepath.IPurePath
}

// 确保实现了 IBasePath 接口
var _ IBasePath = (*BasePath)(nil)

func NewBasePath(segments ...string) *BasePath {
	return &BasePath{
		IPurePath: purepath.New(segments...),
	}
}

func (p *BasePath) Parents() []IPath {
	parents := p.IPurePath.Parents()
	result := make([]IPath, len(parents))
	for i, parent := range parents {
		result[i] = FromPurePath(parent)
	}
	return result
}

func (p *BasePath) Parent() IPath {
	parent := p.IPurePath.Parent()
	return FromPurePath(parent)
}

func (p *BasePath) Join(segments ...string) IPath {
	joined := p.IPurePath.Join(segments...)
	return FromPurePath(joined)
}

func (p *BasePath) WithAnchor(anchor string) (IPath, error) {
	newPath, err := p.IPurePath.WithAnchor(anchor)
	return FromPurePath(newPath), err
}

func (p *BasePath) WithName(name string) (IPath, error) {
	newPath, err := p.IPurePath.WithName(name)
	return FromPurePath(newPath), err
}

func (p *BasePath) WithParent(parent IPath) (IPath, error) {
	newPath, err := p.IPurePath.WithParent(parent.ToPurePath())
	return FromPurePath(newPath), err
}

func (p *BasePath) WithStem(stem string) (IPath, error) {
	newPath, err := p.IPurePath.WithStem(stem)
	return FromPurePath(newPath), err
}

func (p *BasePath) WithSuffix(suffix string) (IPath, error) {
	newPath, err := p.IPurePath.WithSuffix(suffix)
	return FromPurePath(newPath), err
}

func (p *BasePath) ToValid() IPath {
	validPath := p.IPurePath.ToValid()
	return FromPurePath(validPath)
}

func (p *BasePath) IsRelTo(other IPath, walkUp ...bool) bool {
	return p.IPurePath.IsRelTo(other.ToPurePath(), walkUp...)
}

func (p *BasePath) RelTo(other IPath, walkUp ...bool) (IPath, error) {
	relativePath, err := p.IPurePath.RelTo(other.ToPurePath(), walkUp...)
	return FromPurePath(relativePath), err
}

func (p *BasePath) RelToFile(other IPath, walkUp ...bool) (IPath, error) {
	relativePath, err := p.IPurePath.RelToFile(other.ToPurePath(), walkUp...)
	return FromPurePath(relativePath), err
}

func (p *BasePath) MustWithAnchor(anchor string) IPath {
	return FromPurePath(p.IPurePath.MustWithAnchor(anchor))
}

func (p *BasePath) MustWithName(name string) IPath {
	return FromPurePath(p.IPurePath.MustWithName(name))
}

func (p *BasePath) MustWithParent(parent IPath) IPath {
	return FromPurePath(p.IPurePath.MustWithParent(parent.ToPurePath()))
}

func (p *BasePath) MustWithStem(stem string) IPath {
	return FromPurePath(p.IPurePath.MustWithStem(stem))
}

func (p *BasePath) MustWithSuffix(suffix string) IPath {
	return FromPurePath(p.IPurePath.MustWithSuffix(suffix))
}

func (p *BasePath) MustRelTo(other IPath, walkUp ...bool) IPath {
	return FromPurePath(p.IPurePath.MustRelTo(other.ToPurePath(), walkUp...))
}

func (p *BasePath) MustRelToFile(other IPath, walkUp ...bool) IPath {
	return FromPurePath(p.IPurePath.MustRelToFile(other.ToPurePath(), walkUp...))
}
