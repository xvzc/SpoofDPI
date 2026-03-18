package netutil

func isIPv4Mapped(ip [16]byte) bool {
	// IPv4-mapped IPv6 address has the prefix 0:0:0:0:0:FFFF
	for i := 0; i < 10; i++ {
		if ip[i] != 0 {
			return false
		}
	}
	if ip[10] != 0xff || ip[11] != 0xff {
		return false
	}
	return true
}
