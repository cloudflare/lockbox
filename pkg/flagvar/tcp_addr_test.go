package flagvar_test

import (
	"net"
	"testing"

	"github.com/cloudflare/lockbox/pkg/flagvar"
	"github.com/google/go-cmp/cmp"
)

func TestTCPAddrString(t *testing.T) {
	tests := []struct {
		name     string
		fv       *flagvar.TCPAddr
		expected string
	}{
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.fv.String(), tt.expected); diff != "" {
				t.Errorf("unexpected string returned: (+want -got)\n%s", diff)
			}
		})
	}
}

func TestTCPAddrSet(t *testing.T) {
	tests := []struct {
		name string
		text string
		addr *net.TCPAddr
		err  error
	}{
		{
			name: "host:port address",
			text: "127.0.0.1:8080",
			addr: &net.TCPAddr{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 8080,
			},
		},
		{
			name: "port-only address",
			text: ":8080",
			addr: &net.TCPAddr{
				IP:   net.ParseIP("0.0.0.0"),
				Port: 8080,
			},
		},
		{
			name: "IPv6 support",
			text: "[::1]:8080",
			addr: &net.TCPAddr{
				IP:   net.ParseIP("[::1]"),
				Port: 8080,
			},
		},
		{
			name: "invalid address",
			text: "google.com",
			addr: nil,
			err: &net.AddrError{
				Err:  "missing port in address",
				Addr: "google.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fv := &flagvar.TCPAddr{}

			if diff := cmp.Diff(fv.Set(tt.text), tt.err); diff != "" {
				t.Errorf("unexpected error returned: (+want -got)\n%s", diff)
			}

			if diff := cmp.Diff(fv.String(), tt.text); diff != "" {
				t.Errorf("unexpected string returned: (+want -got)\n%s", diff)
			}
		})
	}
}
