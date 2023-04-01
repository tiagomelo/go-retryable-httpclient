package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// For ease of unit testing.
// Declaring these functions as global variables
// makes it easy to mock them.
var (
	jsonEncode = func(w io.Writer, data any) error {
		return json.NewEncoder(w).Encode(data)
	}
	newRequestWithContext = http.NewRequestWithContext
)

// NewRequest returns an *http.Request.
func NewRequest(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := newRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating request")
	}
	return req, nil
}

// NewRequestWithHeaders returns an *http.Request with
// specified headers.
func NewRequestWithHeaders(ctx context.Context, method, url string,
	headers map[string]string) (*http.Request, error) {
	req, err := newRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating request")
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	return req, nil
}

// NewJsonRequest returns an *http.Request with a json encoded body.
func NewJsonRequest(ctx context.Context, method,
	url string, data any) (*http.Request, error) {
	body, err := body(data)
	if err != nil {
		return nil, err
	}
	req, err := newRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "creating request")
	}
	req.Header = http.Header{
		"Content-Type": {"application/json"},
	}
	return req, nil
}

// NewJsonRequestWithHeaders returns an *http.Request with a json encoded body
// and specified headers.
func NewJsonRequestWithHeaders(ctx context.Context, method, url string,
	data any, headers map[string]string) (*http.Request, error) {
	req, err := NewJsonRequest(ctx, method, url, data)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	return req, nil
}

// body returns the appropriate payload.
func body(data any) (io.Reader, error) {
	var body io.Reader
	var j []byte
	switch p := data.(type) {
	case nil:
		body = nil
	case string:
		j = []byte(p)
		body = bytes.NewBuffer(j)
	default:
		var buf bytes.Buffer
		if err := jsonEncode(&buf, data); err != nil {
			return nil, errors.Wrap(err, "encoding request payload")
		}
		body = &buf
	}
	return body, nil
}
