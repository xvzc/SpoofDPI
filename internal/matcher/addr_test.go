package matcher

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xvzc/spoofdpi/internal/config"
)

func TestAddrMatcher(t *testing.T) {
	matcher := NewAddrMatcher()

	rule1 := &config.Rule{
		Name:     "rule1",
		Priority: uint16(10),
		Match: &config.MatchAttrs{
			Addrs: []config.AddrMatch{
				{
					CIDR:     config.MustParseCIDR("192.168.1.0/24"),
					PortFrom: uint16(80),
					PortTo:   uint16(80),
				},
			},
		},
	}

	rule2 := &config.Rule{
		Name:     "rule2",
		Priority: uint16(20),
		Match: &config.MatchAttrs{
			Addrs: []config.AddrMatch{
				{
					CIDR:     config.MustParseCIDR("10.0.0.0/8"),
					PortFrom: uint16(0),
					PortTo:   uint16(65535),
				},
			},
		},
	}

	// Overlapping lower priority rule
	rule3 := &config.Rule{
		Name:     "rule3",
		Priority: uint16(5),
		Match: &config.MatchAttrs{
			Addrs: []config.AddrMatch{
				{
					CIDR:     config.MustParseCIDR("172.16.0.0/16"),
					PortFrom: uint16(0),
					PortTo:   uint16(65535),
				},
			},
		},
	}

	// Overlapping lower priority rule
	rule4 := &config.Rule{
		Name:     "rule4",
		Priority: uint16(4),
		Match: &config.MatchAttrs{
			Addrs: []config.AddrMatch{
				{
					CIDR:     config.MustParseCIDR("172.16.0.0/16"),
					PortFrom: uint16(443),
					PortTo:   uint16(443),
				},
			},
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
				assert.Equal(t, "rule1", output.Name)
			},
		},
		{
			name: "match rule2 on 8080",
			ip:   "10.0.0.5",
			port: 8080,
			assert: func(t *testing.T, output *config.Rule) {
				// Should still match rule2 because priority 20 > 5
				assert.NotNil(t, output)
				assert.Equal(t, "rule2", output.Name)
			},
		},
		{
			name: "match rule2 on 443",
			ip:   "10.0.0.5",
			port: 443,
			assert: func(t *testing.T, output *config.Rule) {
				// Should still match rule2 because priority 20 > 5
				assert.NotNil(t, output)
				assert.Equal(t, "rule2", output.Name)
			},
		},
		{
			name: "match rule3 (higher priority check)",
			ip:   "172.16.0.5",
			port: 443,
			assert: func(t *testing.T, output *config.Rule) {
				// Should still match rule2 because priority 20 > 5
				assert.NotNil(t, output)
				assert.Equal(t, "rule3", output.Name)
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
			p := uint16(port)
			selector := &Selector{IP: &ip, Port: &p}
			output := matcher.Search(selector)
			tc.assert(t, output)
		})
	}
}
