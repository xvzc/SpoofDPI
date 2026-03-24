package server

import "context"

// Server represents a core component that processes network traffic.
// ListenAndServe blocks until ctx is cancelled, then releases all resources.
type Server interface {
	ListenAndServe(ctx context.Context, ready chan<- struct{}) error
	SetNetworkConfig() (func() error, error)

	// Addr returns the network address or interface name the server is bound to
	Addr() string
}
