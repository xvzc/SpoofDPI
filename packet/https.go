package packet

import (
	"regexp"
)

type HttpsPacket struct {
	raw []byte
}

func NewHttpsPacket(raw []byte) HttpsPacket {
	return HttpsPacket{
		raw: raw,
	}
}

func (p *HttpsPacket) Raw() []byte {
	return p.raw
}

var PatternMatcher *regexp.Regexp
var UrlsMatcher *regexp.Regexp

func (p *HttpsPacket) SplitInChunks() [][]byte {
	if len(p.Raw()) < 1 {
		return [][]byte{p.Raw()}
	}

	// If the packet matches the pattern or the URLs, we don't split it
	if PatternMatcher != nil || UrlsMatcher != nil {
		if (PatternMatcher != nil && PatternMatcher.Match(p.Raw())) || (UrlsMatcher != nil && UrlsMatcher.Match(p.Raw())) {
			return [][]byte{(p.Raw())[:1], (p.Raw())[1:]}
		}
		
		return [][]byte{p.Raw()}
	}

	return [][]byte{(p.Raw())[:1], (p.Raw())[1:]}
}
