package matcher

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

func TestAddrMatcher(t *testing.T) {
	matcher := NewAddrMatcher()

	rule1 := &config.Rule{
		Name:     ptr.FromValue("rule1"),
		Priority: ptr.FromValue(uint16(10)),
		Match: &config.MatchAttrs{
			CIDR:     ptr.FromValue(config.MustParseCIDR("192.168.1.0/24")),
			PortFrom: ptr.FromValue(uint16(80)),
			PortTo:   ptr.FromValue(uint16(80)),
		},
	}

	rule2 := &config.Rule{
		Name:     ptr.FromValue("rule2"),
		Priority: ptr.FromValue(uint16(20)),
		Match: &config.MatchAttrs{
			CIDR:     ptr.FromValue(config.MustParseCIDR("10.0.0.0/8")),
			PortFrom: ptr.FromValue(uint16(0)),
			PortTo:   ptr.FromValue(uint16(65535)),
		},
	}

	// Overlapping lower priority rule
	rule3 := &config.Rule{
		Name:     ptr.FromValue("rule3"),
		Priority: ptr.FromValue(uint16(5)),
		Match: &config.MatchAttrs{
			CIDR:     ptr.FromValue(config.MustParseCIDR("172.16.0.0/16")),
			PortFrom: ptr.FromValue(uint16(0)),
			PortTo:   ptr.FromValue(uint16(65535)),
		},
	}

	// Overlapping lower priority rule
	rule4 := &config.Rule{
		Name:     ptr.FromValue("rule4"),
		Priority: ptr.FromValue(uint16(4)),
		Match: &config.MatchAttrs{
			CIDR:     ptr.FromValue(config.MustParseCIDR("172.16.0.0/16")),
			PortFrom: ptr.FromValue(uint16(443)),
			PortTo:   ptr.FromValue(uint16(443)),
		},
	}

	assert.NoError(t, matcher.Add(rule1))
	assert.NoError(t, matcher.Add(rule2))
	assert.NoError(t, matcher.Add(rule3))
	assert.NoError(t, matcher.Add(rule4))

	tcs := []struct {
		name   string
		ip     string
		port   int
		assert func(t *testing.T, output *config.Rule)
	}{
		{
			name: "match rule1",
			ip:   "192.168.1.10",
			port: 80,
			assert: func(t *testing.T, output *config.Rule) {
				assert.NotNil(t, output)
				assert.Equal(t, "rule1", *output.Name)
			},
		},
		{
			name: "match rule2 on 8080",
			ip:   "10.0.0.5",
			port: 8080,
			assert: func(t *testing.T, output *config.Rule) {
				// Should still match rule2 because priority 20 > 5
				assert.NotNil(t, output)
				assert.Equal(t, "rule2", *output.Name)
			},
		},
		{
			name: "match rule2 on 443",
			ip:   "10.0.0.5",
			port: 443,
			assert: func(t *testing.T, output *config.Rule) {
				// Should still match rule2 because priority 20 > 5
				assert.NotNil(t, output)
				assert.Equal(t, "rule2", *output.Name)
			},
		},
		{
			name: "match rule3 (higher priority check)",
			ip:   "172.16.0.5",
			port: 443,
			assert: func(t *testing.T, output *config.Rule) {
				// Should still match rule2 because priority 20 > 5
				assert.NotNil(t, output)
				assert.Equal(t, "rule3", *output.Name)
			},
		},
		{
			name: "no match port",
			ip:   "192.168.1.10",
			port: 443,
			assert: func(t *testing.T, output *config.Rule) {
				assert.Nil(t, output)
			},
		},
		{
			name: "no match ip",
			ip:   "172.128.0.1",
			port: 80,
			assert: func(t *testing.T, output *config.Rule) {
				assert.Nil(t, output)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			port := tc.port
			selector := &Selector{IP: &ip, Port: ptr.FromValue(uint16(port))}
			output := matcher.Search(selector)
			tc.assert(t, output)
		})
	}
}
