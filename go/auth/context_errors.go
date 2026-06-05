package auth

import "errors"

var (
	// ErrSubjectMissing is returned when the caller's context has no Auth0
	// subject. Treat as an authentication failure (the auth middleware should
	// have populated this before the handler ran).
	ErrSubjectMissing = errors.New("auth: no Auth0 subject on context")

	// ErrAccountMissing is returned when the caller's context has no
	// account_name. Same severity as ErrSubjectMissing.
	ErrAccountMissing = errors.New("auth: no account_name on context")
)
