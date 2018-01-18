package assetfs

import (
	"fmt"
	"runtime/debug"
)

// Interface assetfs interface
type Interface interface {
	PrependPath(path string) error
	RegisterPath(path string) error
	Asset(name string) ([]byte, error)
	Glob(pattern string) (matches []string, err error)
	Compile() error

	NameSpace(nameSpace string) Interface
}

// AssetFS default assetfs
var assetFS Interface = &AssetFileSystem{}
var used bool

// AssetFS get AssetFS
func AssetFS() Interface {
	used = true
	return assetFS
}

// SetAssetFS set assetfs
func SetAssetFS(fs Interface) {
	if used {
		fmt.Println("WARNING: AssetFS is used before overwrite it!")
		debug.PrintStack()
	}

	assetFS = fs
}
