package adb

import (
	"fmt"
	"strconv"
	"strings"
)

// Command ADB命令基类
type Command struct {
	conn     *Connection
	parser   *Parser
	protocol *Protocol
}

// NewCommand 创建新的命令实例
func NewCommand(conn *Connection) *Command {
	return &Command{
		conn:     conn,
		parser:   NewParser(),
		protocol: NewProtocol(),
	}
}

// Execute 执行命令（基类方法）
func (c *Command) Execute() error {
	return fmt.Errorf("missing implementation")
}

// send 发送数据
func (c *Command) send(data []byte) error {
	encoded := c.protocol.EncodeData(data)
	_, err := c.conn.Write(encoded)
	return err
}

// escape 转义参数（用于安全性）
func (c *Command) escape(arg interface{}) string {
	switch v := arg.(type) {
	case int:
		return strconv.Itoa(v)
	default:
		// 使用单引号包裹并转义内部的单引号
		str := fmt.Sprintf("%v", v)
		return "'" + strings.ReplaceAll(str, "'", "'\"'\"'") + "'"
	}
}

// escapeCompat 兼容性转义（用于特定设备）
func (c *Command) escapeCompat(arg interface{}) string {
	switch v := arg.(type) {
	case int:
		return strconv.Itoa(v)
	default:
		// 使用双引号包裹并转义特殊字符
		str := fmt.Sprintf("%v", v)
		escaped := strings.NewReplacer(
			"$", "\\$",
			"`", "\\`",
			"\\", "\\\\",
			"!", "\\!",
			"\"", "\\\"",
		).Replace(str)
		return "\"" + escaped + "\""
	}
}

// checkResponse 检查响应
func (c *Command) checkResponse(expected string) error {
	response, err := c.parser.ReadString(4)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if response != expected {
		return fmt.Errorf("unexpected response: %s (expected %s)", response, expected)
	}

	return nil
}

// readLength 读取长度
func (c *Command) readLength() (int, error) {
	lenStr, err := c.parser.ReadString(4)
	if err != nil {
		return 0, fmt.Errorf("failed to read length: %v", err)
	}

	length, err := strconv.ParseInt(lenStr, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid length format: %v", err)
	}

	return int(length), nil
}

// readData 读取数据
func (c *Command) readData(length int) ([]byte, error) {
	data, err := c.parser.ReadBytes(length)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %v", err)
	}
	return data, nil
}

// formatCmd 格式化命令
func (c *Command) formatCmd(cmd string, args ...interface{}) string {
	escapedArgs := make([]string, len(args))
	for i, arg := range args {
		escapedArgs[i] = c.escape(arg)
	}
	return fmt.Sprintf(cmd, escapedArgs...)
}

// formatCmdCompat 格式化兼容命令
func (c *Command) formatCmdCompat(cmd string, args ...interface{}) string {
	escapedArgs := make([]string, len(args))
	for i, arg := range args {
		escapedArgs[i] = c.escapeCompat(arg)
	}
	return fmt.Sprintf(cmd, escapedArgs...)
}

// HostCommand host命令基类
type HostCommand struct {
	*Command
}

// NewHostCommand 创建新的host命令
func NewHostCommand(conn *Connection) *HostCommand {
	return &HostCommand{
		Command: NewCommand(conn),
	}
}

// HostSerialCommand host-serial命令基类
type HostSerialCommand struct {
	*Command
	serial string
}

// NewHostSerialCommand 创建新的host-serial命令
func NewHostSerialCommand(conn *Connection, serial string) *HostSerialCommand {
	return &HostSerialCommand{
		Command: NewCommand(conn),
		serial:  serial,
	}
}

// HostTransportCommand host-transport命令基类
type HostTransportCommand struct {
	*Command
}

// NewHostTransportCommand 创建新的host-transport命令
func NewHostTransportCommand(conn *Connection) *HostTransportCommand {
	return &HostTransportCommand{
		Command: NewCommand(conn),
	}
}

// Example commands:

// ShellCommand shell命令
type ShellCommand struct {
	*HostTransportCommand
}

// NewShellCommand 创建新的shell命令
func NewShellCommand(conn *Connection) *ShellCommand {
	return &ShellCommand{
		HostTransportCommand: NewHostTransportCommand(conn),
	}
}

// Execute 执行shell命令
func (c *ShellCommand) Execute(command string) error {
	// 发送shell命令
	err := c.send([]byte("shell:" + command))
	if err != nil {
		return fmt.Errorf("failed to send shell command: %v", err)
	}

	// 检查响应
	return c.checkResponse("OKAY")
}

// InstallCommand 安装命令
type InstallCommand struct {
	*HostTransportCommand
}

// NewInstallCommand 创建新的安装命令
func NewInstallCommand(conn *Connection) *InstallCommand {
	return &InstallCommand{
		HostTransportCommand: NewHostTransportCommand(conn),
	}
}

// Execute 执行安装命令
func (c *InstallCommand) Execute(path string) error {
	// 发送安装命令
	err := c.send([]byte("install:" + path))
	if err != nil {
		return fmt.Errorf("failed to send install command: %v", err)
	}

	// 检查响应
	return c.checkResponse("OKAY")
}
