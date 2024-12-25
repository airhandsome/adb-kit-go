package adb

import (
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"
)

// Connection ADB连接
type Connection struct {
	options       *Options
	socket        net.Conn
	parser        *Parser
	mu            sync.Mutex
	handlers      map[string][]func(interface{})
	closed        bool
	triedStarting bool
}

// NewConnection 创建新的连接
func NewConnection(options *Options) *Connection {
	if options == nil {
		options = &Options{
			Port: 5037,
			Bin:  "adb",
		}
	}

	return &Connection{
		options:  options,
		handlers: make(map[string][]func(interface{})),
	}
}

// Connect 建立连接
func (c *Connection) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.socket != nil {
		return fmt.Errorf("connection already established")
	}

	// 连接到ADB服务器
	addr := fmt.Sprintf("127.0.0.1:%d", c.options.Port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		// 如果连接失败，尝试启动ADB服务器
		if !c.triedStarting {
			c.triedStarting = true
			if err := c.startServer(); err != nil {
				return fmt.Errorf("failed to start ADB server: %v", err)
			}
			return c.Connect()
		}
		return fmt.Errorf("failed to connect to ADB server: %v", err)
	}

	// 设置TCP选项
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}

	c.socket = conn
	c.parser = NewParser(conn)

	// 触发连接事件
	c.emit("connect", nil)

	// 启动读取循环
	go c.readLoop()

	return nil
}

// Write 写入数据
func (c *Connection) Write(data []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.socket == nil {
		return 0, fmt.Errorf("connection not established")
	}

	return c.socket.Write(data)
}

// Close 关闭连接
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.socket == nil || c.closed {
		return nil
	}

	c.closed = true
	err := c.socket.Close()
	c.socket = nil

	// 触发关闭事件
	c.emit("close", nil)

	return err
}

// startServer 启动ADB服务器
func (c *Connection) startServer() error {
	cmd := exec.Command(c.options.Bin, "start-server")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("adb start-server failed: %v, output: %s", err, output)
	}
	return nil
}

// readLoop 读取循环
func (c *Connection) readLoop() {
	buffer := make([]byte, 4096)
	for {
		c.mu.Lock()
		if c.closed || c.socket == nil {
			c.mu.Unlock()
			return
		}
		socket := c.socket
		c.mu.Unlock()

		n, err := socket.Read(buffer)
		if err != nil {
			if !c.closed {
				c.handleError(err)
			}
			return
		}

		if n > 0 {
			c.handleData(buffer[:n])
		}
	}
}

// handleData 处理接收到的数据
func (c *Connection) handleData(data []byte) {
	c.emit("data", data)
}

// handleError 处理错误
func (c *Connection) handleError(err error) {
	c.emit("error", err)
	c.Close()
}

// On 注册事件处理器
func (c *Connection) On(event string, handler func(interface{})) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.handlers[event] == nil {
		c.handlers[event] = make([]func(interface{}), 0)
	}
	c.handlers[event] = append(c.handlers[event], handler)
}

// Off 移除事件处理器
func (c *Connection) Off(event string, handler func(interface{})) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if handlers, ok := c.handlers[event]; ok {
		newHandlers := make([]func(interface{}), 0)
		for _, h := range handlers {
			if fmt.Sprintf("%p", h) != fmt.Sprintf("%p", handler) {
				newHandlers = append(newHandlers, h)
			}
		}
		c.handlers[event] = newHandlers
	}
}

// emit 触发事件
func (c *Connection) emit(event string, data interface{}) {
	c.mu.Lock()
	handlers := make([]func(interface{}), len(c.handlers[event]))
	copy(handlers, c.handlers[event])
	c.mu.Unlock()

	for _, handler := range handlers {
		handler(data)
	}
}

// GetParser 获取解析器
func (c *Connection) GetParser() *Parser {
	return c.parser
}

// IsConnected 检查是否已连接
func (c *Connection) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.socket != nil && !c.closed
}

// GetRemoteAddress 获取远程地址
func (c *Connection) GetRemoteAddress() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.socket != nil {
		return c.socket.RemoteAddr().String()
	}
	return ""
}

// SetTimeout 设置超时
func (c *Connection) SetTimeout(timeout time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.socket != nil {
		return c.socket.SetDeadline(time.Now().Add(timeout))
	}
	return fmt.Errorf("connection not established")
}

// ClearTimeout 清除超时
func (c *Connection) ClearTimeout() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.socket != nil {
		return c.socket.SetDeadline(time.Time{})
	}
	return fmt.Errorf("connection not established")
}
