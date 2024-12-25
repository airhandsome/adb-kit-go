package hosttransport

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// RootCommand 实现root权限命令
type RootCommand struct {
	BaseCommand
}

// NewRootCommand 创建新的root命令实例
func NewRootCommand(sender func(string) error, reader func(int) (string, error)) *RootCommand {
	return &RootCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行root命令
func (c *RootCommand) Execute() error {
	// 发送root命令
	if err := c.sender("root:"); err != nil {
		return fmt.Errorf("发送root命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取完整响应
		response, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取root响应失败: %v", err)
		}

		// 检查是否包含成功重启的消息
		if c.isRestartingAsRoot(response) {
			return nil
		}

		// 如果响应不包含预期消息，返回错误
		return fmt.Errorf("root失败: %s", strings.TrimSpace(response))

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取错误信息失败: %v", err)
		}
		return fmt.Errorf("root失败: %s", errMsg)

	default:
		return fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// isRestartingAsRoot 检查响应是否表明正在以root权限重启
func (c *RootCommand) isRestartingAsRoot(response string) bool {
	re := regexp.MustCompile(`restarting adbd as root`)
	return re.MatchString(response)
}

// ExecuteWithVerification 执行root命令并验证结果
func (c *RootCommand) ExecuteWithVerification() error {
	// 首先执行root命令
	if err := c.Execute(); err != nil {
		return err
	}

	// 等待一段时间让adbd重启
	time.Sleep(time.Second * 2)

	// 验证root状态
	if err := c.verifyRoot(); err != nil {
		return fmt.Errorf("root验证失败: %v", err)
	}

	return nil
}

// verifyRoot 验证是否成功获取root权限
func (c *RootCommand) verifyRoot() error {
	// 发送验证命令
	if err := c.sender("shell:id"); err != nil {
		return fmt.Errorf("发送验证命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取验证响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取id命令输出
		output, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取验证输出失败: %v", err)
		}

		// 检查输出是否包含 uid=0(root)
		if !strings.Contains(output, "uid=0(root)") {
			return fmt.Errorf("未获取root权限")
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
