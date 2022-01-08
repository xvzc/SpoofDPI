package packet

type HttpsRequest struct {
	Raw *[]byte
}

func NewHttpsRequest(raw *[]byte) HttpsRequest {
	return HttpsRequest{
		Raw: raw,
	}
}

func (r HttpsRequest) SplitInChunks() [][]byte {
	if len(*r.Raw) < 1 {
		return [][]byte{*r.Raw}
	}

	return [][]byte{(*r.Raw)[:1], (*r.Raw)[1:]}
}
