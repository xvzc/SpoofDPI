package matcher

import (
	"fmt"
	"net"
	"sort"
	"sync"

	"github.com/xvzc/spoofdpi/internal/config"
)

// AddrMatcher implements Matcher for IP/CIDR rules.
type AddrMatcher struct {
	mu    sync.RWMutex
	rules []cidrRule
}

// cidrRule holds the parsed CIDR and the pointer to the original Rule.
type cidrRule struct {
	cidr     net.IPNet
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

	if len(r.Match.Addrs) == 0 {
		return fmt.Errorf("addr rule must have addr attribute")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, addr := range r.Match.Addrs {
		if len(addr.CIDR.IP) == 0 {
			return fmt.Errorf("addr rule must have cidr attribute")
		}

		if addr.PortFrom == 0 && addr.PortTo == 0 {
			return fmt.Errorf("addr rule must have port-from, port-to attribute")
		}

		// 4. Create internal rule
		cr := cidrRule{
			cidr:     addr.CIDR,
			portFrom: addr.PortFrom,
			portTo:   addr.PortTo,
			rule:     r,
		}

		// 5. Append
		m.rules = append(m.rules, cr)
	}

	// Sort (Priority Descending)
	sort.SliceStable(m.rules, func(i, j int) bool {
		return m.rules[i].rule.Priority > m.rules[j].rule.Priority
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
