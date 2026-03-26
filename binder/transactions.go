package binder

const (
	FirstCallTransaction = 0x00000001
	LastCallTransaction  = 0x00ffffff

	PingTransaction                = uint32('_')<<24 | uint32('P')<<16 | uint32('N')<<8 | uint32('G')
	DumpTransaction                = uint32('_')<<24 | uint32('D')<<16 | uint32('M')<<8 | uint32('P')
	ShellCommandTransaction        = uint32('_')<<24 | uint32('C')<<16 | uint32('M')<<8 | uint32('D')
	InterfaceTransaction           = uint32('_')<<24 | uint32('N')<<16 | uint32('T')<<8 | uint32('F')
	ExtensionTransaction           = uint32('_')<<24 | uint32('E')<<16 | uint32('X')<<8 | uint32('T')
	DebugPIDTransaction            = uint32('_')<<24 | uint32('P')<<16 | uint32('I')<<8 | uint32('D')
	GetInterfaceHashTransaction    = 0x00fffffd
	GetInterfaceVersionTransaction = 0x00fffffe
)
