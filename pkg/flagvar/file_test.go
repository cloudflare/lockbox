package flagvar_test

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/cloudflare/lockbox/pkg/flagvar"
	"gotest.tools/v3/assert"
)

func TestFileString(t *testing.T) {
	type testCase struct {
		name     string
		fv       *flagvar.File
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.fv.String()
		assert.Equal(t, actual, tc.expected)
	}

	testCases := []testCase{
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

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestFileSet(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		expected string
		err      error
	}

	run := func(t *testing.T, tc testCase) {
		fv := &flagvar.File{}

		err := fv.Set(tc.input)
		if tc.err != nil {
			assert.ErrorIs(t, err, tc.err)
		} else {
			assert.Equal(t, fv.Value, tc.expected)
		}
	}

	testCases := []testCase{
		{
			name:     "file exists",
			input:    filepath.Join("testdata", "file"),
			expected: "testdata/file",
		},
		{
			name:  "file does not exist",
			input: filepath.Join("testdata", "file_nonexistant.go"),
			err:   fs.ErrNotExist,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
