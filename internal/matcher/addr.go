package matcher

import (
	"fmt"
	"net"
	"sort"
	"sync"

	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

// AddrMatcher implements Matcher for IP/CIDR rules.
type AddrMatcher struct {
	mu    sync.RWMutex
	rules []cidrRule
}

// cidrRule holds the parsed CIDR and the pointer to the original Rule.
// [Modified]: Changed 'attrs' to 'rule' to satisfy the Search return type (*Rule).
type cidrRule struct {
	cidr     *net.IPNet
	portFrom uint16
	portTo   uint16
	rule     *config.Rule
}

func NewAddrMatcher() *AddrMatcher {
	return &AddrMatcher{
		rules: make([]cidrRule, 0),
	}
}

// Add parses the rule and inserts it into the list.
// It keeps the list sorted by Priority (Descending) to optimize Search.
func (m *AddrMatcher) Add(r *config.Rule) error {
	if r.Match == nil {
		return fmt.Errorf("addr rule must have match attribute")
	}

	if r.Match.CIDR == nil {
		return fmt.Errorf("addr rule must have cidr attribute")
	}

	if r.Match.PortFrom == nil || r.Match.PortTo == nil {
		return fmt.Errorf("addr rule must have port-from, port-to attribute")
	}

	if r.Priority == nil {
		r.Priority = ptr.FromValue(uint16(0))
	}

	if r.Block == nil {
		r.Block = ptr.FromValue(false)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 4. Create internal rule
	cr := cidrRule{
		cidr:     r.Match.CIDR,
		portFrom: *r.Match.PortFrom,
		portTo:   *r.Match.PortTo,
		rule:     r,
	}

	// 5. Append and Sort (Priority Descending)
	m.rules = append(m.rules, cr)

	sort.SliceStable(m.rules, func(i, j int) bool {
		p1 := uint16(0)
		if m.rules[i].rule.Priority != nil {
			p1 = *m.rules[i].rule.Priority
		}

		p2 := uint16(0)
		if m.rules[j].rule.Priority != nil {
			p2 = *m.rules[j].rule.Priority
		}
		return p1 > p2
	})

	return nil
}

// Search finds the highest priority rule matching the selector.
func (m *AddrMatcher) Search(s *Selector) *config.Rule {
	if s.IP == nil {
		return nil
	}

	// Parse IP from Selector
	if s.IP == nil || s.Port == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Linear search is O(N), but since the list is sorted by Priority,
	// the first match is guaranteed to be the highest priority one.
	for _, cr := range m.rules {
		// 1. Check IP containment
		if !cr.cidr.Contains(*s.IP) {
			continue
		}

		// 2. Check Port range
		if *s.Port < cr.portFrom || *s.Port > cr.portTo {
			continue
		}

		// Match Found
		return cr.rule.Clone()
	}

	return nil
}

// SearchAll finds the highest priority rule among multiple selectors.
func (m *AddrMatcher) SearchAll(ss []*Selector) *config.Rule {
	var bestRule *config.Rule

	for _, s := range ss {
		rule := m.Search(s)
		if rule == nil {
			continue
		}

		bestRule = GetHigherPriorityRule(bestRule, rule)
	}

	if bestRule != nil {
		return bestRule.Clone()
	}

	return nil
}
