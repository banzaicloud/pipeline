package mailer

import (
	"fmt"
	"html/template"
	"testing"
)

func TestTemplateAddFuncMap(t *testing.T) {
	tmpl := Template{
		Name:   "template",
		Layout: "layout",
		Data:   "data",
	}

	newTmpl := tmpl.Funcs(template.FuncMap{
		"hello": "hello",
	})

	if newTmpl.Name != "template" || newTmpl.Layout != "layout" || fmt.Sprint(newTmpl.Data) != "data" {
		t.Errorf("no data should lost after assign funcmap")
	}

	if _, ok := newTmpl.funcMap["hello"]; !ok {
		t.Errorf("func map should be added")
	}
}
