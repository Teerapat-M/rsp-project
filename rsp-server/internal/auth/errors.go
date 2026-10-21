package auth

import "errors"

var (
	ErrUnauthorized   = errors.New("unauthorized")
	ErrValidation     = errors.New("validation")
	ErrUsernameTaken  = errors.New("username is taken")
	ErrEmailTaken     = errors.New("email is taken")
	ErrNotGuest       = errors.New("only a guest can upgrade this account")
	ErrGoogleDisabled = errors.New("google sign-in is not configured")
	ErrGoogleToken    = errors.New("invalid google token")
	ErrResetInvalid   = errors.New("reset token is invalid or expired")
	ErrDevOnly        = errors.New("this endpoint is only available in development")
)
