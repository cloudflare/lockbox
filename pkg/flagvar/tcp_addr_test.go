package flagvar_test

import (
	"net"
	"testing"

	"github.com/cloudflare/lockbox/pkg/flagvar"
	"gotest.tools/v3/assert"
)

func TestTCPAddrString(t *testing.T) {
	type testCase struct {
		name     string
		fv       *flagvar.TCPAddr
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.fv.String()
		assert.Equal(t, actual, tc.expected)
	}

	testCases := []testCase{
		{
			name:     "non-nil receiver",
			fv:       &flagvar.TCPAddr{Text: ":8080"},
			expected: ":8080",
		},
		{
			name:     "nil receiver",
			fv:       nil,
			expected: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestTCPAddrSet(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		expected *net.TCPAddr
		err      string
	}

	run := func(t *testing.T, tc testCase) {
		fv := flagvar.TCPAddr{}
		err := fv.Set(tc.input)

		if err != nil {
			assert.Error(t, err, tc.err)
		} else {
			assert.DeepEqual(t, fv.Value, tc.expected)
		}
	}

	testCases := []testCase{
		{
			name:  "host:port address",
			input: "127.0.0.1:8080",
			expected: &net.TCPAddr{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 8080,
			},
		},
		{
			name:  "port-only address",
			input: ":8080",
			expected: &net.TCPAddr{
				Port: 8080,
			},
		},
		{
			name:  "IPv6 support",
			input: "[::1]:8080",
			expected: &net.TCPAddr{
				IP:   net.ParseIP("::1"),
				Port: 8080,
			},
		},
		{
			name:     "invalid address",
			input:    "google.com",
			expected: nil,
			err:      "address google.com: missing port in address",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
