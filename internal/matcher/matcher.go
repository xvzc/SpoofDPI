package matcher

import (
	"fmt"
	"net"

	"github.com/xvzc/SpoofDPI/internal/config"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

// -----------------------------------------------------------------------------
// Enums & Interfaces
// -----------------------------------------------------------------------------

type MatchKind int

const (
	MatchKindDomain MatchKind = iota
	MatchKindAddr
)

func (k MatchKind) String() string {
	return []string{"domain", "addr"}[k]
}

type RuleMatcher interface {
	Add(r *config.Rule) error
	Search(s *Selector) *config.Rule
	SearchAll(ss []*Selector) *config.Rule
}

// -----------------------------------------------------------------------------
// Structs
// -----------------------------------------------------------------------------

type MatchAttrs struct {
	Domain   *string `json:"do,omitempty"`
	CIDR     *string `json:"ci,omitempty"`
	PortFrom *int    `json:"pf,omitempty"`
	PortTo   *int    `json:"pt,omitempty"`
}

type Selector struct {
	Kind   MatchKind
	Domain *string
	IP     *net.IP
	Port   *uint16
}

// -----------------------------------------------------------------------------
// RuleMatcher Implementation
// -----------------------------------------------------------------------------

type CompositeMatcher struct {
	addrMatcher   *AddrMatcher
	domainMatcher *DomainMatcher
}

func NewRuleMatcher(
	addrMatcher *AddrMatcher,
	domainMatcher *DomainMatcher,
) RuleMatcher {
	return &CompositeMatcher{
		addrMatcher:   addrMatcher,
		domainMatcher: domainMatcher,
	}
}

func (rs *CompositeMatcher) Add(r *config.Rule) error {
	if r.Match == nil {
		return fmt.Errorf("rule match attributes cannot be nil")
	}

	hasDomain := r.Match.Domain != nil
	hasCIDR := r.Match.CIDR != nil

	if !hasDomain && !hasCIDR {
		return fmt.Errorf("invalid rule: match must contain 'domain' or 'cidr'")
	}

	// A rule can be added to both matchers if it has both fields (rare but possible)

	// 1. Add to Domain Matcher
	if hasDomain {
		if rs.domainMatcher == nil {
			return fmt.Errorf("domain matcher not initialized")
		}
		if err := rs.domainMatcher.Add(r); err != nil {
			return err
		}
	}

	// 2. Add to Addr Matcher
	if hasCIDR {
		if rs.addrMatcher == nil {
			return fmt.Errorf("addr matcher not initialized")
		}
		// Assuming AddrMatcher handles PortFrom/PortTo internal defaults if nil
		if err := rs.addrMatcher.Add(r); err != nil {
			return err
		}
	}

	return nil
}

func (rs *CompositeMatcher) Search(s *Selector) *config.Rule {
	if s == nil {
		return nil
	}

	switch s.Kind {
	case MatchKindDomain:
		if rs.domainMatcher != nil {
			return rs.domainMatcher.Search(s)
		}
	case MatchKindAddr:
		if rs.addrMatcher != nil {
			return rs.addrMatcher.Search(s)
		}
	}

	return nil
}

func (rs *CompositeMatcher) SearchAll(ss []*Selector) *config.Rule {
	// 1. Get the best candidate from the AddrMatcher
	var bestAddrMatch *config.Rule
	if rs.addrMatcher != nil {
		bestAddrMatch = rs.addrMatcher.SearchAll(ss)
	}

	// 2. Get the best candidate from the Domain Matcher
	var bestDomainMatch *config.Rule
	if rs.domainMatcher != nil {
		bestDomainMatch = rs.domainMatcher.SearchAll(ss)
	}

	// 3. Compare and return the winner
	return GetHigherPriorityRule(bestAddrMatch, bestDomainMatch)
}

// GetHigherPriorityRule helper
func GetHigherPriorityRule(r1, r2 *config.Rule) *config.Rule {
	if r1 == nil {
		return r2
	}
	if r2 == nil {
		return r1
	}

	if ptr.FromPtr(r1.Priority) >= ptr.FromPtr(r2.Priority) {
		return r1
	}
	return r2
}
