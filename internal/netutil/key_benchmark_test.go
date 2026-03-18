package netutil

import (
	"net"
	"testing"
)

func BenchmarkKeyAllocation(b *testing.B) {
	// Dummy test data
	clientAddr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 12345}
	targetAddrStr := "142.250.190.46:443"

	uAddr, _ := net.ResolveUDPAddr("udp", targetAddrStr)

	b.Run("StringKey_Legacy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// This was the old way
			_ = clientAddr.String() + ">" + targetAddrStr
		}
	})

	b.Run("StructKey_NATKey", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// This is the new way (zero allocation)
			_ = NewNATKey(clientAddr, uAddr)
		}
	})
}
