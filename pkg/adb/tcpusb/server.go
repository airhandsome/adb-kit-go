package tcpusb

import (
	"fmt"
	"net"
	"sync"
)

// Server 实现TCP/USB服务器
type Server struct {
	client      interface{} // ADB客户端接口
	serial      string      // 设备序列号
	options     map[string]interface{}
	connections []*Socket
	server      net.Listener
	mu          sync.Mutex
	handlers    map[string][]func(interface{})
}

// NewServer 创建新的服务器实例
func NewServer(client interface{}, serial string, options map[string]interface{}) *Server {
	return &Server{
		client:      client,
		serial:      serial,
		options:     options,
		connections: make([]*Socket, 0),
		handlers:    make(map[string][]func(interface{})),
	}
}

// Listen 开始监听连接
func (s *Server) Listen(address string) error {
	var err error
	s.server, err = net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("监听失败: %v", err)
	}

	// 触发监听事件
	s.emit("listening", nil)

	// 开始接受连接
	go s.acceptLoop()

	return nil
}

// acceptLoop 持续接受新连接
func (s *Server) acceptLoop() {
	for {
		conn, err := s.server.Accept()
		if err != nil {
			s.emit("error", err)
			continue
		}

		// 创建新的Socket
		socket := NewSocket(s.client, s.serial, conn, s.options)

		s.mu.Lock()
		s.connections = append(s.connections, socket)
		s.mu.Unlock()

		// 处理Socket事件
		socket.On("error", func(data interface{}) {
			s.emit("error", data)
		})

		socket.On("end", func(data interface{}) {
			s.mu.Lock()
			// 从连接列表中移除
			for i, c := range s.connections {
				if c == socket {
					s.connections = append(s.connections[:i], s.connections[i+1:]...)
					break
				}
			}
			s.mu.Unlock()
		})

		// 触发连接事件
		s.emit("connection", socket)
	}
}

// Close 关闭服务器
func (s *Server) Close() error {
	if s.server != nil {
		err := s.server.Close()
		if err != nil {
			return fmt.Errorf("关闭服务器失败: %v", err)
		}
	}

	s.emit("close", nil)
	return nil
}

// End 结束所有连接
func (s *Server) End() {
	s.mu.Lock()
	for _, conn := range s.connections {
		conn.End("")
	}
	s.connections = make([]*Socket, 0)
	s.mu.Unlock()
}

// On 注册事件处理器
func (s *Server) On(event string, handler func(interface{})) {
	if s.handlers[event] == nil {
		s.handlers[event] = make([]func(interface{}), 0)
	}
	s.handlers[event] = append(s.handlers[event], handler)
}

// emit 触发事件
func (s *Server) emit(event string, data interface{}) {
	if handlers, ok := s.handlers[event]; ok {
		for _, handler := range handlers {
			handler(data)
		}
	}
}

// GetConnections 获取当前所有连接
func (s *Server) GetConnections() []*Socket {
	s.mu.Lock()
	defer s.mu.Unlock()
	connections := make([]*Socket, len(s.connections))
	copy(connections, s.connections)
	return connections
}

// GetConnectionCount 获取当前连接数
func (s *Server) GetConnectionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.connections)
}

// IsListening 检查服务器是否正在监听
func (s *Server) IsListening() bool {
	return s.server != nil
}
