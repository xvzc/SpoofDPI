package packet

type HttpsPacket struct {
	Raw *[]byte
}

func NewHttpsPacket(raw *[]byte) HttpsPacket {
	return HttpsPacket{
		Raw: raw,
	}
}

func (r HttpsPacket) SplitInChunks() [][]byte {
	if len(*r.Raw) < 1 {
		return [][]byte{*r.Raw}
	}

	return [][]byte{(*r.Raw)[:1], (*r.Raw)[1:]}
}
