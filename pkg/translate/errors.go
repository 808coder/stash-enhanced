package translate

import (
	"errors"
	"fmt"
)

// ErrorKind classifies translation failures, so batch jobs can decide whether
// to skip an item or abort.
type ErrorKind int

const (
	// ErrorKindConnection: the provider could not be reached (host down,
	// DNS failure, timeout). Not retried.
	ErrorKindConnection ErrorKind = iota
	// ErrorKindAuth: the provider rejected the credentials (HTTP 401/403).
	// Not retried.
	ErrorKindAuth
	// ErrorKindRateLimit: the provider kept answering HTTP 429/503 after all
	// retries.
	ErrorKindRateLimit
	// ErrorKindResponse: any other unexpected status or an unusable answer.
	ErrorKindResponse
)

func (k ErrorKind) String() string {
	switch k {
	case ErrorKindConnection:
		return "connection"
	case ErrorKindAuth:
		return "auth"
	case ErrorKindRateLimit:
		return "rate limit"
	case ErrorKindResponse:
		return "response"
	default:
		return "unknown"
	}
}

// Error is a classified translation failure.
type Error struct {
	Kind ErrorKind
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("translation %s error: %v", e.Kind, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// IsKind reports whether err is a translation Error of the given kind.
func IsKind(err error, kind ErrorKind) bool {
	var te *Error
	return errors.As(err, &te) && te.Kind == kind
}

func newError(kind ErrorKind, format string, args ...interface{}) *Error {
	return &Error{Kind: kind, Err: fmt.Errorf(format, args...)}
}
