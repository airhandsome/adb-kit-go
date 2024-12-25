package hosttransport

import (
	"fmt"
	"strings"
)

// TcpIpCommand 实现TCP/IP命令
type TcpIpCommand struct {
	BaseCommand
}

// NewTcpIpCommand 创建新的TCP/IP命令实例
func NewTcpIpCommand(sender func(string) error, reader func(int) (string, error)) *TcpIpCommand {
	return &TcpIpCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行TCP/IP命令
func (c *TcpIpCommand) Execute(port int) (int, error) {
	// 发送TCP/IP命令
	cmd := fmt.Sprintf("tcpip:%d", port)
	if err := c.sender(cmd); err != nil {
		return 0, fmt.Errorf("发送TCP/IP命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return 0, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取完整响应
		response, err := c.reader(0)
		if err != nil {
			return 0, fmt.Errorf("读取响应数据失败: %v", err)
		}

		// 检查响应是否包含成功消息
		if strings.Contains(response, "restarting in") {
			return port, nil
		}
		return 0, fmt.Errorf("TCP/IP命令失败: %s", strings.TrimSpace(response))

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return 0, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return 0, fmt.Errorf("TCP/IP命令失败: %s", errMsg)

	default:
		return 0, fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}
