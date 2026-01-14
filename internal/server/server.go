package server

import "context"

// Server represents a core component that processes network traffic
type Server interface {
	// Start begins the execution of the server module
	Start(ctx context.Context, ready chan<- struct{}) error
	SetNetworkConfig() error
	UnsetNetworkConfig() error

	// Stop gracefully terminates the server and releases resources
	Stop() error

	// Addr returns the network address or interface name the server is bound to
	Addr() string
}
