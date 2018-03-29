# AssetFS

AssetFS is a golang package and defined AssetFS Interface that abstracts the access to files.

It has a default implementation that based on FileSystem, which could be used in development, load required files directly from disk.

If you want to compile all required files into a binary and load files from from binary, you could refer our [bindatafs](http://github.com/qor/bindatafs)

# Usage

```go
import "github.com/qor/assetfs"

func main() {
	// Default implemention based on filesystem, you could overwrite with other implemention, for example bindatafs will do this.
	assetfs := assetfs.AssetFS

	// Register path to AssetFS
	assetfs.RegisterPath("/web/app/views")

	// Prepend path to AssetFS
	assetfs.PrependPath("/web/app/views")

	// Get file's content with name from path `/web/app/views`
	assetfs.Asset("filename.tmpl")

	// List matched files from assetfs
	assetfs.Glob("*.tmpl")

	// NameSpace return namespaced filesystem
	namespacedFS := assetfs.NameSpace("asset")
	namespacedFS.RegisterPath("/web/app/myspecialviews")
	namespacedFS.PrependPath("/web/app/myspecialviews")
	// Will lookup file with name "filename.tmpl" from path `/web/app/myspecialviews` but not `/web/app/views`
	namespacedFS.Asset("filename.tmpl")
	namespacedFS.Glob("*.tmpl")
}
```
