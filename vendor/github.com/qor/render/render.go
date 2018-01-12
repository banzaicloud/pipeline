// Package render support to render templates by your control.
package render

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/qor/assetfs"
	"github.com/qor/qor/utils"
)

// DefaultLayout default layout name
const DefaultLayout = "application"

// DefaultViewPath default view path
const DefaultViewPath = "app/views"

// Config render config
type Config struct {
	ViewPaths       []string
	DefaultLayout   string
	FuncMapMaker    func(render *Render, request *http.Request, writer http.ResponseWriter) template.FuncMap
	AssetFileSystem assetfs.Interface
}

// Render the render struct.
type Render struct {
	*Config

	funcMaps template.FuncMap
}

// New initalize the render struct.
func New(config *Config, viewPaths ...string) *Render {
	if config == nil {
		config = &Config{}
	}

	if config.DefaultLayout == "" {
		config.DefaultLayout = DefaultLayout
	}

	if config.AssetFileSystem == nil {
		config.AssetFileSystem = assetfs.AssetFS().NameSpace("views")
	}

	config.ViewPaths = append(append(config.ViewPaths, viewPaths...), DefaultViewPath)

	render := &Render{funcMaps: map[string]interface{}{}, Config: config}

	for _, viewPath := range config.ViewPaths {
		render.RegisterViewPath(viewPath)
	}

	return render
}

// RegisterViewPath register view path
func (render *Render) RegisterViewPath(paths ...string) {
	for _, pth := range paths {
		if filepath.IsAbs(pth) {
			render.ViewPaths = append(render.ViewPaths, pth)
			render.AssetFileSystem.RegisterPath(pth)
		} else {
			if absPath, err := filepath.Abs(pth); err == nil && isExistingDir(absPath) {
				render.ViewPaths = append(render.ViewPaths, absPath)
				render.AssetFileSystem.RegisterPath(absPath)
			} else if isExistingDir(filepath.Join(utils.AppRoot, "vendor", pth)) {
				render.AssetFileSystem.RegisterPath(filepath.Join(utils.AppRoot, "vendor", pth))
			} else {
				for _, gopath := range utils.GOPATH() {
					if p := filepath.Join(gopath, "src", pth); isExistingDir(p) {
						render.ViewPaths = append(render.ViewPaths, p)
						render.AssetFileSystem.RegisterPath(p)
					}
				}
			}
		}
	}
}

// PrependViewPath prepend view path
func (render *Render) PrependViewPath(paths ...string) {
	for _, pth := range paths {
		if filepath.IsAbs(pth) {
			render.ViewPaths = append([]string{pth}, render.ViewPaths...)
			render.AssetFileSystem.PrependPath(pth)
		} else {
			if absPath, err := filepath.Abs(pth); err == nil && isExistingDir(absPath) {
				render.ViewPaths = append([]string{absPath}, render.ViewPaths...)
				render.AssetFileSystem.PrependPath(absPath)
			} else if isExistingDir(filepath.Join(utils.AppRoot, "vendor", pth)) {
				render.AssetFileSystem.PrependPath(filepath.Join(utils.AppRoot, "vendor", pth))
			} else {
				for _, gopath := range utils.GOPATH() {
					if p := filepath.Join(gopath, "src", pth); isExistingDir(p) {
						render.ViewPaths = append([]string{p}, render.ViewPaths...)
						render.AssetFileSystem.PrependPath(p)
					}
				}
			}
		}
	}
}

// SetAssetFS set asset fs for render
func (render *Render) SetAssetFS(assetFS assetfs.Interface) {
	for _, viewPath := range render.ViewPaths {
		assetFS.RegisterPath(viewPath)
	}

	render.AssetFileSystem = assetFS
}

// Layout set layout for template.
func (render *Render) Layout(name string) *Template {
	return &Template{render: render, layout: name}
}

// Funcs set helper functions for template with default "application" layout.
func (render *Render) Funcs(funcMap template.FuncMap) *Template {
	tmpl := &Template{render: render, usingDefaultLayout: true}
	return tmpl.Funcs(funcMap)
}

// Execute render template with default "application" layout.
func (render *Render) Execute(name string, context interface{}, request *http.Request, writer http.ResponseWriter) error {
	tmpl := &Template{render: render, usingDefaultLayout: true}
	return tmpl.Execute(name, context, request, writer)
}

// RegisterFuncMap register FuncMap for render.
func (render *Render) RegisterFuncMap(name string, fc interface{}) {
	if render.funcMaps == nil {
		render.funcMaps = template.FuncMap{}
	}
	render.funcMaps[name] = fc
}

// Asset get content from AssetFS by name
func (render *Render) Asset(name string) ([]byte, error) {
	return render.AssetFileSystem.Asset(name)
}
