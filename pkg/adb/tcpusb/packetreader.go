package tcpusb

import (
	"encoding/binary"
	"io"
)

// PacketReader 实现数据包读取器
type PacketReader struct {
	stream   io.Reader
	inBody   bool
	buffer   []byte
	packet   *Packet
	handlers map[string][]func(interface{})
}

// NewPacketReader 创建新的数据包读取器
func NewPacketReader(stream io.Reader) *PacketReader {
	pr := &PacketReader{
		stream:   stream,
		handlers: make(map[string][]func(interface{})),
	}

	// 开始读取循环
	go pr.readLoop()
	return pr
}

// readLoop 持续读取数据包
func (r *PacketReader) readLoop() {
	for {
		if err := r.tryRead(); err != nil {
			if err == io.EOF {
				r.emit("end", nil)
			} else {
				r.emit("error", err)
			}
			return
		}
	}
}

// tryRead 尝试读取数据包
func (r *PacketReader) tryRead() error {
	for {
		if err := r.appendChunk(); err != nil {
			return err
		}

		for r.buffer != nil {
			if r.inBody {
				if len(r.buffer) < int(r.packet.Length) {
					return nil
				}

				r.packet.Data = make([]byte, r.packet.Length)
				copy(r.packet.Data, r.buffer[:r.packet.Length])
				r.consume(int(r.packet.Length))

				if !r.packet.VerifyChecksum() {
					return &ChecksumError{Packet: r.packet}
				}

				r.emit("packet", r.packet)
				r.inBody = false
			} else {
				if len(r.buffer) < 24 {
					return nil
				}

				header := make([]byte, 24)
				copy(header, r.buffer[:24])
				r.consume(24)

				r.packet = &Packet{
					Command: binary.LittleEndian.Uint32(header[0:4]),
					Arg0:    binary.LittleEndian.Uint32(header[4:8]),
					Arg1:    binary.LittleEndian.Uint32(header[8:12]),
					Length:  binary.LittleEndian.Uint32(header[12:16]),
					Check:   binary.LittleEndian.Uint32(header[16:20]),
					Magic:   binary.LittleEndian.Uint32(header[20:24]),
				}

				if !r.packet.VerifyMagic() {
					return &MagicError{Packet: r.packet}
				}

				if r.packet.Length == 0 {
					r.emit("packet", r.packet)
				} else {
					r.inBody = true
				}
			}
		}
	}
}

// appendChunk 追加数据块
func (r *PacketReader) appendChunk() error {
	chunk := make([]byte, 16384) // 16KB buffer
	n, err := r.stream.Read(chunk)
	if err != nil {
		return err
	}

	if n > 0 {
		if r.buffer != nil {
			newBuffer := make([]byte, len(r.buffer)+n)
			copy(newBuffer, r.buffer)
			copy(newBuffer[len(r.buffer):], chunk[:n])
			r.buffer = newBuffer
		} else {
			r.buffer = make([]byte, n)
			copy(r.buffer, chunk[:n])
		}
	}

	return nil
}

// consume 消费缓冲区数据
func (r *PacketReader) consume(length int) {
	if length == len(r.buffer) {
		r.buffer = nil
	} else {
		r.buffer = r.buffer[length:]
	}
}

// On 注册事件处理器
func (r *PacketReader) On(event string, handler func(interface{})) {
	if r.handlers[event] == nil {
		r.handlers[event] = make([]func(interface{}), 0)
	}
	r.handlers[event] = append(r.handlers[event], handler)
}

// emit 触发事件
func (r *PacketReader) emit(event string, data interface{}) {
	if handlers, ok := r.handlers[event]; ok {
		for _, handler := range handlers {
			handler(data)
		}
	}
}

// ChecksumError 校验和错误
type ChecksumError struct {
	Packet *Packet
}

func (e *ChecksumError) Error() string {
	return "Checksum mismatch"
}

// MagicError 魔数错误
type MagicError struct {
	Packet *Packet
}

func (e *MagicError) Error() string {
	return "Magic value mismatch"
}

// Close 关闭读取器
func (r *PacketReader) Close() error {
	if closer, ok := r.stream.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
