package errorutil

import (
	"errors"
	"github.com/adamluzsi/frameless/pkg/internal/constant"
)

func IsUserError(err error) bool {
	_, ok := LookupUserError(err)
	return ok
}

func LookupUserError(err error) (UserError, bool) {
	var ue UserError
	return ue, errors.As(err, &ue)
}

type UserError struct {
	// ID is a constant string value that expresses the user's error scenario.
	// The caller who receives the error will use this code to present the UserError to their users and,
	// most likely, provide a localised error message about the error scenario to the end user.
	// Traditionally this should be a string without any white space.
	//
	// Example: "foo-is-forbidden-with-active-baz"
	ID constant.String
	// Message is the error message meant to be read by a developer working on the implementation of the caller.
	// It is not expected to be seen by end users.
	// It might be written in English for portability reasons.
	//
	// Example: "Authentication failed due to incorrect username or password."
	Message constant.String
}

func (err UserError) Error() string {
	return "[" + string(err.ID) + "] " + string(err.Message)
}

func (err UserError) With() WithErr {
	return WithErr{Err: err}
}