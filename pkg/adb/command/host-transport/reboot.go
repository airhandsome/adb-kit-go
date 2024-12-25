package hosttransport

import (
	"fmt"
)

// RebootCommand 实现重启命令
type RebootCommand struct {
	BaseCommand
}

// NewRebootCommand 创建新的重启命令实例
func NewRebootCommand(sender func(string) error, reader func(int) (string, error)) *RebootCommand {
	return &RebootCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行重启命令
func (c *RebootCommand) Execute() error {
	// 发送重启命令
	if err := c.sender("reboot:"); err != nil {
		return fmt.Errorf("发送重启命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取所有剩余数据
		_, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取剩余数据失败: %v", err)
		}
		return nil

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取错误信息失败: %v", err)
		}
		return fmt.Errorf(errMsg)

	default:
		return fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// RebootMode 定义重启模式
type RebootMode string

const (
	// RebootNormal 正常重启
	RebootNormal RebootMode = ""
	// RebootRecovery 重启到恢复模式
	RebootRecovery RebootMode = "recovery"
	// RebootBootloader 重启到引导加载程序
	RebootBootloader RebootMode = "bootloader"
)

// ExecuteWithMode 执行带模式的重启命令
func (c *RebootCommand) ExecuteWithMode(mode RebootMode) error {
	var cmd string
	if mode == RebootNormal {
		cmd = "reboot:"
	} else {
		cmd = fmt.Sprintf("reboot:%s", mode)
	}

	// 发送重启命令
	if err := c.sender(cmd); err != nil {
		return fmt.Errorf("发送重启命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取所有剩余数据
		_, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取剩余数据失败: %v", err)
		}
		return nil

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取错误信息失败: %v", err)
		}
		return fmt.Errorf(errMsg)

	default:
		return fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}
