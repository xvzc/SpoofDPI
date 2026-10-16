package executil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mvdan.cc/sh/shell"
)

func TestQuote_RoundTrip(t *testing.T) {
	tcs := []struct {
		name string
		in   string
	}{
		{"plain", "Wi-Fi"},
		{"whitespace", "USB 10/100/1000 LAN"},
		{"embedded apostrophe", "Bob's Wi-Fi"},
		{"dollar", "$HOME"},
		{"glob", "*"},
		{"non-ascii", "Wi-Fi 日本語"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			fields, err := shell.Fields(Quote(tc.in), nil)
			require.NoError(t, err)
			assert.Equal(t, []string{tc.in}, fields)
		})
	}
}
