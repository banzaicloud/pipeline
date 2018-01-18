package auth

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/qor/auth/claims"
	"github.com/qor/responder"
	"github.com/qor/session"
)

func respondAfterLogged(claims *claims.Claims, context *Context) {
	// login user
	context.Auth.Login(context.Writer, context.Request, claims)

	responder.With("html", func() {
		// write cookie
		context.Auth.Redirector.Redirect(context.Writer, context.Request, "login")
	}).With([]string{"json"}, func() {
		// TODO write json token
	}).Respond(context.Request)
}

// DefaultLoginHandler default login behaviour
var DefaultLoginHandler = func(context *Context, authorize func(*Context) (*claims.Claims, error)) {
	var (
		req         = context.Request
		w           = context.Writer
		claims, err = authorize(context)
	)

	if err == nil && claims != nil {
		context.SessionStorer.Flash(w, req, session.Message{Message: "logged"})
		respondAfterLogged(claims, context)
		return
	}

	context.SessionStorer.Flash(w, req, session.Message{Message: template.HTML(err.Error()), Type: "error"})

	// error handling
	responder.With("html", func() {
		context.Auth.Config.Render.Execute("auth/login", context, req, w)
	}).With([]string{"json"}, func() {
		// TODO write json error
	}).Respond(context.Request)
}

// DefaultRegisterHandler default register behaviour
var DefaultRegisterHandler = func(context *Context, register func(*Context) (*claims.Claims, error)) {
	var (
		req         = context.Request
		w           = context.Writer
		claims, err = register(context)
	)

	if err == nil && claims != nil {
		respondAfterLogged(claims, context)
		return
	}

	context.SessionStorer.Flash(w, req, session.Message{Message: template.HTML(err.Error()), Type: "error"})

	// error handling
	responder.With("html", func() {
		context.Auth.Config.Render.Execute("auth/register", context, req, w)
	}).With([]string{"json"}, func() {
		// TODO write json error
	}).Respond(context.Request)
}

// DefaultLogoutHandler default logout behaviour
var DefaultLogoutHandler = func(context *Context) {
	// Clear auth session
	context.SessionStorer.Delete(context.Writer, context.Request)
	context.Auth.Redirector.Redirect(context.Writer, context.Request, "logout")
}

var cacheSince = time.Now().Format(http.TimeFormat)

// DefaultAssetHandler render auth asset file
var DefaultAssetHandler = func(context *Context) {
	asset := strings.TrimPrefix(context.Request.URL.Path, context.Auth.URLPrefix)

	if context.Request.Header.Get("If-Modified-Since") == cacheSince {
		context.Writer.WriteHeader(http.StatusNotModified)
		return
	}
	context.Writer.Header().Set("Last-Modified", cacheSince)

	if content, err := context.Config.Render.Asset(path.Join("/auth", asset)); err == nil {
		etag := fmt.Sprintf("%x", md5.Sum(content))
		if context.Request.Header.Get("If-None-Match") == etag {
			context.Writer.WriteHeader(http.StatusNotModified)
			return
		}

		if ctype := mime.TypeByExtension(filepath.Ext(asset)); ctype != "" {
			context.Writer.Header().Set("Content-Type", ctype)
		}

		context.Writer.Header().Set("Cache-control", "private, must-revalidate, max-age=300")
		context.Writer.Header().Set("ETag", etag)
		context.Writer.Write(content)
	} else {
		http.NotFound(context.Writer, context.Request)
	}
}
