package hosttransport

import (
	"fmt"
	"time"
)

// WaitBootCompleteCommand 实现等待启动完成命令
type WaitBootCompleteCommand struct {
	BaseCommand
}

// NewWaitBootCompleteCommand 创建新的等待启动完成命令实例
func NewWaitBootCompleteCommand(sender func(string) error, reader func(int) (string, error)) *WaitBootCompleteCommand {
	return &WaitBootCompleteCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行等待启动完成命令
func (c *WaitBootCompleteCommand) Execute() error {
	// 发送等待启动完成命令
	cmd := "shell:while getprop sys.boot_completed 2>/dev/null; do sleep 1; done"
	if err := c.sender(cmd); err != nil {
		return fmt.Errorf("发送等待启动完成命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 等待直到收到 "1" 表示启动完成
		for {
			output, err := c.reader(0)
			if err != nil {
				return fmt.Errorf("读取启动状态失败: %v", err)
			}

			if output == "1\n" {
				return nil
			}

			// 短暂休眠避免过度消耗CPU
			time.Sleep(100 * time.Millisecond)
		}

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取错误信息失败: %v", err)
		}
		return fmt.Errorf("等待启动完成失败: %s", errMsg)

	default:
		return fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// ExecuteWithTimeout 执行带超时的等待启动完成命令
func (c *WaitBootCompleteCommand) ExecuteWithTimeout(timeout time.Duration) error {
	done := make(chan error)
	go func() {
		done <- c.Execute()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("等待启动完成超时: %v", timeout)
	}
}

// ExecuteWithCallback 执行带回调的等待启动完成命令
func (c *WaitBootCompleteCommand) ExecuteWithCallback(callback func(bool)) error {
	err := c.Execute()
	if err != nil {
		callback(false)
		return err
	}
	callback(true)
	return nil
}
