package tree

import "strings"

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
	root *domainNode
}

// NewDomainSearchTree creates a new, empty domain tree.
func NewDomainSearchTree() RadixTree {
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
func (t *domainSearchTree) Insert(domain string, value any) {
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

// Lookup searches for the domain in the tree.
func (t *domainSearchTree) Lookup(domain string) (any, bool) {
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
		// 1. Check for a specific value at this node.
		if node.hasValue {
			return node.value, true
		}
		// 2. Check for a root domain wildcard match (e.g., "*.net" for "net")
		if node.wildcardChild != nil && node.wildcardChild.hasValue {
			return node.wildcardChild.value, true
		}
		// 3. No match.
		return nil, false
	}

	segment := segments[0]
	remainingSegments := segments[1:]

	// Priority 1: Check static match.
	if child, ok := node.children[segment]; ok {
		// Found a static path, recurse.
		val, found := t.lookupRecursive(child, remainingSegments)
		// If this path (e.g., "api") leads to a match, return it.
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
		// If the wildcard path leads to a match, return it.
		if found {
			return val, true
		}
		// If this path failed, continue to check globstar.
	}

	// Priority 3: Check globstar match.
	// This is terminal: it matches all remaining segments.
	if node.globstarChild != nil && node.globstarChild.hasValue {
		return node.globstarChild.value, true
	}

	// 4. No match found on any path from this node.
	return nil, false
}

// --- Main Function (Demonstration) ---
