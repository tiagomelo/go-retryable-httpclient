//go:build integration

package test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tiagomelo/go-retryable-httpclient/httpclient"
)

var client *httpclient.Client

type HttpBinResponse struct {
	Headers map[string]string `json:"headers"`
	Data    string            `json:"data"`
}

type DummyPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func logRequestDump(dump []byte) {
	fmt.Print("request sent:\n\n")
	fmt.Println(string(dump))
}

func TestMain(m *testing.M) {
	client = httpclient.New(httpclient.WithRequestDumpLogger(logRequestDump, true))
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestSendRequest(t *testing.T) {
	const url = "http://localhost/get"
	req, err := httpclient.NewRequest(context.TODO(), http.MethodGet, url)
	if err != nil {
		t.Fatalf(`creating request for "%v": %v`, url, err)
	}
	resp, err := client.SendRequest(req)
	if err != nil {
		t.Fatalf(`making request for "%v": %v`, url, err)
	}
	require.Equal(t, resp.StatusCode, http.StatusOK)
}

func TestSendRequestWithHeadersAndUnmarshallJsonResponse(t *testing.T) {
	const url = "http://localhost/get"
	var httpBinResponse HttpBinResponse
	expectedResponse := HttpBinResponse{
		Headers: map[string]string{
			"Accept-Encoding": "gzip",
			"Custom-Header-1": "some value",
			"Custom-Header-2": "some other value",
			"Host":            "localhost",
			"User-Agent":      "Go-http-client/1.1",
		},
	}
	headers := map[string]string{
		"Custom-Header-1": "some value",
		"Custom-Header-2": "some other value",
	}
	req, err := httpclient.NewRequestWithHeaders(context.TODO(), http.MethodGet, url, headers)
	if err != nil {
		t.Fatalf(`creating request for "%v": %v`, url, err)
	}
	resp, err := client.SendRequestAndUnmarshallJsonResponse(req, &httpBinResponse)
	if err != nil {
		t.Fatalf(`making request for "%v": %v`, url, err)
	}
	require.Equal(t, resp.StatusCode, http.StatusOK)
	require.Equal(t, expectedResponse, httpBinResponse)
}

func TestSendJsonRequestAndUnmarshallJsonResponse(t *testing.T) {
	const url = "http://localhost/post"
	var httpBinResponse HttpBinResponse
	expectedResponse := HttpBinResponse{
		Headers: map[string]string{
			"Accept-Encoding": "gzip",
			"Content-Length":  "55",
			"Content-Type":    "application/json",
			"Host":            "localhost",
			"User-Agent":      "Go-http-client/1.1",
		},
		Data: "{\"name\":\"Steve Harris\",\"email\":\"steve@ironmaiden.com\"}\n",
	}
	dummyPayload := &DummyPayload{
		Name:  "Steve Harris",
		Email: "steve@ironmaiden.com",
	}
	req, err := httpclient.NewJsonRequest(context.TODO(), http.MethodPost, url, dummyPayload)
	if err != nil {
		t.Fatalf(`creating request for "%v": %v`, url, err)
	}
	resp, err := client.SendRequestAndUnmarshallJsonResponse(req, &httpBinResponse)
	if err != nil {
		t.Fatalf(`making request for "%v": %v`, url, err)
	}
	require.Equal(t, resp.StatusCode, http.StatusOK)
	require.Equal(t, expectedResponse, httpBinResponse)
}

func TestSendJsonRequestWithHeadersAndUnmarshallJsonResponse(t *testing.T) {
	const url = "http://localhost/post"
	var httpBinResponse HttpBinResponse
	expectedResponse := HttpBinResponse{
		Headers: map[string]string{
			"Accept-Encoding": "gzip",
			"Content-Length":  "55",
			"Content-Type":    "application/json",
			"Custom-Header-1": "some value",
			"Custom-Header-2": "some other value",
			"Host":            "localhost",
			"User-Agent":      "Go-http-client/1.1",
		},
		Data: "{\"name\":\"Steve Harris\",\"email\":\"steve@ironmaiden.com\"}\n",
	}
	headers := map[string]string{
		"Custom-Header-1": "some value",
		"Custom-Header-2": "some other value",
	}
	dummyPayload := &DummyPayload{
		Name:  "Steve Harris",
		Email: "steve@ironmaiden.com",
	}
	req, err := httpclient.NewJsonRequestWithHeaders(context.TODO(),
		http.MethodPost, url, dummyPayload, headers)
	if err != nil {
		t.Fatalf(`creating request for "%v": %v`, url, err)
	}
	resp, err := client.SendRequestAndUnmarshallJsonResponse(req, &httpBinResponse)
	if err != nil {
		t.Fatalf(`making request for "%v": %v`, url, err)
	}
	require.Equal(t, resp.StatusCode, http.StatusOK)
	require.Equal(t, expectedResponse, httpBinResponse)
}
