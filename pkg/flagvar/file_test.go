package flagvar_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudflare/lockbox/pkg/flagvar"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFileString(t *testing.T) {
	tests := []struct {
		name     string
		fv       *flagvar.File
		expected string
	}{
		{
			name:     "non-nil receiver",
			fv:       &flagvar.File{Value: "/path/to/default.log"},
			expected: "/path/to/default.log",
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

func TestFileSet(t *testing.T) {
	tests := []struct {
		name string
		path string
		err  error
	}{
		{
			name: "file exists",
			path: filepath.Join("testdata", "file"),
		},
		{
			name: "file does not exist",
			path: filepath.Join("testdata", "file_nonexistant.go"),
			err: &os.PathError{
				Op:   "stat",
				Path: filepath.Join("testdata", "file_nonexistant.go"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fv := &flagvar.File{}

			opts := []cmp.Option{
				cmpopts.IgnoreFields(os.PathError{}, "Err"),
			}

			if diff := cmp.Diff(fv.Set(tt.path), tt.err, opts...); diff != "" {
				t.Errorf("unexpected error returned: (+want -got)\n%s", diff)
			}

			if diff := cmp.Diff(fv.String(), tt.path); diff != "" {
				t.Errorf("unexpected string returned: (+want -got)\n%s", diff)
			}
		})
	}
}
