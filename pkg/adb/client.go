package adb

import (
	"adb-kit-go/pkg/adb/command/host"
	"fmt"
	"sync"
)

// Client ADB客户端
type Client struct {
	options *Options
	mu      sync.Mutex
}

// Options 客户端配置选项
type Options struct {
	Port int    // ADB服务器端口
	Bin  string // ADB可执行文件路径
}

// NewClient 创建新的ADB客户端
func NewClient(options *Options) *Client {
	if options == nil {
		options = &Options{}
	}
	if options.Port == 0 {
		options.Port = 5037
	}
	if options.Bin == "" {
		options.Bin = "adb"
	}

	return &Client{
		options: options,
	}
}

// CreateConnection 创建新的连接
func (c *Client) CreateConnection() (*Connection, error) {
	conn := NewConnection(c.options)

	err := conn.Connect()
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}

	return conn, nil
}

// Version 获取ADB服务器版本
func (c *Client) Version() (int, error) {
	conn, err := c.CreateConnection()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	cmd := host.NewVersionCommand(conn.reader, conn.writer)
	return cmd.Execute()
}

// Connect 连接到设备
func (c *Client) Connect(host string, port int) error {
	if port == 0 {
		port = 5555
	}

	conn, err := c.CreateConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	cmd := NewHostConnectCommand(conn)
	return cmd.Execute(host, port)
}

// Disconnect 断开设备连接
func (c *Client) Disconnect(host string, port int) error {
	if port == 0 {
		port = 5555
	}

	conn, err := c.CreateConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	cmd := NewHostDisconnectCommand(conn)
	return cmd.Execute(host, port)
}

// ListDevices 列出所有设备
func (c *Client) ListDevices() ([]Device, error) {
	conn, err := c.CreateConnection()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cmd := NewHostDevicesCommand(conn)
	return cmd.Execute()
}

// Transport 创建设备传输
func (c *Client) Transport(serial string) (*Transport, error) {
	conn, err := c.CreateConnection()
	if err != nil {
		return nil, err
	}

	cmd := NewHostTransportCommand(conn)
	err = cmd.Execute(serial)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return NewTransport(conn), nil
}

// Shell 执行Shell命令
func (c *Client) Shell(serial string, command string) (*ShellResponse, error) {
	transport, err := c.Transport(serial)
	if err != nil {
		return nil, err
	}

	cmd := NewShellCommand(transport)
	return cmd.Execute(command)
}

// Install 安装APK
func (c *Client) Install(serial string, apkPath string) error {
	transport, err := c.Transport(serial)
	if err != nil {
		return err
	}

	cmd := NewInstallCommand(transport)
	return cmd.Execute(apkPath)
}

// Uninstall 卸载应用
func (c *Client) Uninstall(serial string, packageName string) error {
	transport, err := c.Transport(serial)
	if err != nil {
		return err
	}

	cmd := NewUninstallCommand(transport)
	return cmd.Execute(packageName)
}

// Push 推送文件到设备
func (c *Client) Push(serial string, local string, remote string) error {
	transport, err := c.Transport(serial)
	if err != nil {
		return err
	}

	cmd := NewSyncCommand(transport)
	return cmd.Push(local, remote)
}

// Pull 从设备拉取文件
func (c *Client) Pull(serial string, remote string, local string) error {
	transport, err := c.Transport(serial)
	if err != nil {
		return err
	}

	cmd := NewSyncCommand(transport)
	return cmd.Pull(remote, local)
}

// Forward 端口转发
func (c *Client) Forward(serial string, local string, remote string) error {
	conn, err := c.CreateConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	cmd := NewForwardCommand(conn)
	return cmd.Execute(serial, local, remote)
}

// CreateTcpUsbBridge 创建TCP/USB桥接
func (c *Client) CreateTcpUsbBridge(serial string, options map[string]interface{}) (*TcpUsbServer, error) {
	return NewTcpUsbServer(c, serial, options), nil
}

// Device 设备信息
type Device struct {
	Serial string
	State  string
	Path   string
}

// ShellResponse Shell命令响应
type ShellResponse struct {
	Output string
	Error  error
}

// Transport 设备传输
type Transport struct {
	conn *Connection
}

func NewTransport(conn *Connection) *Transport {
	return &Transport{conn: conn}
}

func (t *Transport) Close() error {
	return t.conn.Close()
}
