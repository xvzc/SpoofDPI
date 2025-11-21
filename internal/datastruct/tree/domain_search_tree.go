package tree

import (
	"strings"
	"sync"
)

var _ SearchTree = (*domainSearchTree)(nil)

// domainNode represents a single node in the radix tree implementation.
type domainNode struct {
	// children holds static sub-segments (e.g., "www", "api").
	children map[string]*domainNode

	// wildcardChild holds the node for a '*' segment (matches one segment).
	wildcardChild *domainNode

	// globstarChild holds the node for a '**' segment (matches all subsequent segments).
	globstarChild *domainNode

	// value stores the data associated with the *exact* domain
	// that terminates at this node.
	value any

	// hasValue indicates whether a value has been set on this node.
	// This is crucial to differentiate a set zero value (like false, 0, or "")
	// from a node with no value set at all (nil).
	hasValue bool
}

// newDomainNode creates a new Node, initializing its children map.
func newDomainNode() *domainNode {
	return &domainNode{
		children: make(map[string]*domainNode),
		// hasValue is false by default
		// value is nil by default
	}
}

// domainSearchTree is the concrete implementation of the DomainTree interface.
type domainSearchTree struct {
	// Mutex protects the entire tree structure during write operations (Insert).
	// Read operations (Search) can safely run concurrently.
	mu   sync.RWMutex
	root *domainNode
}

// NewDomainSearchTree creates a new, empty domain tree.
func NewDomainSearchTree() SearchTree {
	return &domainSearchTree{root: newDomainNode()}
}

// reverseSegments reverses a slice of strings in place.
// This is a helper function.
func reverseSegments(segments []string) {
	for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
		segments[i], segments[j] = segments[j], segments[i]
	}
}

// splitAndReverseDomain splits a domain by '.' and reverses the segments.
// This is a helper function.
func splitAndReverseDomain(domain string) []string {
	if domain == "" {
		return []string{}
	}
	segments := strings.Split(domain, ".")
	reverseSegments(segments)
	return segments
}

// Insert adds a domain and its associated value to the tree.
// This method is thread-safe using a Write Lock.
func (t *domainSearchTree) Insert(domain string, value any) {
	// Acquire Write Lock to prevent concurrent map access during tree modification
	t.mu.Lock()
	defer t.mu.Unlock()

	segments := splitAndReverseDomain(domain)
	node := t.root

	for _, segment := range segments {
		switch segment {
		case "*":
			// Handle wildcard segment
			if node.wildcardChild == nil {
				node.wildcardChild = newDomainNode()
			}
			node = node.wildcardChild
		case "**":
			// Handle globstar segment (e.g., "**.example.com")
			if node.globstarChild == nil {
				node.globstarChild = newDomainNode()
			}
			node = node.globstarChild
		default:
			// Handle static segment
			if _, ok := node.children[segment]; !ok {
				node.children[segment] = newDomainNode()
			}
			node = node.children[segment]
		}
	}

	// Set the value and mark this node as having a value
	node.value = value
	node.hasValue = true
}

// Search searches for the domain in the tree.
// This method is concurrently safe using a Read Lock.
func (t *domainSearchTree) Search(domain string) (any, bool) {
	// Acquire Read Lock to allow multiple concurrent readers.
	t.mu.RLock()
	defer t.mu.RUnlock()

	segments := splitAndReverseDomain(domain)
	// Start the recursive lookup from the root node.
	return t.lookupRecursive(t.root, segments)
}

// lookupRecursive performs a depth-first search with backtracking to find the domain.
func (t *domainSearchTree) lookupRecursive(
	node *domainNode,
	segments []string,
) (any, bool) {
	// If no segments are left, we are at the target node.
	if len(segments) == 0 {
		// Check for a specific value at this node.
		if node.hasValue {
			return node.value, true
		}
		// Check for a root domain wildcard match (e.g., "*.net" for "net")
		if node.wildcardChild != nil && node.wildcardChild.hasValue {
			return node.wildcardChild.value, true
		}

		// Check for a root domain globstar match (e.g., "**.net" for "net")
		if node.globstarChild != nil && node.globstarChild.hasValue {
			return node.globstarChild.value, true
		}

		// No match.
		return nil, false
	}

	segment := segments[0]
	remainingSegments := segments[1:]

	// Priority 1: Check static match.
	if child, ok := node.children[segment]; ok {
		// Found a static path, recurse.
		val, found := t.lookupRecursive(child, remainingSegments)
		// If this path leads to a match, return it immediately.
		if found {
			return val, true
		}
		// If this static path didn't lead to a match,
		// continue to check wildcard/globstar at this level.
	}

	// Priority 2: Check wildcard match.
	if node.wildcardChild != nil {
		// Found a wildcard path, recurse.
		val, found := t.lookupRecursive(node.wildcardChild, remainingSegments)
		// If the wildcard path leads to a match, return it immediately.
		if found {
			return val, true
		}
		// If this path failed, continue to check globstar.
	}

	// Priority 3: Check globstar match (Recursive)
	if node.globstarChild != nil {
		nodeGlobstar := node.globstarChild

		// Path A: The globstar itself has a value, matching all remaining segments.
		if nodeGlobstar.hasValue {
			return nodeGlobstar.value, true
		}

		// Path B: The globstar matches 0 or more segments, and a subsequent part of the pattern matches.
		// We loop from i=0 (globstar matches 0 segments) up to
		// i=len(segments) (globstar matches all remaining segments).
		for i := 0; i <= len(segments); i++ {
			// Try to match the rest of the pattern (at node_globstar)
			// against the segments remaining (segments[i:]).
			val, found := t.lookupRecursive(nodeGlobstar, segments[i:])
			if found {
				return val, true
			}
		}
	}

	// No match found on any path from this node.
	return nil, false
}
