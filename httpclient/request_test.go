package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	originalNewRequestWithContext = newRequestWithContext
	originalJsonEncode            = jsonEncode
)

type (
	newRequestMock = func(ctx context.Context, method string,
		url string, body io.Reader) (*http.Request, error)
	jsonEncodeMock = func(w io.Writer, data any) error
)

func TestNewRequest(t *testing.T) {
	testCases := []struct {
		name           string
		mockNewRequest func(ctx context.Context, method string,
			url string, body io.Reader) (*http.Request, error)
		expectedError error
	}{
		{
			name: "happy path",
		},
		{
			name: "error creating request",
			mockNewRequest: func(ctx context.Context, method,
				url string, body io.Reader) (*http.Request, error) {
				return nil, errors.New("random error")
			},
			expectedError: errors.New("creating request: random error"),
		},
	}
	originalNewRequestWithContext = newRequestWithContext
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handleNewRequestMock(tc.mockNewRequest, originalNewRequestWithContext)
			req, err := NewRequest(context.TODO(), "method", "url")
			if err != nil {
				checkIfErrorIsExpected(t, err, tc.expectedError)
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				checkIfErrorIsNotExpected(t, err, tc.expectedError)
				require.NotNil(t, req)
			}
		})
	}
}

func TestNewRequestWithHeaders(t *testing.T) {
	testCases := []struct {
		name           string
		mockNewRequest func(ctx context.Context, method string,
			url string, body io.Reader) (*http.Request, error)
		headers         map[string]string
		expectedHeaders http.Header
		expectedError   error
	}{
		{
			name: "happy path",
			headers: map[string]string{
				"Custom-Header-1": "xxx",
				"Custom-Header-2": "yyy",
			},
			expectedHeaders: http.Header{
				"Custom-Header-1": {"xxx"},
				"Custom-Header-2": {"yyy"},
			},
		},
		{
			name: "error creating request",
			mockNewRequest: func(ctx context.Context, method,
				url string, body io.Reader) (*http.Request, error) {
				return nil, errors.New("random error")
			},
			expectedError: errors.New("creating request: random error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handleNewRequestMock(tc.mockNewRequest, originalNewRequestWithContext)
			req, err := NewRequestWithHeaders(context.TODO(), "method", "url", tc.headers)
			if err != nil {
				checkIfErrorIsExpected(t, err, tc.expectedError)
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				checkIfErrorIsNotExpected(t, err, tc.expectedError)
				headers := req.Header
				require.Equal(t, tc.expectedHeaders, headers)
			}
		})
	}
}

func TestNewJsonRequestWithHeaders(t *testing.T) {
	testCases := []struct {
		name           string
		payload        any
		mockNewRequest func(ctx context.Context, method string,
			url string, body io.Reader) (*http.Request, error)
		headers         map[string]string
		expectedReqBody string
		expectedHeaders http.Header
		expectedError   error
	}{
		{
			name:    "adds all provided headers",
			payload: map[string]string{"key": "value"},
			headers: map[string]string{
				"Custom-Header-1": "xxx",
				"Custom-Header-2": "yyy",
			},
			mockNewRequest: func(ctx context.Context, method,
				url string, body io.Reader) (*http.Request, error) {
				req := new(http.Request)
				req.Body = io.NopCloser(body)
				return req, nil
			},
			expectedReqBody: `{"key":"value"}`,
			expectedHeaders: http.Header{
				"Content-Type":    {"application/json"},
				"Custom-Header-1": {"xxx"},
				"Custom-Header-2": {"yyy"},
			},
		},
		{
			name:    "error creating request",
			payload: map[string]string{"key": "value"},
			mockNewRequest: func(ctx context.Context, method,
				url string, body io.Reader) (*http.Request, error) {
				return nil, errors.New("random error")
			},
			expectedError: errors.New("creating request: random error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handleNewRequestMock(tc.mockNewRequest, originalNewRequestWithContext)
			req, err := NewJsonRequestWithHeaders(context.TODO(), "method", "url", tc.payload, tc.headers)
			if err != nil {
				checkIfErrorIsExpected(t, err, tc.expectedError)
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				checkIfErrorIsNotExpected(t, err, tc.expectedError)
				checkExpectedReqBody(t, req.Body, tc.expectedReqBody)
				headers := req.Header
				require.Equal(t, tc.expectedHeaders, headers)
			}
		})
	}
}

func TestNewJsonRequest(t *testing.T) {
	testCases := []struct {
		name           string
		payload        any
		mockJsonEncode func(w io.Writer, data any) error
		mockNewRequest func(ctx context.Context, method string,
			url string, body io.Reader) (*http.Request, error)
		expectedReqBody string
		expectedHeader  http.Header
		expectedError   error
	}{
		{
			name:    "happy path with payload",
			payload: map[string]string{"key": "value"},
			mockNewRequest: func(ctx context.Context, method,
				url string, body io.Reader) (*http.Request, error) {
				req := new(http.Request)
				req.Body = io.NopCloser(body)
				return req, nil
			},
			expectedReqBody: `{"key":"value"}`,
			expectedHeader: http.Header{
				"Content-Type": {"application/json"},
			},
		},
		{
			name:    "happy path with string payload",
			payload: `{"key":"value"}`,
			mockNewRequest: func(ctx context.Context, method,
				url string, body io.Reader) (*http.Request, error) {
				req := new(http.Request)
				req.Body = io.NopCloser(body)
				return req, nil
			},
			expectedReqBody: `{"key":"value"}`,
			expectedHeader: http.Header{
				"Content-Type": {"application/json"},
			},
		},
		{
			name: "happy path without payload",
			mockNewRequest: func(ctx context.Context, method,
				url string, body io.Reader) (*http.Request, error) {
				rc, ok := body.(io.ReadCloser)
				if !ok && body != nil {
					rc = io.NopCloser(body)
				}
				req := new(http.Request)
				req.Body = rc
				return req, nil
			},
			expectedHeader: http.Header{
				"Content-Type": {"application/json"},
			},
		},
		{
			name:    "error encoding payload",
			payload: map[string]string{"key": "value"},
			mockJsonEncode: func(w io.Writer, data any) error {
				return errors.New("random error")
			},
			expectedError: errors.New("encoding request payload: random error"),
		},
		{
			name:    "error creating request",
			payload: map[string]string{"key": "value"},
			mockNewRequest: func(ctx context.Context, method,
				url string, body io.Reader) (*http.Request, error) {
				return nil, errors.New("random error")
			},
			expectedError: errors.New("creating request: random error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handleJsonEncodeMock(tc.mockJsonEncode, originalJsonEncode)
			handleNewRequestMock(tc.mockNewRequest, originalNewRequestWithContext)
			req, err := NewJsonRequest(context.TODO(), "method", "url", tc.payload)
			if err != nil {
				checkIfErrorIsExpected(t, err, tc.expectedError)
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				checkIfErrorIsNotExpected(t, err, tc.expectedError)
				checkExpectedReqBody(t, req.Body, tc.expectedReqBody)
				headers := req.Header
				require.Equal(t, tc.expectedHeader, headers)
			}
		})
	}
}

func handleNewRequestMock(mocked newRequestMock, original newRequestMock) {
	if mocked != nil {
		newRequestWithContext = mocked
	} else {
		newRequestWithContext = original
	}
}

func handleJsonEncodeMock(mocked jsonEncodeMock, original jsonEncodeMock) {
	if mocked != nil {
		jsonEncode = mocked
	} else {
		jsonEncode = original
	}
}

func checkExpectedReqBody(t *testing.T, reqBody io.ReadCloser, expectedReqBody string) {
	if expectedReqBody != "" {
		b, err := io.ReadAll(reqBody)
		require.NoError(t, err)
		require.JSONEq(t, expectedReqBody, string(b))
	} else {
		require.Nil(t, reqBody)
	}
}
