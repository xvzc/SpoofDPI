package packet

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

func (p *HttpsPacket) SplitInChunks() [][]byte {
	if len(p.Raw()) < 1 {
		return [][]byte{p.Raw()}
	}

	return [][]byte{(p.Raw())[:1], (p.Raw())[1:]}
}
