package flagvar_test

import (
	"testing"

	"github.com/cloudflare/lockbox/pkg/flagvar"
	"gotest.tools/v3/assert"
)

func TestEnumString(t *testing.T) {
	type testCase struct {
		name     string
		fv       *flagvar.Enum
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		actual := tc.fv.String()
		assert.Equal(t, actual, tc.expected)
	}

	testCases := []testCase{
		{
			name:     "non-nil receiver",
			fv:       &flagvar.Enum{Value: "yaml"},
			expected: "yaml",
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

func TestEnumSet(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		expected string
		err      error
	}

	run := func(t *testing.T, tc testCase) {
		fv := &flagvar.Enum{
			Choices: []string{"yaml", "json"},
		}
		err := fv.Set(tc.input)

		if err != nil {
			assert.ErrorIs(t, err, tc.err)
		} else {
			assert.Equal(t, fv.Value, tc.expected)
		}
	}

	testCases := []testCase{
		{
			name:     "valid enum option",
			input:    "yaml",
			expected: "yaml",
		},
		{
			name:     "ignores option capitalization",
			input:    "YaMl",
			expected: "yaml",
		},
		{
			name:  "invalid enum option",
			input: "cue",
			err:   flagvar.ErrInvalidEnum,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
