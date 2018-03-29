// Package responder respond differently according to request's accepted mime type
//
// Github: http://github.com/qor/responder
package responder

import (
	"net/http"
	"path/filepath"
	"strings"
)

// registered mime types
var mimeTypes = map[string]string{}

// Register new mime type and format
//     responder.Register("application/json", "json")
func Register(mime string, format string) {
	mimeTypes[mime] = format
}

func init() {
	for mimeType, format := range map[string]string{
		"text/html":        "html",
		"application/json": "json",
		"application/xml":  "xml",
	} {
		Register(mimeType, format)
	}
}

// Responder is holder of registed response handlers, response `Request` based on its accepted mime type
type Responder struct {
	responds map[string]func()
}

// With could be used to register response handler for mime type formats, the formats could be string or []string
//     responder.With("html", func() {
//       writer.Write([]byte("this is a html request"))
//     }).With([]string{"json", "xml"}, func() {
//       writer.Write([]byte("this is a json or xml request"))
//     })
func With(formats interface{}, fc func()) *Responder {
	rep := &Responder{responds: map[string]func(){}}
	return rep.With(formats, fc)
}

// With could be used to register response handler for mime type formats, the formats could be string or []string
func (rep *Responder) With(formats interface{}, fc func()) *Responder {
	if f, ok := formats.(string); ok {
		rep.responds[f] = fc
	} else if fs, ok := formats.([]string); ok {
		for _, f := range fs {
			rep.responds[f] = fc
		}
	}
	return rep
}

// Respond differently according to request's accepted mime type
func (rep *Responder) Respond(request *http.Request) {
	// get request format from url
	if ext := filepath.Ext(request.URL.Path); ext != "" {
		if respond, ok := rep.responds[strings.TrimPrefix(ext, ".")]; ok {
			respond()
			return
		}
	}

	// get request format from Accept
	for _, accept := range strings.Split(request.Header.Get("Accept"), ",") {
		if format, ok := mimeTypes[accept]; ok {
			if respond, ok := rep.responds[format]; ok {
				respond()
				return
			}
		}
	}

	// use first format as default
	for _, respond := range rep.responds {
		respond()
		break
	}
	return
}
