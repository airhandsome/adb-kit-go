package hosttransport

import (
	"fmt"
	"strings"
	"time"
)

// RemountCommand 实现重新挂载命令
type RemountCommand struct {
	BaseCommand
}

// NewRemountCommand 创建新的重新挂载命令实例
func NewRemountCommand(sender func(string) error, reader func(int) (string, error)) *RemountCommand {
	return &RemountCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行重新挂载命令
func (c *RemountCommand) Execute() error {
	// 发送重新挂载命令
	if err := c.sender("remount:"); err != nil {
		return fmt.Errorf("发送重新挂载命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		return nil

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取错误信息失败: %v", err)
		}
		return fmt.Errorf("remount failed: %s", errMsg)

	default:
		return fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// ExecuteWithRetry 执行重新挂载命令并在需要时重试
func (c *RemountCommand) ExecuteWithRetry(maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := c.Execute()
		if err == nil {
			return nil
		}
		lastErr = err

		// 等待一小段时间后重试
		time.Sleep(time.Second)
	}
	return fmt.Errorf("remount failed after %d attempts: %v", maxRetries, lastErr)
}

// ExecuteWithVerification 执行重新挂载命令并验证结果
func (c *RemountCommand) ExecuteWithVerification() error {
	// 首先执行重新挂载
	if err := c.Execute(); err != nil {
		return err
	}

	// 验证挂载状态
	// 这里可以添加额外的验证逻辑，比如检查 mount 命令输出
	cmd := "shell:mount | grep system"
	if err := c.sender(cmd); err != nil {
		return fmt.Errorf("验证挂载状态失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取验证响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		output, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取验证输出失败: %v", err)
		}

		// 检查输出中是否包含 "rw" 标志
		if !strings.Contains(output, " rw,") {
			return fmt.Errorf("system 分区未成功重新挂载为读写模式")
		}
		return nil

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取验证错误信息失败: %v", err)
		}
		return fmt.Errorf("验证失败: %s", errMsg)

	default:
		return fmt.Errorf("unexpected verification response: %s", reply)
	}
}
