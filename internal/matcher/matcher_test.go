package matcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

func TestGetHigherPriorityRule(t *testing.T) {
	r1 := &config.Rule{Priority: ptr.FromValue(uint16(10))}
	r2 := &config.Rule{Priority: ptr.FromValue(uint16(20))}

	tcs := []struct {
		name   string
		r1     *config.Rule
		r2     *config.Rule
		expect *config.Rule
	}{
		{"r1 vs r2", r1, r2, r2},
		{"r2 vs r1", r2, r1, r2},
		{"r1 vs nil", r1, nil, r1},
		{"nil vs r2", nil, r2, r2},
		{"nil vs nil", nil, nil, nil},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := GetHigherPriorityRule(tc.r1, tc.r2)
			assert.Equal(t, tc.expect, output)
		})
	}
}
