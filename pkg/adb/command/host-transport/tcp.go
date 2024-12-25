package hosttransport

import (
	"fmt"
	"io"
)

// TcpCommand 实现TCP连接命令
type TcpCommand struct {
	BaseCommand
}

// NewTcpCommand 创建新的TCP命令实例
func NewTcpCommand(sender func(string) error, reader func(int) (string, error)) *TcpCommand {
	return &TcpCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行TCP连接命令
func (c *TcpCommand) Execute(port int, host string) (io.Reader, error) {
	// 构建TCP命令
	var cmd string
	if host != "" {
		cmd = fmt.Sprintf("tcp:%d:%s", port, host)
	} else {
		cmd = fmt.Sprintf("tcp:%d", port)
	}

	// 发送TCP命令
	if err := c.sender(cmd); err != nil {
		return nil, fmt.Errorf("发送TCP命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 返回一个读取器来处理TCP数据流
		return c.createTcpReader(), nil

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return nil, fmt.Errorf("TCP连接失败: %s", errMsg)

	default:
		return nil, fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// tcpReader 实现io.Reader接口，用于读取TCP数据流
type tcpReader struct {
	command *TcpCommand
	buffer  []byte
}

// createTcpReader 创建新的TCP读取器
func (c *TcpCommand) createTcpReader() io.Reader {
	return &tcpReader{
		command: c,
		buffer:  make([]byte, 4096), // 使用4KB的缓冲区
	}
}

// Read 实现io.Reader接口
func (r *tcpReader) Read(p []byte) (n int, err error) {
	// 从命令读取数据
	data, err := r.command.reader(len(p))
	if err != nil {
		if err.Error() == "EOF" {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("读取TCP数据失败: %v", err)
	}

	// 如果没有数据返回，表示结束
	if len(data) == 0 {
		return 0, io.EOF
	}

	// 复制数据到目标缓冲区
	return copy(p, []byte(data)), nil
}

// Close 关闭TCP读取器（如果需要）
func (r *tcpReader) Close() error {
	// 实现任何必要的清理逻辑
	return nil
}
