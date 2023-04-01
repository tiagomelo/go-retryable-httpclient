package httpclient

import (
	"fmt"
	"net/http"
)

// AddAuthorizationBearerHeaderToRequest adds bearer authorization header
// to request.
func AddAuthorizationBearerHeaderToRequest(req *http.Request, token string) {
	const authHeader = "Authorization"
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req.Header.Set(authHeader, fmt.Sprintf("Bearer %s", token))
}
