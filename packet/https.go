package packet

type Https struct {
	Raw *[]byte
}

func NewHttps(raw *[]byte) Https {
	return Https{
		Raw: raw,
	}
}

func (r Https) SplitInChunks() [][]byte {
	if len(*r.Raw) < 1 {
		return [][]byte{*r.Raw}
	}

	return [][]byte{(*r.Raw)[:1], (*r.Raw)[1:]}
}
