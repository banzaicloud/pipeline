package auth

import (
	"net/http"

	"github.com/jinzhu/gorm"
)

type User struct {
	gorm.Model
	Name  string `form:"name"`
	Email string `form:"email"`
}

// GetCurrentUser get current user from request
func GetCurrentUser(req *http.Request) *User {
	if currentUser, ok := Auth.GetCurrentUser(req).(*User); ok {
		return currentUser
	}
	return nil
}
