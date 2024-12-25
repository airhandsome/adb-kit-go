package hosttransport

import (
	"fmt"
)

// ReverseCommand 实现反向端口转发命令
type ReverseCommand struct {
	BaseCommand
}

// NewReverseCommand 创建新的反向端口转发命令实例
func NewReverseCommand(sender func(string) error, reader func(int) (string, error)) *ReverseCommand {
	return &ReverseCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行反向端口转发命令
func (c *ReverseCommand) Execute(remote, local string) error {
	// 构建反向端口转发命令
	cmd := fmt.Sprintf("reverse:forward:%s;%s", remote, local)

	if err := c.sender(cmd); err != nil {
		return fmt.Errorf("发送反向端口转发命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取确认响应
		confirm, err := c.reader(4)
		if err != nil {
			return fmt.Errorf("读取确认响应失败: %v", err)
		}

		switch confirm {
		case OKAY:
			return nil
		case FAIL:
			errMsg, err := c.reader(0)
			if err != nil {
				return fmt.Errorf("读取错误信息失败: %v", err)
			}
			return fmt.Errorf("反向端口转发失败: %s", errMsg)
		default:
			return fmt.Errorf("unexpected confirmation response: %s, expected OKAY or FAIL", confirm)
		}

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取错误信息失败: %v", err)
		}
		return fmt.Errorf("反向端口转发失败: %s", errMsg)

	default:
		return fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}
