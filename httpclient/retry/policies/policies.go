package policies

import (
	"context"
	"net/http"
	"strings"
)

// DoNotRetry policy does not retry a failed request.
func DoNotRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	return false, nil
}

// Eof policy retries a request in case of EOF error.
func Eof(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if err != nil {
		return strings.Contains(err.Error(), "EOF"), err
	}
	return false, err
}
