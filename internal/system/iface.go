package system

import (
	"fmt"
	"net"
)

// FindDefaultInterface는 공용 DNS 서버로의 UDP 다이얼링을 시도하여
// 인터넷 연결에 사용되는 기본 네트워크 인터페이스를 찾습니다.
// [MODIFIED] string 대신 *net.Interface를 반환합니다.
func FindDefaultInterface() (*net.Interface, error) {
	// 1. 공용 DNS 서버 목록
	dnsServers := []string{
		"8.8.8.8:53",
		"8.8.4.4:53",
		"1.1.1.1:53",
		"1.0.0.1:53",
		"9.9.9.9:53",
	}

	var conn net.Conn
	var err error

	// 2. 성공할 때까지 하나씩 UDP Dial 시도
	for _, server := range dnsServers {
		conn, err = net.Dial("udp", server)
		if err == nil {
			// 성공
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf( // [MODIFIED] "" -> nil
			"could not dial any public DNS to determine default interface: %w",
			err,
		)
	}
	defer func() { _ = conn.Close() }()

	// 3. 연결에 사용된 로컬 IP 주소 확인
	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf(
			"could not determine local address from UDP connection",
		) // [MODIFIED] "" -> nil
	}

	// 4. 이 로컬 IP를 가진 네트워크 인터페이스(iface)를 검색
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf(
			"could not get network interfaces: %w",
			err,
		) // [MODIFIED] "" -> nil
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue // 주소를 가져올 수 없는 인터페이스는 건너뜀
		}
		for _, addr := range addrs {
			// net.IPNet 타입인지 확인
			if ipnet, ok := addr.(*net.IPNet); ok {
				// Dial에 사용된 IP와 인터페이스의 IP가 일치하는지 확인
				if ipnet.IP.Equal(localAddr.IP) {
					// [MODIFIED] iface.Name (string) 대신 &iface (*net.Interface)를 반환
					return &iface, nil // 기본 인터페이스 찾음
				}
			}
		}
	}

	return nil, fmt.Errorf( // [MODIFIED] "" -> nil
		"failed to find default interface for local IP: %s",
		localAddr.IP,
	)
}
