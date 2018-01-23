package render

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
)

// Template template struct
type Template struct {
	render             *Render
	layout             string
	usingDefaultLayout bool
	funcMap            template.FuncMap
}

// FuncMap get func maps from tmpl
func (tmpl *Template) funcMapMaker(req *http.Request, writer http.ResponseWriter) template.FuncMap {
	var funcMap = template.FuncMap{}

	for key, fc := range tmpl.render.funcMaps {
		funcMap[key] = fc
	}

	if tmpl.render.Config.FuncMapMaker != nil {
		for key, fc := range tmpl.render.Config.FuncMapMaker(tmpl.render, req, writer) {
			funcMap[key] = fc
		}
	}

	for key, fc := range tmpl.funcMap {
		funcMap[key] = fc
	}
	return funcMap
}

// Funcs register Funcs for tmpl
func (tmpl *Template) Funcs(funcMap template.FuncMap) *Template {
	tmpl.funcMap = funcMap
	return tmpl
}

// Render render tmpl
func (tmpl *Template) Render(templateName string, obj interface{}, request *http.Request, writer http.ResponseWriter) (template.HTML, error) {
	var (
		content []byte
		t       *template.Template
		err     error
		funcMap = tmpl.funcMapMaker(request, writer)
		render  = func(name string, objs ...interface{}) (template.HTML, error) {
			var (
				err           error
				renderObj     interface{}
				renderContent []byte
			)

			if len(objs) == 0 {
				// default obj
				renderObj = obj
			} else {
				// overwrite obj
				for _, o := range objs {
					renderObj = o
					break
				}
			}

			if renderContent, err = tmpl.findTemplate(name); err == nil {
				var partialTemplate *template.Template
				result := bytes.NewBufferString("")
				if partialTemplate, err = template.New(filepath.Base(name)).Funcs(funcMap).Parse(string(renderContent)); err == nil {
					if err = partialTemplate.Execute(result, renderObj); err == nil {
						return template.HTML(result.String()), err
					}
				}
			} else {
				err = fmt.Errorf("failed to find template: %v", name)
			}

			if err != nil {
				fmt.Println(err)
			}

			return "", err
		}
	)

	// funcMaps
	funcMap["render"] = render
	funcMap["yield"] = func() (template.HTML, error) { return render(templateName) }

	layout := tmpl.layout
	usingDefaultLayout := false

	if layout == "" && tmpl.usingDefaultLayout {
		usingDefaultLayout = true
		layout = tmpl.render.DefaultLayout
	}

	if layout != "" {
		content, err = tmpl.findTemplate(filepath.Join("layouts", layout))
		if err == nil {
			if t, err = template.New("").Funcs(funcMap).Parse(string(content)); err == nil {
				var tpl bytes.Buffer
				if err = t.Execute(&tpl, obj); err == nil {
					return template.HTML(tpl.String()), nil
				}
			}
		} else if !usingDefaultLayout {
			err = fmt.Errorf("Failed to render layout: '%v.tmpl', got error: %v", filepath.Join("layouts", tmpl.layout), err)
			fmt.Println(err)
			return template.HTML(""), err
		}
	}

	if content, err = tmpl.findTemplate(templateName); err == nil {
		if t, err = template.New("").Funcs(funcMap).Parse(string(content)); err == nil {
			var tpl bytes.Buffer
			if err = t.Execute(&tpl, obj); err == nil {
				return template.HTML(tpl.String()), nil
			}
		}
	} else {
		err = fmt.Errorf("failed to find template: %v", templateName)
	}

	if err != nil {
		fmt.Println(err)
	}
	return template.HTML(""), err
}

// Execute execute tmpl
func (tmpl *Template) Execute(templateName string, obj interface{}, req *http.Request, w http.ResponseWriter) error {
	result, err := tmpl.Render(templateName, obj, req, w)
	if err == nil {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "text/html")
		}

		_, err = w.Write([]byte(result))
	}
	return err
}

func (tmpl *Template) findTemplate(name string) ([]byte, error) {
	return tmpl.render.Asset(strings.TrimSpace(name) + ".tmpl")
}
