# Render

Render provides handy controls when rendering templates.

## Usage

### Initialize [Render](https://github.com/qor/render)

```go
import "github.com/qor/render"

func main() {
  Render := render.New(&render.Config{
    ViewPaths:     []string{"app/new_view_path"},
    DefaultLayout: "application", // default value is application
    FuncMapMaker:  func(*Render, *http.Request, http.ResponseWriter) template.FuncMap {
      // genereate FuncMap that could be used when render template based on request info
    },
  })
}
```

Next, invoke `Execute` function to render your template...

```go
Render.Execute("index", context, request, writer)
```

The `Execute` function accepts 4 parameters:

1. The template name. In this example [Render](https://github.com/qor/render) will look up template `index.tmpl` from view paths. the default view path is `{current_repo_path}/app/views`, and you could register more view paths.
2. The context you can use in the template, it is an `interface{}`, you could use that in views. for example, if you pass `context["CurrentUserName"] = "Memememe"` as the context. In the template, you can call `{% raw %}{{.CurrentUserName}}{% endraw %}` to get the value "Memememe".
3. [http.Request](https://golang.org/pkg/net/http/#Request) of Go.
4. [http.ResponseWriter](https://golang.org/pkg/net/http/#ResponseWriter) of Go.

### Understanding `yield`

`yield` is a func that could be used in layout views, it will render current specified template. For above example, think `yield` as a placeholder, and it will replaced with template `index.tmpl`'s content.

```html
<!-- app/views/layout/application.tmpl -->
<html>
  <head>
  </head>
  <body>
    {{yield}}
  </body>
</html>
```

### Specify Layout

The default layout is `{current_repo_path}/app/views/layouts/application.tmpl`. If you want use another layout like `new_layout`, you can pass it as a parameter to `Layout` function.

```go
Render.Layout("new_layout").Execute("index", context, request, writer)
```

[Render](https://github.com/qor/render) will find the layout at `{current_repo_path}/app/views/layouts/new_layout.tmpl`.

### Render with helper functions

Sometimes you may want to have some helper functions in your template. [Render](https://github.com/qor/render) supports passing helper functions by `Funcs` function.

```go
Render.Funcs(funcsMap).Execute("index", obj, request, writer)
```

The `funcsMap` is based on [html/template.FuncMap](https://golang.org/src/html/template/template.go?h=FuncMap#L305). So with

```go
funcMap := template.FuncMap{
  "Greet": func(name string) string { return "Hello " + name },
}
```

You can call this in the template

```go
{{Greet "Memememe" }}
```

The output is `Hello Memememe`.

### Use with [Responder](./responder.md)

Put the [Render](https://github.com/qor/render) inside [Responder](./responder.md) handle function like this.

```go
func handler(writer http.ResponseWriter, request *http.Request) {
  responder.With("html", func() {
    Render.Execute("demo/index", viewContext, *http.Request, http.ResponseWriter)
  }).With([]string{"json", "xml"}, func() {
    writer.Write([]byte("this is a json or xml request"))
  }).Respond(request)
})
```

### Use with [Bindatafs](../plugins/bindata.md)

```go
$ bindatafs --exit-after-compile=false config/views

func main() {
	Render := render.New()
	Render.SetAssetFS(views.AssetFS)

	Render.Execute("index", context, request, writer)
}
```

## License

Released under the [MIT License](http://opensource.org/licenses/MIT).
