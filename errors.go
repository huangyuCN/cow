package cow

import "errors"

var (
	ErrNoSession        = errors.New("cow: no active session")
	ErrSessionClosed    = errors.New("cow: session already finished")
	ErrInvalidSavepoint = errors.New("cow: invalid savepoint")
)
