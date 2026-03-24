package matcher

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/xvzc/SpoofDPI/internal/config"
)

func TestDomainMatcher(t *testing.T) {
	// Shared setup for table-driven tests
	matcher := NewDomainMatcher()

	rule1 := &config.Rule{
		Name:     lo.ToPtr("rule1"),
		Priority: lo.ToPtr(uint16(10)),
		Match: &config.MatchAttrs{
			Domains: []string{"example.com"},
		},
	}

	rule2 := &config.Rule{
		Name:     lo.ToPtr("rule2"),
		Priority: lo.ToPtr(uint16(20)),
		Match: &config.MatchAttrs{
			Domains: []string{"*.google.com"},
		},
	}

	rule3 := &config.Rule{
		Name:     lo.ToPtr("rule3"),
		Priority: lo.ToPtr(uint16(5)),
		Match: &config.MatchAttrs{
			Domains: []string{"**.youtube.com"},
		},
	}

	// Additional rule for priority check
	rule4 := &config.Rule{
		Name:     lo.ToPtr("rule4"),
		Priority: lo.ToPtr(uint16(30)),
		Match: &config.MatchAttrs{
			Domains: []string{"mail.google.com"},
		},
	}

	assert.NoError(t, matcher.Add(rule1))
	assert.NoError(t, matcher.Add(rule2))
	assert.NoError(t, matcher.Add(rule3))
	assert.NoError(t, matcher.Add(rule4))

	tcs := []struct {
		name     string
		selector *Selector
		assert   func(t *testing.T, output *config.Rule)
	}{
		{
			name:     "exact match",
			selector: &Selector{Domain: lo.ToPtr("example.com")},
			assert: func(t *testing.T, output *config.Rule) {
				assert.NotNil(t, output)
				assert.Equal(t, "rule1", *output.Name)
			},
		},
		{
			name:     "wildcard match",
			selector: &Selector{Domain: lo.ToPtr("maps.google.com")},
			assert: func(t *testing.T, output *config.Rule) {
				assert.NotNil(t, output)
				assert.Equal(t, "rule2", *output.Name)
			},
		},
		{
			name:     "globstar match",
			selector: &Selector{Domain: lo.ToPtr("foo.bar.youtube.com")},
			assert: func(t *testing.T, output *config.Rule) {
				assert.NotNil(t, output)
				assert.Equal(t, "rule3", *output.Name)
			},
		},
		{
			name:     "wildcard higher priority check",
			selector: &Selector{Domain: lo.ToPtr("mail.google.com")},
			assert: func(t *testing.T, output *config.Rule) {
				// Should pick rule4 (priority 30) over rule2 (priority 20)
				assert.NotNil(t, output)
				assert.Equal(t, "rule4", *output.Name)
			},
		},
		{
			name:     "no match",
			selector: &Selector{Domain: lo.ToPtr("naver.com")},
			assert: func(t *testing.T, output *config.Rule) {
				assert.Nil(t, output)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := matcher.Search(tc.selector)
			tc.assert(t, output)
		})
	}
}
