package httpclient

import (
	"fmt"
	"strconv"
	"strings"
)

// HttpError is an error that wraps an HTTP response and/or an error.
type HttpError struct {
	Url        string
	StatusCode int
	Body       string
	Err        error
}

// Error returns the error message. It implements the error interface.
func (e *HttpError) Error() string {
	httpStatusCode := "no status"
	if e.StatusCode > 0 {
		httpStatusCode = strconv.Itoa(e.StatusCode)
	}
	return fmt.Sprintf("request to %v failed. "+
		"httpStatus: [ %v ] responseBody: [ %v ] "+
		"error: [ %v ]", e.Url, httpStatusCode, e.Body, e.Err)
}

// sameStatusCodes checks whether status codes are
// equal, if `anotherStatus` is greater than zero.
func sameStatusCodes(status, anotherStatus int) bool {
	return status == anotherStatus || anotherStatus == 0
}

// sameBodies checks whether bodies are equal,
// if `anotherBody` is not empty.
func sameBodies(body, anotherBody string) bool {
	return strings.Contains(body, anotherBody) || anotherBody == ""
}

// sameErrors checks whether errors are equal.
func sameErrors(err, anotherErr error) bool {
	return err == anotherErr
}

// Is returns true if the error is an HTTPError with the given
// status code, body and error.
func (e *HttpError) Is(targetErr error) bool {
	if targetErr == nil {
		return false
	}
	t, ok := targetErr.(*HttpError)
	if !ok {
		return false
	}
	return sameStatusCodes(e.StatusCode, t.StatusCode) &&
		sameBodies(e.Body, t.Body) &&
		sameErrors(e.Err, t.Err)
}
