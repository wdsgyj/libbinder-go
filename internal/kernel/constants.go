package kernel

const (
	FirstCallTransaction = 0x00000001
	LastCallTransaction  = 0x00ffffff
	PingTransaction      = uint32('_')<<24 | uint32('P')<<16 | uint32('N')<<8 | uint32('G')
	InterfaceTransaction = uint32('_')<<24 | uint32('N')<<16 | uint32('T')<<8 | uint32('F')
)
