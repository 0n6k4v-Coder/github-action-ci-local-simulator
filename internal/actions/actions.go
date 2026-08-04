package actions

import (
	"fmt"
	"strings"
)

// ActionRef represents a parsed action reference (e.g. actions/checkout@v4).
type ActionRef struct {
	Raw   string
	Owner string
	Repo  string
	Ref   string
}

// ActionName returns the "owner/repo" format of the action.
func (a ActionRef) ActionName() string {
	if a.Owner != "" && a.Repo != "" {
		return a.Owner + "/" + a.Repo
	}
	return a.Raw
}

// ParseActionRef parses a step's uses string into an ActionRef.
func ParseActionRef(uses string) (ActionRef, error) {
	uses = strings.TrimSpace(uses)
	if uses == "" {
		return ActionRef{}, fmt.Errorf("empty action reference")
	}

	raw := uses
	ref := ""
	if idx := strings.Index(uses, "@"); idx != -1 {
		ref = uses[idx+1:]
		uses = uses[:idx]
	}

	parts := strings.Split(uses, "/")
	if len(parts) >= 2 {
		return ActionRef{
			Raw:   raw,
			Owner: parts[0],
			Repo:  parts[1],
			Ref:   ref,
		}, nil
	}

	return ActionRef{
		Raw: raw,
		Ref: ref,
	}, nil
}

// ActionResult holds output logs and environment changes from an action simulation.
type ActionResult struct {
	Stdout string
	Stderr string
	Env    map[string]string
}

// UnsupportedActionError represents an error when an action is not supported. Exit code 3.
type UnsupportedActionError struct {
	Action string
}

func (e *UnsupportedActionError) Error() string {
	return fmt.Sprintf("unsupported action: %s", e.Action)
}

func (e *UnsupportedActionError) Code() int {
	return 3
}

// NewUnsupportedActionError creates a new UnsupportedActionError.
func NewUnsupportedActionError(action string) *UnsupportedActionError {
	return &UnsupportedActionError{Action: action}
}

// ActionValidationError represents a validation error for action inputs. Exit code 2.
type ActionValidationError struct {
	Message string
}

func (e *ActionValidationError) Error() string {
	return e.Message
}

func (e *ActionValidationError) Code() int {
	return 2
}

// NewActionValidationError creates a new ActionValidationError.
func NewActionValidationError(msg string) *ActionValidationError {
	return &ActionValidationError{Message: msg}
}
