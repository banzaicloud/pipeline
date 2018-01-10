package password

import (
	"net/mail"
	"path"
	"reflect"
	"strings"
	"time"

	"html/template"

	"github.com/qor/auth"
	"github.com/qor/auth/auth_identity"
	"github.com/qor/auth/claims"
	"github.com/qor/mailer"
	"github.com/qor/qor/utils"
	"github.com/qor/session"
)

var (
	// ResetPasswordMailSubject reset password mail's subject
	ResetPasswordMailSubject = "Reset your password"

	// SendChangePasswordMailFlashMessage send change password mail flash message
	SendChangePasswordMailFlashMessage = template.HTML("You will receive an email with instructions on how to reset your password in a few minutes")

	// ChangedPasswordFlashMessage changed password success flash message
	ChangedPasswordFlashMessage = template.HTML("Changed your password!")
)

// DefaultResetPasswordMailer default reset password mailer
var DefaultResetPasswordMailer = func(email string, context *auth.Context, claims *claims.Claims, currentUser interface{}) error {
	claims.Subject = "reset_password"

	return context.Auth.Mailer.Send(
		mailer.Email{
			TO:      []mail.Address{{Address: email}},
			Subject: ResetPasswordMailSubject,
		}, mailer.Template{
			Name:    "auth/reset_password",
			Data:    context,
			Request: context.Request,
			Writer:  context.Writer,
		}.Funcs(template.FuncMap{
			"current_user": func() interface{} {
				return currentUser
			},
			"reset_password_url": func() string {
				resetPasswordURL := utils.GetAbsURL(context.Request)
				resetPasswordURL.Path = path.Join(context.Auth.AuthURL("password/edit"))
				qry := resetPasswordURL.Query()
				qry.Set("token", context.SessionStorer.SignedToken(claims))
				resetPasswordURL.RawQuery = qry.Encode()
				return resetPasswordURL.String()
			},
		}),
	)
}

// DefaultRecoverPasswordHandler default reset password handler
var DefaultRecoverPasswordHandler = func(context *auth.Context) error {
	context.Request.ParseForm()

	var (
		authInfo    auth_identity.Basic
		email       = context.Request.Form.Get("email")
		provider, _ = context.Provider.(*Provider)
	)

	authInfo.Provider = provider.GetName()
	authInfo.UID = strings.TrimSpace(email)

	currentUser, err := context.Auth.UserStorer.Get(authInfo.ToClaims(), context)

	if err != nil {
		return err
	}

	err = provider.ResetPasswordMailer(email, context, authInfo.ToClaims(), currentUser)

	if err == nil {
		context.SessionStorer.Flash(context.Writer, context.Request, session.Message{Message: SendChangePasswordMailFlashMessage, Type: "success"})
		context.Auth.Redirector.Redirect(context.Writer, context.Request, "send_recover_password_mail")
	}
	return err
}

// DefaultResetPasswordHandler default reset password handler
var DefaultResetPasswordHandler = func(context *auth.Context) error {
	context.Request.ParseForm()

	var (
		authInfo    auth_identity.Basic
		token       = context.Request.Form.Get("reset_password_token")
		provider, _ = context.Provider.(*Provider)
		tx          = context.Auth.GetDB(context.Request)
	)

	claims, err := context.SessionStorer.ValidateClaims(token)

	if err == nil {
		if err = claims.Valid(); err == nil {
			authInfo.Provider = provider.GetName()
			authInfo.UID = claims.Id
			authIdentity := reflect.New(utils.ModelType(context.Auth.Config.AuthIdentityModel)).Interface()

			if tx.Where(authInfo).First(authIdentity).RecordNotFound() {
				return auth.ErrInvalidAccount
			}

			if authInfo.EncryptedPassword, err = provider.Encryptor.Digest(strings.TrimSpace(context.Request.Form.Get("new_password"))); err == nil {
				// Confirm account after reset password, as user already click a link from email
				if provider.Config.Confirmable && authInfo.ConfirmedAt == nil {
					now := time.Now()
					authInfo.ConfirmedAt = &now
				}
				err = tx.Model(authIdentity).Update(authInfo).Error
			}
		}
	}

	if err == nil {
		context.SessionStorer.Flash(context.Writer, context.Request, session.Message{Message: ChangedPasswordFlashMessage, Type: "success"})
		context.Auth.Redirector.Redirect(context.Writer, context.Request, "reset_password")
	}
	return err
}
