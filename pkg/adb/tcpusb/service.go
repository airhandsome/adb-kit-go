package tcpusb

import (
	"fmt"
	"io"
	"sync"
)

const (
	nullByteLen = 1
)

// Service 实现ADB服务
type Service struct {
	client    interface{}
	serial    string
	localId   uint32
	remoteId  uint32
	socket    io.ReadWriter
	opened    bool
	ended     bool
	transport io.Writer
	needAck   bool
	mu        sync.Mutex
	handlers  map[string][]func(interface{})
}

// NewService 创建新的服务实例
func NewService(client interface{}, serial string, localId, remoteId uint32, socket io.ReadWriter) *Service {
	return &Service{
		client:   client,
		serial:   serial,
		localId:  localId,
		remoteId: remoteId,
		socket:   socket,
		opened:   false,
		ended:    false,
		needAck:  false,
		handlers: make(map[string][]func(interface{})),
	}
}

// End 结束服务
func (s *Service) End() error {
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return nil
	}

	// 结束传输
	if s.transport != nil {
		if t, ok := s.transport.(io.Closer); ok {
			t.Close()
		}
	}

	s.ended = true
	localId := uint32(0)
	if s.opened {
		localId = s.localId
	}

	// 发送关闭包
	packet := Assemble(A_CLSE, localId, s.remoteId, nil)
	_, err := s.socket.Write(packet)
	s.mu.Unlock()

	s.emit("end", nil)
	return err
}

// Handle 处理接收到的数据包
func (s *Service) Handle(packet *Packet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ended {
		return nil
	}

	switch packet.Command {
	case A_OPEN:
		return s.handleOpenPacket(packet)
	case A_OKAY:
		return s.handleOkayPacket(packet)
	case A_WRTE:
		return s.handleWritePacket(packet)
	case A_CLSE:
		return s.handleClosePacket(packet)
	default:
		return fmt.Errorf("Unexpected packet %d", packet.Command)
	}
}

// handleOpenPacket 处理打开包
func (s *Service) handleOpenPacket(packet *Packet) error {
	// 建立传输连接
	transport, err := s.establishTransport(packet.Data)
	if err != nil {
		return err
	}
	s.transport = transport

	// 发送确认包
	ack := Assemble(A_OKAY, s.localId, s.remoteId, nil)
	if _, err := s.socket.Write(ack); err != nil {
		return err
	}

	s.opened = true
	go s.readLoop()
	return nil
}

// handleOkayPacket 处理确认包
func (s *Service) handleOkayPacket(packet *Packet) error {
	if !s.opened || s.transport == nil {
		return fmt.Errorf("Premature OKAY packet")
	}
	s.needAck = false
	s.tryPush()
	return nil
}

// handleWritePacket 处理写入包
func (s *Service) handleWritePacket(packet *Packet) error {
	if !s.opened || s.transport == nil {
		return fmt.Errorf("Premature WRTE packet")
	}

	// 写入数据
	if packet.Data != nil {
		if w, ok := s.transport.(io.Writer); ok {
			if _, err := w.Write(packet.Data); err != nil {
				return err
			}
		}
	}

	// 发送确认
	ack := Assemble(A_OKAY, s.localId, s.remoteId, nil)
	_, err := s.socket.Write(ack)
	return err
}

// handleClosePacket 处理关闭包
func (s *Service) handleClosePacket(packet *Packet) error {
	if !s.opened || s.transport == nil {
		return fmt.Errorf("Premature CLSE packet")
	}
	return s.End()
}

// readLoop 读取循环
func (s *Service) readLoop() {
	buffer := make([]byte, 16384)
	for {
		s.mu.Lock()
		if s.ended {
			s.mu.Unlock()
			return
		}

		if s.needAck {
			s.mu.Unlock()
			continue
		}

		// 读取数据
		if r, ok := s.transport.(io.Reader); ok {
			n, err := r.Read(buffer)
			if err != nil {
				if err != io.EOF {
					s.emit("error", err)
				}
				s.End()
				s.mu.Unlock()
				return
			}

			if n > 0 {
				// 发送数据包
				packet := Assemble(A_WRTE, s.localId, s.remoteId, buffer[:n])
				if _, err := s.socket.Write(packet); err != nil {
					s.emit("error", err)
					s.End()
					s.mu.Unlock()
					return
				}
				s.needAck = true
			}
		}
		s.mu.Unlock()
	}
}

// establishTransport 建立传输连接
func (s *Service) establishTransport(data []byte) (interface{}, error) {
	if len(data) < nullByteLen {
		return nil, fmt.Errorf("empty service name")
	}

	// 移除末尾的空字节
	serviceName := string(data[:len(data)-nullByteLen])

	// 这里需要根据服务名称创建对应的传输接口
	// 例如: "sync:", "shell:", "reverse:", 等

	// TODO: 实现具体的服务处理逻辑
	// 目前先返回一个基础的读写接口作为示例
	// 实际使用时需要根据不同的服务类型返回相应的处理接口

	r, w := io.Pipe()
	return struct {
		io.Reader
		io.Writer
		io.Closer
	}{
		Reader: r,
		Writer: w,
		Closer: r,
	}, nil
}

// tryPush 尝试推送数据
func (s *Service) tryPush() {
	if s.needAck || s.ended {
		return
	}
	// 实现数据推送逻辑
}

// On 注册事件处理器
func (s *Service) On(event string, handler func(interface{})) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.handlers[event] == nil {
		s.handlers[event] = make([]func(interface{}), 0)
	}
	s.handlers[event] = append(s.handlers[event], handler)
}

// emit 触发事件
func (s *Service) emit(event string, data interface{}) {
	s.mu.Lock()
	handlers := s.handlers[event]
	s.mu.Unlock()

	for _, handler := range handlers {
		handler(data)
	}
}

// IsEnded 检查服务是否已结束
func (s *Service) IsEnded() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ended
}

// IsOpened 检查服务是否已打开
func (s *Service) IsOpened() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opened
}
