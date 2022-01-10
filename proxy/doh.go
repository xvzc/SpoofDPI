package proxy

func (p *Proxy) DnsLookupOverHttps(domain string) (string, error) {
	// Perform a A lookup on example.com

	a, _, err := p.DNS.LookupA(domain)
	if err != nil {
		return "", err
	}

	ip := a[0].IP4

	return ip, nil
}
