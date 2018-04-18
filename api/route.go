package api

import (
	"github.com/banzaicloud/pipeline/auth"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"net/http"
)

func RedirectRoot(c *gin.Context) {
	currentUser := auth.Auth.GetCurrentUser(c.Request)
	if currentUser != nil {
		c.Redirect(http.StatusTemporaryRedirect, viper.GetString("pipeline.uipath"))
	} else {
		c.Redirect(http.StatusTemporaryRedirect, "/auth/github/login")
	}
	return
}
