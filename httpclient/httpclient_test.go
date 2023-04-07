package httpclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/require"
)

type (
	customRoundTriper  struct{}
	ioReadAllMock      func(r io.Reader) ([]byte, error)
	jsonDecodeMock     func(r io.Reader, data any) error
	dumpRequestOutMock func(req *http.Request, body bool) ([]byte, error)
	dumpResponseMock   func(resp *http.Response, body bool) ([]byte, error)
	dummyType          struct {
		Key string `json:"key"`
	}
)

func (c *customRoundTriper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, nil
}

func TestDefaultPolicy(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer svr.Close()
	url := svr.URL
	client := New()
	req, err := NewRequest(context.TODO(), http.MethodGet, url)
	if err != nil {
		t.Fatalf(`creating request for "%v": %v`, url, err)
	}
	_, err = client.SendRequest(req)
	expectedError := fmt.Errorf(`request to %s failed. `+
		`httpStatus: [ %d ] responseBody: [  ] `+
		`error: [ <nil> ]`, url, http.StatusBadGateway)
	require.NotNil(t, err)
	require.Equal(t, expectedError.Error(), err.Error())
}

func TestEofRetryPolicy(t *testing.T) {
	checkRetryPolicy := func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			return strings.Contains(err.Error(), "EOF"), err
		}
		return false, err
	}
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(errors.New("random error"))
	}))
	defer svr.Close()
	url := svr.URL
	client := New(WithMaxRetries(1), WithCheckRetryPolicy(checkRetryPolicy))
	req, err := NewRequest(context.TODO(), http.MethodGet, url)
	if err != nil {
		t.Fatalf(`creating request for "%v": %v`, url, err)
	}
	_, err = client.SendRequest(req)
	expectedError := fmt.Errorf(`request to %s failed. `+
		`httpStatus: [ no status ] responseBody: [  ] `+
		`error: [ GET %s giving up after 2 attempt(s): Get "%s": EOF ]`, url, url, url)
	require.NotNil(t, err)
	require.Equal(t, expectedError.Error(), err.Error())
}

func TestNew(t *testing.T) {
	testCases := []struct {
		name                        string
		options                     []Option
		expectedTimeout             time.Duration
		expectedMaxIdleConns        int
		expectedMaxIdleConnsPerHost int
		expectedMaxConnsPerHost     int
	}{
		{
			name:    "no http client provided, no options provided",
			options: []Option{},
		},
		{
			name: "no http client provided, with options",
			options: []Option{
				WithTimeout(30 * time.Second),
				WithMaxConnsPerHost(200),
				WithMaxIdleConnsPerHost(200),
				WithMaxIdleConns(200),
				WithMaxRetries(5),
				WithRetryWaitMin(1 * time.Second),
				WithRetryWaitMax(5 * time.Second),
				WithCheckRetryPolicy(func(ctx context.Context,
					resp *http.Response, err error) (bool, error) {
					if resp != nil {
						statusCode := resp.StatusCode
						if statusCode == http.StatusBadRequest {
							return true, err
						}
					}
					return false, err
				}),
				WithRequestDumpLogger(func(dump []byte) {}, false),
			},
			expectedTimeout:             time.Second * 30,
			expectedMaxConnsPerHost:     200,
			expectedMaxIdleConnsPerHost: 200,
			expectedMaxIdleConns:        200,
		},
		{
			name: "http client provided, no other options provided",
			options: []Option{
				WithHttpClient(&http.Client{Timeout: time.Duration(25 * time.Second)}),
			},
			expectedTimeout: time.Second * 25,
		},
		{
			name: "http client provided, options provided",
			options: []Option{
				WithHttpClient(&http.Client{Timeout: time.Duration(25 * time.Second)}),
				WithMaxConnsPerHost(200),
				WithMaxIdleConnsPerHost(200),
				WithMaxIdleConns(200),
			},
			expectedTimeout:             time.Second * 25,
			expectedMaxConnsPerHost:     200,
			expectedMaxIdleConnsPerHost: 200,
			expectedMaxIdleConns:        200,
		},
		{
			name: "http client provided, no transport",
			options: []Option{
				WithHttpClient(&http.Client{Timeout: time.Duration(25 * time.Second),
					Transport: &customRoundTriper{}}),
			},
			expectedTimeout:             time.Second * 25,
			expectedMaxIdleConns:        100,
			expectedMaxIdleConnsPerHost: 100,
			expectedMaxConnsPerHost:     100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := New(tc.options...)
			require.Equal(t, client.retryableHttpClient.HTTPClient.Timeout, tc.expectedTimeout)
			tr, isTransport := castClientTransport(client.retryableHttpClient.HTTPClient.Transport)
			if isTransport {
				require.Equal(t, tr.MaxIdleConns, tc.expectedMaxIdleConns)
				require.Equal(t, tr.MaxIdleConnsPerHost, tc.expectedMaxIdleConnsPerHost)
				require.Equal(t, tr.MaxConnsPerHost, tc.expectedMaxConnsPerHost)
			}
		})
	}
}

func TestSendRequest(t *testing.T) {
	testCases := []struct {
		name                      string
		requestDumpLogger         func(dump []byte)
		responseDumpLogger        func(dump []byte)
		retryableHttpClientDoMock func(retryableHttpClient *retryablehttp.Client,
			req *retryablehttp.Request) (*http.Response, error)
		ioReadAllMock      func(r io.Reader) ([]byte, error)
		dumpRequestOutMock func(req *http.Request, body bool) ([]byte, error)
		dumpResponseMock   func(resp *http.Response, body bool) ([]byte, error)
		expectedError      error
	}{
		{
			name: "happy path",
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(""))),
				}
				return resp, nil
			},
		},
		{
			name: "dumping request",
			requestDumpLogger: func(dump []byte) {
				expectedDump := "POST /some/path HTTP/1.1\r\nHost: localhost\r\nUser-Agent: Go-http-client/1.1\r\nContent-Length: 0\r\nAccept-Encoding: gzip\r\n\r\n"
				require.Equal(t, expectedDump, string(dump))
			},
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(""))),
				}
				return resp, nil
			},
		},
		{
			name: "error when dumping request",
			dumpRequestOutMock: func(req *http.Request, body bool) ([]byte, error) {
				return nil, errors.New("random error")
			},
			requestDumpLogger: func(dump []byte) {
				require.Equal(t, "", string(dump))
			},
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(""))),
				}
				return resp, nil
			},
		},
		{
			name: "dumping non-nil response",
			responseDumpLogger: func(dump []byte) {
				expectedDump := "HTTP/0.0 200 OK\r\nContent-Length: 0\r\n\r\n"
				require.Equal(t, expectedDump, string(dump))
			},
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"message":"ok"}`))),
				}
				return resp, nil
			},
		},
		{
			name: "dumping nil response",
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				return nil, nil
			},
		},
		{
			name: "error when dumping response",
			dumpResponseMock: func(resp *http.Response, body bool) ([]byte, error) {
				return nil, errors.New("random error")
			},
			responseDumpLogger: func(dump []byte) {
				require.Equal(t, "", string(dump))
			},
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(""))),
				}
				return resp, nil
			},
		},
		{
			name: "unsuccessful response with responseBody",
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"some random error"}`))),
				}
				return resp, nil
			},
			expectedError: errors.New(`request to http://localhost/some/path failed. ` +
				`httpStatus: [ 500 ] responseBody: [ {"error":"some random error"} ] ` +
				`error: [ <nil> ]`),
		},
		{
			name: "unsuccessful response with responseBody, error when parsing it",
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"some random error"}`))),
				}
				return resp, nil
			},
			ioReadAllMock: func(r io.Reader) ([]byte, error) {
				return nil, errors.New("random error")
			},
			expectedError: errors.New(`request to http://localhost/some/path failed. ` +
				`httpStatus: [ 500 ] responseBody: [  ] error: [ parsing response: random error ]`),
		},
		{
			name: "unsuccessful response with empty body, with some other error",
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte(``))),
				}
				return resp, errors.New("random error")
			},
			expectedError: errors.New(`request to http://localhost/some/path failed. ` +
				`httpStatus: [ 500 ] responseBody: [  ] error: [ random error ]`),
		},
		{
			name: "without response",
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				return nil, errors.New("random error")
			},
			expectedError: errors.New(`request to http://localhost/some/path failed. ` +
				`httpStatus: [ no status ] responseBody: [  ] error: [ random error ]`),
		},
	}
	originalIoReadAll := ioReadAll
	originalDumpRequestOut := dumpRequestOut
	originalDumpResponse := dumpResponse
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			retryableHttpClientDo = tc.retryableHttpClientDoMock
			handleIoReadAllMock(tc.ioReadAllMock, originalIoReadAll)
			handleDumpRequestOut(tc.dumpRequestOutMock, originalDumpRequestOut)
			handleDumpResponse(tc.dumpResponseMock, originalDumpResponse)
			client := New(WithRequestDumpLogger(tc.requestDumpLogger, false), WithResponseDumpLogger(tc.responseDumpLogger, false))
			req, err := http.NewRequest(http.MethodPost, "http://localhost/some/path", nil)
			if err != nil {
				t.Fatalf(`error when creating request: "%v"`, err)
			}
			_, err = client.SendRequest(req)
			if err != nil {
				checkIfErrorIsExpected(t, err, tc.expectedError)
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				checkIfErrorIsNotExpected(t, err, tc.expectedError)
			}
		})
	}
}

func TestSendRequestAndUnmarshallJsonResponse(t *testing.T) {
	testCases := []struct {
		name                      string
		retryableHttpClientDoMock func(retryableHttpClient *retryablehttp.Client,
			req *retryablehttp.Request) (*http.Response, error)
		ioReadAllMock  func(r io.Reader) ([]byte, error)
		jsonDecodeMock func(r io.Reader, data any) error
		expectedData   dummyType
		expectedError  error
	}{
		{
			name: "happy path",
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				body := `{"key":"value"}`
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(body))),
				}
				return resp, nil
			},
			expectedData: dummyType{
				Key: "value",
			},
		},
		{
			name: "error when decoding response",
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				body := `{"key":"value"}`
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(body))),
				}
				return resp, nil
			},
			jsonDecodeMock: func(r io.Reader, data any) error {
				return errors.New("random error")
			},
			expectedError: errors.New("request to http://localhost/some/path failed. " +
				"httpStatus: [ 200 ] responseBody: [  ] error: [ decoding response: random error ]"),
		},
		{
			name: "unsuccessful response with responseBody",
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"some random error"}`))),
				}
				return resp, nil
			},
			expectedError: errors.New(`request to http://localhost/some/path failed. ` +
				`httpStatus: [ 500 ] responseBody: [ {"error":"some random error"} ] error: [ <nil> ]`),
		},
		{
			name: "unsuccessful response with responseBody, error when parsing it",
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"some random error"}`))),
				}
				return resp, nil
			},
			ioReadAllMock: func(r io.Reader) ([]byte, error) {
				return nil, errors.New("random error")
			},
			expectedError: errors.New(`request to http://localhost/some/path failed. ` +
				`httpStatus: [ 500 ] responseBody: [  ] error: [ parsing response: random error ]`),
		},
		{
			name: "unsuccessful response with empty body, with some other error",
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte(``))),
				}
				return resp, errors.New("random error")
			},
			expectedError: errors.New(`request to http://localhost/some/path failed. ` +
				`httpStatus: [ 500 ] responseBody: [  ] error: [ random error ]`),
		},
		{
			name: "without response",
			retryableHttpClientDoMock: func(retryableHttpClient *retryablehttp.Client,
				req *retryablehttp.Request) (*http.Response, error) {
				return nil, errors.New("random error")
			},
			expectedError: errors.New(`request to http://localhost/some/path failed. ` +
				`httpStatus: [ no status ] responseBody: [  ] error: [ random error ]`),
		},
	}
	originalIoReadAll := ioReadAll
	originalJsonDecode := jsonDecode
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			retryableHttpClientDo = tc.retryableHttpClientDoMock
			handleIoReadAllMock(tc.ioReadAllMock, originalIoReadAll)
			handleJsonDecodeMock(tc.jsonDecodeMock, originalJsonDecode)
			client := New()
			req, err := http.NewRequest(http.MethodPost, "http://localhost/some/path", nil)
			if err != nil {
				t.Fatalf(`error when creating request: "%v"`, err)
			}
			var data dummyType
			resp, err := client.SendRequestAndUnmarshallJsonResponse(req, &data)
			if err != nil {
				checkIfErrorIsExpected(t, err, tc.expectedError)
				require.Equal(t, tc.expectedError.Error(), err.Error())
			} else {
				checkIfErrorIsNotExpected(t, err, tc.expectedError)
				require.NotNil(t, resp)
				require.Equal(t, tc.expectedData, data)
			}
		})
	}
}

func handleIoReadAllMock(mocked ioReadAllMock, original ioReadAllMock) {
	if mocked != nil {
		ioReadAll = mocked
	} else {
		ioReadAll = original
	}
}

func handleJsonDecodeMock(mocked jsonDecodeMock, original jsonDecodeMock) {
	if mocked != nil {
		jsonDecode = mocked
	} else {
		jsonDecode = original
	}
}

func handleDumpRequestOut(mocked dumpRequestOutMock, original dumpRequestOutMock) {
	if mocked != nil {
		dumpRequestOut = mocked
	} else {
		dumpRequestOut = original
	}
}

func handleDumpResponse(mocked dumpResponseMock, original dumpResponseMock) {
	if mocked != nil {
		dumpResponse = mocked
	} else {
		dumpResponse = original
	}
}
