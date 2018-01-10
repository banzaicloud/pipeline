package password

import "errors"

var (
	// ErrInvalidResetPasswordToken invalid reset password token
	ErrInvalidResetPasswordToken = errors.New("Invalid Token")
)
