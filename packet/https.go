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
