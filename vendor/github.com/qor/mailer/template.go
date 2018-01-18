package mailer

import (
	"html/template"
	"net/http"
	"net/url"
)

// Template email template
type Template struct {
	Name    string
	Layout  string
	Data    interface{}
	Request *http.Request
	Writer  http.ResponseWriter
	funcMap template.FuncMap
}

// Funcs set template's funcs
func (tmpl Template) Funcs(funcMap template.FuncMap) Template {
	tmpl.funcMap = funcMap
	return tmpl
}

// Render render template
func (mailer Mailer) Render(t Template) Email {
	var email Email

	if t.funcMap == nil {
		t.funcMap = template.FuncMap{}
	}

	if _, ok := t.funcMap["root_url"]; !ok {
		t.funcMap["root_url"] = func() string {
			if t.Request != nil && t.Request.URL != nil {
				var newURL url.URL
				newURL.Host = t.Request.URL.Host
				newURL.Scheme = t.Request.URL.Scheme
				return newURL.String()
			}
			return ""
		}
	}

	if t.Layout != "" {
		if result, err := mailer.Config.Render.Layout(t.Layout+".text").Funcs(t.funcMap).Render(t.Name+".text", t.Data, t.Request, t.Writer); err == nil {
			email.Text = string(result)
		}

		if result, err := mailer.Config.Render.Layout(t.Layout+".html").Funcs(t.funcMap).Render(t.Name+".html", t.Data, t.Request, t.Writer); err == nil {
			email.HTML = string(result)
		} else if result, err := mailer.Config.Render.Layout(t.Layout).Funcs(t.funcMap).Render(t.Name, t.Data, t.Request, t.Writer); err == nil {
			email.HTML = string(result)
		}
	} else {
		if result, err := mailer.Config.Render.Funcs(t.funcMap).Render(t.Name+".text", t.Data, t.Request, t.Writer); err == nil {
			email.Text = string(result)
		}

		if result, err := mailer.Config.Render.Funcs(t.funcMap).Render(t.Name+".html", t.Data, t.Request, t.Writer); err == nil {
			email.HTML = string(result)
		} else if result, err := mailer.Config.Render.Funcs(t.funcMap).Render(t.Name, t.Data, t.Request, t.Writer); err == nil {
			email.HTML = string(result)
		}
	}

	return email
}
