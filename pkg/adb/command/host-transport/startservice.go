package hosttransport

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// StartServiceCommand 实现启动服务命令
type StartServiceCommand struct {
	StartActivityCommand // 继承自StartActivityCommand以复用intentArgs等方法
}

// NewStartServiceCommand 创建新的启动服务命令实例
func NewStartServiceCommand(sender func(string) error, reader func(int) (string, error)) *StartServiceCommand {
	return &StartServiceCommand{
		StartActivityCommand: StartActivityCommand{
			BaseCommand: BaseCommand{
				sender: sender,
				reader: reader,
			},
		},
	}
}

// Execute 执行启动服务命令
func (c *StartServiceCommand) Execute(options map[string]interface{}) error {
	// 获取intent参数
	args := c.intentArgs(options)

	// 添加用户参数（如果存在）
	if user, ok := options["user"].(int); ok {
		args = append(args, "--user", fmt.Sprintf("%d", user))
	}

	// 构建启动服务命令
	cmd := fmt.Sprintf("shell:am startservice %s", strings.Join(args, " "))

	if err := c.sender(cmd); err != nil {
		return fmt.Errorf("发送启动服务命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取并检查服务启动结果
		output, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取服务启动结果失败: %v", err)
		}

		// 检查输出是否包含错误信息
		if strings.Contains(output, "Error:") {
			return fmt.Errorf("启动服务失败: %s", strings.TrimSpace(output))
		}
		return nil

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取错误信息失败: %v", err)
		}
		return fmt.Errorf("启动服务失败: %s", errMsg)

	default:
		return fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// ExecuteWithTimeout 执行启动服务命令并设置超时
func (c *StartServiceCommand) ExecuteWithTimeout(options map[string]interface{}, timeout time.Duration) error {
	// 创建一个带有超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 创建一个错误通道
	done := make(chan error, 1)

	// 在goroutine中执行命令
	go func() {
		done <- c.Execute(options)
	}()

	// 等待命令完成或超时
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("启动服务超时: %v", ctx.Err())
	}
}

// ExecuteWithRetry 执行启动服务命令并在失败时重试
func (c *StartServiceCommand) ExecuteWithRetry(options map[string]interface{}, maxRetries int, retryDelay time.Duration) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := c.Execute(options)
		if err == nil {
			return nil
		}
		lastErr = err

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}
	return fmt.Errorf("启动服务失败（重试%d次后）: %v", maxRetries, lastErr)
}
