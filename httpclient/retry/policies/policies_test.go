package policies

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDoNoRetry(t *testing.T) {
	testCases := []struct {
		name string
		err  error
	}{
		{
			name: "without provided error",
		},
		{
			name: "with provided error",
			err:  errors.New("random error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			retry, err := DoNotRetry(context.TODO(), new(http.Response), tc.err)
			require.False(t, retry)
			require.Nil(t, err)
		})
	}
}

func TestEof(t *testing.T) {
	testCases := []struct {
		name           string
		err            error
		expectedOutput bool
	}{
		{
			name: "without provided error",
		},
		{
			name:           "with EOF error",
			err:            errors.New("blablabla EOF blablabla"),
			expectedOutput: true,
		},
		{
			name: "with different error",
			err:  errors.New("blablabla"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			retry, err := Eof(context.TODO(), new(http.Response), tc.err)
			require.Equal(t, tc.expectedOutput, retry)
			require.Equal(t, tc.err, err)
		})
	}
}
