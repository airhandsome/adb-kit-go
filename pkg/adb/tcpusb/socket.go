package tcpusb

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"sync"
)

const (
	UINT32_MAX = 0xFFFFFFFF
	UINT16_MAX = 0xFFFF

	AUTH_TOKEN        = 1
	AUTH_SIGNATURE    = 2
	AUTH_RSAPUBLICKEY = 3

	TOKEN_LENGTH = 20
)

// Socket 实现ADB Socket连接
type Socket struct {
	client        interface{}
	serial        string
	conn          net.Conn
	options       map[string]interface{}
	reader        *PacketReader
	ended         bool
	version       int
	maxPayload    int
	authorized    bool
	syncToken     *RollingCounter
	remoteId      *RollingCounter
	services      *ServiceMap
	remoteAddress string
	token         []byte
	signature     []byte
	mu            sync.Mutex
	handlers      map[string][]func(interface{})
}

// NewSocket 创建新的Socket实例
func NewSocket(client interface{}, serial string, conn net.Conn, options map[string]interface{}) *Socket {
	s := &Socket{
		client:        client,
		serial:        serial,
		conn:          conn,
		options:       options,
		ended:         false,
		version:       1,
		maxPayload:    4096,
		authorized:    false,
		syncToken:     NewRollingCounter(UINT32_MAX, 0),
		remoteId:      NewRollingCounter(UINT32_MAX, 0),
		services:      NewServiceMap(),
		remoteAddress: conn.RemoteAddr().String(),
		handlers:      make(map[string][]func(interface{})),
	}

	// 设置TCP无延迟
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetNoDelay(true)
	}

	// 创建数据包读取器
	s.reader = NewPacketReader(conn)
	s.reader.On("packet", s.handlePacket)
	s.reader.On("error", func(data interface{}) {
		s.emit("error", fmt.Errorf("PacketReader error: %v", data))
		s.End("")
	})
	s.reader.On("end", s.End)

	return s
}

// handlePacket 处理接收到的数据包
func (s *Socket) handlePacket(data interface{}) {
	packet := data.(*Packet)
	if s.ended {
		return
	}

	s.emit("userActivity", packet)

	switch packet.Command {
	case A_SYNC:
		s.handleSyncPacket(packet)
	case A_CNXN:
		s.handleConnectionPacket(packet)
	case A_OPEN:
		s.handleOpenPacket(packet)
	case A_OKAY, A_WRTE, A_CLSE:
		s.forwardServicePacket(packet)
	case A_AUTH:
		s.handleAuthPacket(packet)
	default:
		s.emitError(fmt.Errorf("Unknown command %d", packet.Command))
	}
}

// handleSyncPacket 处理同步包
func (s *Socket) handleSyncPacket(packet *Packet) {
	s.write(Assemble(A_SYNC, 1, uint32(s.syncToken.Next()), nil))
}

// handleConnectionPacket 处理连接包
func (s *Socket) handleConnectionPacket(packet *Packet) {
	s.version = int(Swap32(packet.Arg0))
	s.maxPayload = int(packet.Arg1)
	if s.maxPayload > UINT16_MAX {
		s.maxPayload = UINT16_MAX
	}

	token, err := s.createToken()
	if err != nil {
		s.emitError(err)
		return
	}
	s.token = token

	s.write(Assemble(A_AUTH, AUTH_TOKEN, 0, token))
}

// handleAuthPacket 处理认证包
func (s *Socket) handleAuthPacket(packet *Packet) {
	switch packet.Arg0 {
	case AUTH_SIGNATURE:
		if s.signature == nil {
			s.signature = packet.Data
		}
		s.write(Assemble(A_AUTH, AUTH_TOKEN, 0, s.token))

	case AUTH_RSAPUBLICKEY:
		if s.signature == nil {
			s.emitError(fmt.Errorf("Public key sent before signature"))
			return
		}

		// 验证签名和处理公钥...
		// 这里需要实现具体的验证逻辑

	default:
		s.emitError(fmt.Errorf("Unknown authentication method %d", packet.Arg0))
	}
}

// write 发送数据
func (s *Socket) write(data []byte) error {
	if s.ended {
		return fmt.Errorf("Connection ended")
	}
	_, err := s.conn.Write(data)
	return err
}

// End 结束连接
func (s *Socket) End(data interface{}) {
	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return
	}
	s.ended = true
	s.mu.Unlock()

	s.services.End()
	s.conn.Close()
	s.emit("end", nil)
}

// createToken 创建认证令牌
func (s *Socket) createToken() ([]byte, error) {
	token := make([]byte, TOKEN_LENGTH)
	_, err := rand.Read(token)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate token: %v", err)
	}
	return token, nil
}

// On 注册事件处理器
func (s *Socket) On(event string, handler func(interface{})) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.handlers[event] == nil {
		s.handlers[event] = make([]func(interface{}), 0)
	}
	s.handlers[event] = append(s.handlers[event], handler)
}

// emit 触发事件
func (s *Socket) emit(event string, data interface{}) {
	s.mu.Lock()
	handlers := s.handlers[event]
	s.mu.Unlock()

	for _, handler := range handlers {
		handler(data)
	}
}

// emitError 触发错误事件
func (s *Socket) emitError(err error) {
	s.emit("error", err)
	s.End("")
}

// IsEnded 检查连接是否已结束
func (s *Socket) IsEnded() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ended
}

// RemoteAddress 获取远程地址
func (s *Socket) RemoteAddress() string {
	return s.remoteAddress
}

// IsAuthorized 检查是否已认证
func (s *Socket) IsAuthorized() bool {
	return s.authorized
}

// forwardServicePacket 转发服务数据包
func (s *Service) forwardServicePacket(packet *Packet) error {
	if !s.opened {
		return fmt.Errorf("service not opened")
	}

	if s.ended {
		return fmt.Errorf("service already ended")
	}

	switch packet.Command {
	case A_OKAY:
		// 处理确认包
		s.needAck = false
		s.tryPush()

	case A_WRTE:
		// 处理写入包
		if packet.Data != nil {
			if _, err := s.transport.Write(packet.Data); err != nil {
				return fmt.Errorf("failed to write data: %v", err)
			}
		}

		// 发送确认
		ack := Assemble(A_OKAY, s.localId, s.remoteId, nil)
		if _, err := s.socket.Write(ack); err != nil {
			return fmt.Errorf("failed to send ACK: %v", err)
		}

	case A_CLSE:
		// 处理关闭包
		return s.End()

	default:
		return fmt.Errorf("unexpected packet command: %d", packet.Command)
	}

	return nil
}

// dataTransferLoop 数据传输循环
func (s *Service) dataTransferLoop(transport io.ReadWriter) {
	defer s.End()

	buffer := make([]byte, 16384) // 16KB buffer
	for {
		s.mu.Lock()
		if s.ended {
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()

		// 读取传输数据
		n, err := transport.Read(buffer)
		if err != nil {
			if err != io.EOF {
				s.emit("error", fmt.Errorf("transport read error: %v", err))
			}
			return
		}

		if n > 0 {
			s.mu.Lock()
			if s.ended {
				s.mu.Unlock()
				return
			}

			// 发送数据包
			packet := Assemble(A_WRTE, s.localId, s.remoteId, buffer[:n])
			if _, err := s.socket.Write(packet); err != nil {
				s.emit("error", fmt.Errorf("failed to write packet: %v", err))
				s.mu.Unlock()
				return
			}
			s.needAck = true
			s.mu.Unlock()

			// 等待确认
			for {
				s.mu.Lock()
				if !s.needAck || s.ended {
					s.mu.Unlock()
					break
				}
				s.mu.Unlock()
			}
		}
	}
}

// readError 读取错误信息
func readError(r io.Reader) (string, error) {
	// 读取错误消息长度
	lenBuf := make([]byte, 4)
	if _, err := r.Read(lenBuf); err != nil {
		return "", fmt.Errorf("failed to read error length: %v", err)
	}

	length := int(lenBuf[0]) | int(lenBuf[1])<<8 | int(lenBuf[2])<<16 | int(lenBuf[3])<<24

	// 读取错误消息
	errBuf := make([]byte, length)
	if _, err := r.Read(errBuf); err != nil {
		return "", fmt.Errorf("failed to read error message: %v", err)
	}

	return string(errBuf), nil
}

// handleOpenPacket 处理打开包
func (s *Socket) handleOpenPacket(packet *Packet) error {
	if !s.authorized {
		return &UnauthorizedError{}
	}

	remoteId := packet.Arg0
	localId := s.remoteId.Next()

	// 检查数据包是否有效
	if packet.Data == nil || len(packet.Data) < 2 {
		return fmt.Errorf("empty service name")
	}

	// 创建新服务
	service := NewService(s.client, s.serial, uint32(localId), remoteId, s)

	// 注册错误处理
	service.On("error", func(data interface{}) {
		if err, ok := data.(error); ok {
			s.emit("error", err)
		}
	})

	// 注册结束处理
	service.On("end", func(interface{}) {
		s.services.Remove(uint32(localId))
	})

	// 将服务添加到服务映射
	if err := s.services.Insert(uint32(localId), service); err != nil {
		return fmt.Errorf("failed to insert service: %v", err)
	}

	// 处理数据包
	if err := service.Handle(packet); err != nil {
		s.services.Remove(uint32(localId))
		service.End()
		return err
	}

	return nil
}

// forwardServicePacket 转发服务数据包
func (s *Socket) forwardServicePacket(packet *Packet) error {
	if !s.authorized {
		return &UnauthorizedError{}
	}

	localId := packet.Arg1

	// 获取对应的服务
	service := s.services.Get(localId)
	if service == nil {
		return fmt.Errorf("received packet for non-existent service")
	}

	// 转发数据包到服务
	return service.Handle(packet)
}

// UnauthorizedError 未授权错误
type UnauthorizedError struct{}

func (e *UnauthorizedError) Error() string {
	return "Unauthorized access"
}
