package httpclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddAuthorizationBearerHeaderToRequest(t *testing.T) {
	testCases := []struct {
		name            string
		req             *http.Request
		expectedHeaders http.Header
	}{
		{
			name: "without existing headers",
			req:  &http.Request{},
			expectedHeaders: http.Header{
				"Authorization": {"Bearer sometoken"},
			},
		},
		{
			name: "with existing headers",
			req: &http.Request{
				Header: http.Header{
					"CustomHeader": {"blablabla"},
				},
			},
			expectedHeaders: http.Header{
				"Authorization": {"Bearer sometoken"},
				"CustomHeader":  {"blablabla"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			AddAuthorizationBearerHeaderToRequest(tc.req, "sometoken")
			require.Equal(t, tc.expectedHeaders, tc.req.Header)
		})
	}
}
