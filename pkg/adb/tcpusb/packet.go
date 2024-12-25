package tcpusb

import (
	"encoding/binary"
	"fmt"
)

// Packet 命令包结构
type Packet struct {
	Command uint32
	Arg0    uint32
	Arg1    uint32
	Length  uint32
	Check   uint32
	Magic   uint32
	Data    []byte
}

// 命令常量
const (
	A_SYNC = 0x434e5953
	A_CNXN = 0x4e584e43
	A_OPEN = 0x4e45504f
	A_OKAY = 0x59414b4f
	A_CLSE = 0x45534c43
	A_WRTE = 0x45545257
	A_AUTH = 0x48545541
)

// NewPacket 创建新的数据包
func NewPacket(command, arg0, arg1 uint32, data []byte) *Packet {
	return &Packet{
		Command: command,
		Arg0:    arg0,
		Arg1:    arg1,
		Length:  uint32(len(data)),
		Check:   checksum(data),
		Magic:   magic(command),
		Data:    data,
	}
}

// checksum 计算数据校验和
func checksum(data []byte) uint32 {
	var sum uint32
	if data != nil {
		for _, b := range data {
			sum += uint32(b)
		}
	}
	return sum
}

// magic 计算命令魔数
func magic(command uint32) uint32 {
	return ^command // 按位取反
}

// Assemble 组装数据包
func Assemble(command, arg0, arg1 uint32, data []byte) []byte {
	var size int
	if data != nil {
		size = 24 + len(data)
	} else {
		size = 24
	}

	chunk := make([]byte, size)
	binary.LittleEndian.PutUint32(chunk[0:], command)
	binary.LittleEndian.PutUint32(chunk[4:], arg0)
	binary.LittleEndian.PutUint32(chunk[8:], arg1)

	if data != nil {
		binary.LittleEndian.PutUint32(chunk[12:], uint32(len(data)))
		binary.LittleEndian.PutUint32(chunk[16:], checksum(data))
		binary.LittleEndian.PutUint32(chunk[20:], magic(command))
		copy(chunk[24:], data)
	} else {
		binary.LittleEndian.PutUint32(chunk[12:], 0)
		binary.LittleEndian.PutUint32(chunk[16:], 0)
		binary.LittleEndian.PutUint32(chunk[20:], magic(command))
	}

	return chunk
}

// VerifyChecksum 验证数据校验和
func (p *Packet) VerifyChecksum() bool {
	return p.Check == checksum(p.Data)
}

// VerifyMagic 验证命令魔数
func (p *Packet) VerifyMagic() bool {
	return p.Magic == magic(p.Command)
}

// String 返回数据包的字符串表示
func (p *Packet) String() string {
	var cmdType string
	switch p.Command {
	case A_SYNC:
		cmdType = "SYNC"
	case A_CNXN:
		cmdType = "CNXN"
	case A_OPEN:
		cmdType = "OPEN"
	case A_OKAY:
		cmdType = "OKAY"
	case A_CLSE:
		cmdType = "CLSE"
	case A_WRTE:
		cmdType = "WRTE"
	case A_AUTH:
		cmdType = "AUTH"
	default:
		cmdType = "UNKNOWN"
	}
	return fmt.Sprintf("%s arg0=%d arg1=%d length=%d", cmdType, p.Arg0, p.Arg1, p.Length)
}

// Swap32 32位整数字节序转换
func Swap32(n uint32) uint32 {
	return ((n & 0xFF) << 24) |
		((n & 0xFF00) << 8) |
		((n & 0xFF0000) >> 8) |
		((n & 0xFF000000) >> 24)
}

// Parse 解析数据包
func Parse(data []byte) (*Packet, error) {
	if len(data) < 24 {
		return nil, fmt.Errorf("数据包太短")
	}

	p := &Packet{
		Command: binary.LittleEndian.Uint32(data[0:4]),
		Arg0:    binary.LittleEndian.Uint32(data[4:8]),
		Arg1:    binary.LittleEndian.Uint32(data[8:12]),
		Length:  binary.LittleEndian.Uint32(data[12:16]),
		Check:   binary.LittleEndian.Uint32(data[16:20]),
		Magic:   binary.LittleEndian.Uint32(data[20:24]),
	}

	if p.Length > 0 {
		if len(data) < 24+int(p.Length) {
			return nil, fmt.Errorf("数据包不完整")
		}
		p.Data = make([]byte, p.Length)
		copy(p.Data, data[24:24+p.Length])
	}

	return p, nil
}
