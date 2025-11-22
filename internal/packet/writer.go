package packet

type PacketWriter interface {
	WritePacketData(data []byte) error
	Close()
}
