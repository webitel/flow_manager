package resolver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseURL(t *testing.T) {
	testCases := []struct {
		name           string
		url            string
		expected       target
		expectErr      bool
		expectedErrMsg string
	}{
		{
			name: "Valid URL with all params",
			url:  "wbt://user:pass@consul:8500/my-service?healthy=true&tag=v1&wait=5s&dc=dc1",
			expected: target{
				Addr:       "consul:8500",
				User:       "user",
				Password:   "pass",
				Service:    "my-service",
				Wait:       5 * time.Second,
				Tag:        "v1",
				Healthy:    true,
				Dc:         "dc1",
				Near:       "_agent",    // Default value
				MaxBackoff: time.Second, // Default value
			},
			expectErr: false,
		},
		{
			name: "Valid URL minimal",
			url:  "wbt://consul:8500/my-service",
			expected: target{
				Addr:       "consul:8500",
				Service:    "my-service",
				Near:       "_agent",
				MaxBackoff: time.Second,
			},
			expectErr: false,
		},
		{
			name:           "Malformed URL - wrong scheme",
			url:            "http://consul:8500/my-service",
			expectErr:      true,
			expectedErrMsg: "Malformed URL('http://consul:8500/my-service'). Must be in the next format: 'consul://[user:passwd]@host/service?param=value'",
		},
		{
			name:      "Malformed URL - no service",
			url:       "wbt://consul:8500/",
			expectErr: true,
		},
		{
			name:      "Malformed URL - bad duration param",
			url:       "wbt://consul:8500/my-service?wait=5seconds",
			expectErr: true,
			// UPDATED: This now matches the error from the 'form' decoder library.
			expectedErrMsg: "time: unknown unit \"seconds\" in duration \"5seconds\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tgt, err := parseURL(tc.url)
			if tc.expectErr {
				require.Error(t, err)

				if tc.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tc.expectedErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, tgt)
			}
		})
	}
}
