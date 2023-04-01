package httpclient

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIs(t *testing.T) {
	testCases := []struct {
		name           string
		httpError      *HttpError
		targetError    error
		expectedOutput bool
	}{
		{
			name:      "target error is nil",
			httpError: &HttpError{},
		},
		{
			name:        "target error is not of type HttpError",
			httpError:   &HttpError{},
			targetError: errors.New("random error"),
		},
		{
			name: "target error with same status code",
			httpError: &HttpError{
				StatusCode: http.StatusBadRequest,
			},
			targetError: &HttpError{
				StatusCode: http.StatusBadRequest,
			},
			expectedOutput: true,
		},
		{
			name: "target error with different status code",
			httpError: &HttpError{
				StatusCode: http.StatusBadRequest,
			},
			targetError: &HttpError{
				StatusCode: http.StatusInternalServerError,
			},
		},
		{
			name: "target error without status code",
			httpError: &HttpError{
				StatusCode: http.StatusBadRequest,
			},
			targetError:    &HttpError{},
			expectedOutput: true,
		},
		{
			name: "target error with same body",
			httpError: &HttpError{
				Body: "some body",
			},
			targetError: &HttpError{
				Body: "some body",
			},
			expectedOutput: true,
		},
		{
			name: "target error with empty body",
			httpError: &HttpError{
				Body: "some body",
			},
			targetError:    &HttpError{},
			expectedOutput: true,
		},
		{
			name: "target error with different body",
			httpError: &HttpError{
				Body: "some body",
			},
			targetError: &HttpError{
				Body: "another body",
			},
		},
		{
			name: "target error with same err",
			httpError: &HttpError{
				Err: http.ErrAbortHandler,
			},
			targetError: &HttpError{
				Err: http.ErrAbortHandler,
			},
			expectedOutput: true,
		},
		{
			name: "target error with different err",
			httpError: &HttpError{
				Err: http.ErrBodyNotAllowed,
			},
			targetError: &HttpError{
				Err: http.ErrAbortHandler,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			is := tc.httpError.Is(tc.targetError)
			require.Equal(t, tc.expectedOutput, is)
		})
	}
}

func checkIfErrorIsExpected(t *testing.T, err, expectedError error) {
	if expectedError == nil {
		t.Fatalf(`expected no error, got "%v"`, err)
	}
}

func checkIfErrorIsNotExpected(t *testing.T, err, expectedError error) {
	if expectedError != nil {
		t.Fatalf(`expected error "%v", got nil`, expectedError.Error())
	}
}
