// Package httpclient provides a tiny http client with retry capability.
package httpclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
)

// For ease of unit testing.
// Declaring these functions as global variables
// makes it easy to mock them.
var (
	retryableHttpClientDo = func(retryableHttpClient *retryablehttp.Client,
		req *retryablehttp.Request) (*http.Response, error) {
		return retryableHttpClient.Do(req)
	}
	handleUnsuccessfulResponse = func(url string, resp *http.Response,
		receivedError error) error {
		if resp != nil {
			if resp.StatusCode >= http.StatusBadRequest {
				httpErr := &HttpError{
					Url:        url,
					StatusCode: resp.StatusCode,
				}
				defer resp.Body.Close()
				respErr, err := ioReadAll(resp.Body)
				if err != nil {
					httpErr.Err = errors.Wrap(err, "parsing response")
					return httpErr
				}
				httpErr.Body = string(respErr)
				httpErr.Err = receivedError
				return httpErr
			}
		}
		if receivedError != nil {
			return &HttpError{
				Url: url,
				Err: receivedError,
			}
		}
		return nil
	}
	decodeResponse = func(url string, resp *http.Response, v any) error {
		if v != nil {
			if resp != nil {
				defer resp.Body.Close()
				if err := jsonDecode(resp.Body, v); err != nil {
					return &HttpError{
						Url:        url,
						StatusCode: resp.StatusCode,
						Err:        errors.Wrap(err, "decoding response"),
					}
				}
			}
		}
		return nil
	}
	jsonDecode = func(r io.Reader, data any) error {
		return json.NewDecoder(r).Decode(data)
	}
	ioReadAll = func(r io.Reader) ([]byte, error) {
		return io.ReadAll(r)
	}
	castClientTransport = func(tr http.RoundTripper) (*http.Transport, bool) {
		transport, isTransport := tr.(*http.Transport)
		return transport, isTransport
	}
	dumpRequestOut = httputil.DumpRequestOut
)

// Client represents an http client.
type Client struct {
	httpClient          *http.Client
	retryableHttpClient *retryablehttp.Client
	timeout             time.Duration
	maxIdleConns        int
	maxIdleConnsPerHost int
	maxConnsPerHost     int
	maxRetries          int
	checkRetryPolicy    retryablehttp.CheckRetry
	retryWaitMin        time.Duration
	retryWaitMax        time.Duration
	requestDumpLogger   func(dump []byte)
	dumpRequestBody     bool
}

// doNotRetryPolicy is the default retry policy
// when a custom one is not provided, meaning that
// the request will not be retried.
// It is necessary because when a custom retry policy is
// not defined, `retryablehttp.DefaultRetryPolicy` becomes
// the default one, and it will always retry when status code >= 500,
// swalling the *http.Response.
func doNotRetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	return false, nil
}

// patchRetryableClient patches retryable http client.
func patchRetryableClient(client *Client) {
	client.retryableHttpClient.RetryMax = client.maxRetries
	client.retryableHttpClient.RetryWaitMin = client.retryWaitMin
	client.retryableHttpClient.RetryWaitMax = client.retryWaitMax
	client.retryableHttpClient.HTTPClient = client.httpClient
	// If no custom check retry policy is provided,
	// doNotRetryPolicy will be used.
	client.retryableHttpClient.CheckRetry = doNotRetryPolicy
	if client.checkRetryPolicy != nil {
		client.retryableHttpClient.CheckRetry = client.checkRetryPolicy
	}
}

// patchTransport patches the specified client with
// options for max idle connections, max idle connections per-host
// and max connections per-host.
func patchTransport(client *Client) {
	if client.httpClient.Transport == nil {
		dt := http.DefaultTransport.(*http.Transport).Clone()
		client.httpClient.Transport = dt
	}
	transport, isTransport := castClientTransport(client.httpClient.Transport)
	if !isTransport {
		// Custom RoundTripper.
		return
	}
	t := transport.Clone()
	t.MaxIdleConns = client.maxIdleConns
	t.MaxConnsPerHost = client.maxConnsPerHost
	t.MaxIdleConnsPerHost = client.maxIdleConnsPerHost
	client.httpClient.Transport = t
}

// newClient returns a new Client with options loaded.
func newClient(options []Option) *Client {
	client := new(Client)
	for _, option := range options {
		option(client)
	}
	return client
}

// New returns a new Client.
func New(options ...Option) *Client {
	client := newClient(options)
	client.retryableHttpClient = retryablehttp.NewClient()
	if client.httpClient == nil {
		client.httpClient = &http.Client{
			Timeout: client.timeout,
		}
	}
	patchTransport(client)
	patchRetryableClient(client)
	return client
}

// do performs a request and parses the response to the given interface, if provided.
func do(retryableHttpClient *retryablehttp.Client, req *retryablehttp.Request, v any) (*http.Response, error) {
	resp, err := retryableHttpClientDo(retryableHttpClient, req)
	if err := handleUnsuccessfulResponse(req.URL.String(), resp, err); err != nil {
		return resp, err
	}
	if err := decodeResponse(req.URL.String(), resp, v); err != nil {
		return resp, err
	}
	return resp, nil
}

// logRequestDump logs the request dump.
func (c *Client) logRequestDump(req *http.Request) {
	if c.requestDumpLogger != nil {
		dump, err := dumpRequestOut(req, c.dumpRequestBody)
		if err == nil {
			c.requestDumpLogger(dump)
		}
	}
}

// sendRequest sends a request with or without payload.
func (c *Client) sendRequest(req *http.Request, v any) (*http.Response, error) {
	c.logRequestDump(req)
	resp, err := do(c.retryableHttpClient, &retryablehttp.Request{Request: req}, v)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// SendRequest sends an HTTP request and returns an HTTP response.
func (c *Client) SendRequest(req *http.Request) (*http.Response, error) {
	return c.sendRequest(req, nil)
}

// SendRequestAndUnmarshallJsonResponse sends an HTTP request \
// and unmarshalls the responseBody to the given interface.
func (c *Client) SendRequestAndUnmarshallJsonResponse(req *http.Request, v any) (*http.Response, error) {
	return c.sendRequest(req, v)
}
