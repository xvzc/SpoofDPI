package tree

// RadixTree defines the interface for a domain matching structure.
// It uses 'any' (interface{}) to store arbitrary values.
type RadixTree interface {
	// Insert adds a domain and its associated value to the tree.
	Insert(domain string, value any)
	// Lookup searches for the domain in the tree.
	// It returns the found value (or nil) and a boolean indicating success.
	Lookup(domain string) (any, bool)
}
