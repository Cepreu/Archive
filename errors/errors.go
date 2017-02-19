/*
Package errors specifies constants and utilities for error handling.

Error messages should be easy to understand and identify so that
a troubleshooting guide (TSG) can reference a specific error and explain what
to do when said error occurs. In order to do so, an error has to have a unique
identifier across all systems; for example:
    WF12345: HTTP response status code was not 2xx; response: ...
Following the above format allows for searching logs for the occurances of
the specific error and for an easy way to communicate which error it is in bugs,
emails, etc.

Error message reuse allows for a TSG to address failure modes (instead
of incident-specific errors); for example:
    WF13579: Invalid credentials specified for user; user ID: ...
Said error is going to occur in multiple subsystems and it should be reused; it
also represents a failure mode that should be mapped to a single TSG. Free-form
error messages are hard to map into a single TSG and may confuse readers into
thinking that two errors represent two distinct issues.

Moreover, exposing errors as functions enforces a strong contract of format
spec using function parameters to minimize formatting errors in runtime
(although in Go that can be vetted using go vet). In addition to searchability,
consistency, reuse, and strong contracts; error message functions have
the following benefits:
    - Flexibility: can preprocess parameters before formatting (in one place)
    - Structured: forcing error messages to be structured and indexable
    - Logging: errors are automatically logged using structured logs
    - Testing: it's a must; if error reporting fails, we fly blind!
    - Readability: parameter names, documentation, etc.
    - Comments (e.g., to add a link to a bug report)
    - Code brevity; no need to inject long strings
    - They are easier to localize (if need be)

This package replaces the built-in errors package and should be used instead.
*/
package errors

import (
	"errors"

	common "github.com/WorkFit/commongo/errors"
	"github.com/WorkFit/go/log"
)

const wf10001 = `WF10001: crypto error`

// WF10001 occurs when a crypto error occurs; it's intentionally ambigious
// to prevent giving up any useful info in case a system is compromised
// and attackers have access to logs.
func WF10001() error {
	log.Error(wf10001)
	return newError(wf10001)
}

const wf11200 = `WF11200: HTTP response status code was not 2xx`

// WF11200 occurs when an HTTP reponse has a status code other than 2xx.
func WF11200(response interface{}) error {
	log.Error(wf11200, "response", response)
	return newError(wf11200)
}

const wf11201 = `WF11201: partial success`

// WF11201 occurs when an operation succeeds but only partially.
func WF11201(request interface{}, response interface{}) error {
	log.Error(wf11201, "request", request, "response", response)
	return newError(wf11201)
}

const wf11301 = `WF11301: all attempts failed with the following errors:`

// WF11301 occurs when all attempts failed with an aggregate error.
func WF11301(errors ...error) error {
	err := common.NewAggregateError(wf11301, errors...)
	log.ErrorObject(err)
	return err
}

// newError returns an error that formats as the given text.
// It's a wrapper around Go's errors.New function to allow for creating
// errors that can be handled differently in recovery.
func newError(text string) error {
	return errors.New(text)
}
