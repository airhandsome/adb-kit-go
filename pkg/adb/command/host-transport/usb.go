package hosttransport

import (
	"fmt"
	"strings"
)

// UsbCommand 实现USB命令
type UsbCommand struct {
	BaseCommand
}

// NewUsbCommand 创建新的USB命令实例
func NewUsbCommand(sender func(string) error, reader func(int) (string, error)) *UsbCommand {
	return &UsbCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行USB命令
func (c *UsbCommand) Execute() (bool, error) {
	// 发送USB命令
	if err := c.sender("usb:"); err != nil {
		return false, fmt.Errorf("发送USB命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return false, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取完整响应
		response, err := c.reader(0)
		if err != nil {
			return false, fmt.Errorf("读取响应数据失败: %v", err)
		}

		// 检查响应是否包含成功消息
		if strings.Contains(response, "restarting in") {
			return true, nil
		}
		return false, fmt.Errorf("USB命令失败: %s", strings.TrimSpace(response))

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return false, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return false, fmt.Errorf("USB命令失败: %s", errMsg)

	default:
		return false, fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}
