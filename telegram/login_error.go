package telegram

// LoginError identifies a failed step without private request data.
type LoginError struct {
	Code string
}

func (e *LoginError) Error() string { return e.Code }

func loginError(code string) error { return &LoginError{Code: code} }
