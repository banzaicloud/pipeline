package auth

import (
	"net/http"

	"github.com/qor/redirect_back"
)

// RedirectorInterface redirector interface
type RedirectorInterface interface {
	// Redirect redirect after action
	Redirect(w http.ResponseWriter, req *http.Request, action string)
}

// Redirector default redirector
type Redirector struct {
	*redirect_back.RedirectBack
}

// Redirect redirect back after action
func (redirector Redirector) Redirect(w http.ResponseWriter, req *http.Request, action string) {
	redirector.RedirectBack.RedirectBack(w, req)
}
