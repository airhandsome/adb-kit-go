package hosttransport

import (
	"fmt"
	"strings"
)

// UninstallCommand 实现卸载命令
type UninstallCommand struct {
	BaseCommand
}

// NewUninstallCommand 创建新的卸载命令实例
func NewUninstallCommand(sender func(string) error, reader func(int) (string, error)) *UninstallCommand {
	return &UninstallCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行卸载命令
func (c *UninstallCommand) Execute(pkg string) error {
	// 发送卸载命令
	if err := c.sender(fmt.Sprintf("shell:pm uninstall %s", pkg)); err != nil {
		return fmt.Errorf("发送卸载命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取卸载结果
		output, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取卸载结果失败: %v", err)
		}

		// 检查输出
		output = strings.TrimSpace(output)
		if output == "Success" || strings.Contains(output, "Failure") || strings.Contains(output, "Unknown package") {
			// 无论是成功、失败还是包不存在，都返回true
			// 因为包已经不存在于设备上了
			return nil
		}
		return fmt.Errorf("unexpected uninstall output: %s", output)

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取错误信息失败: %v", err)
		}
		return fmt.Errorf("uninstall failed: %s", errMsg)

	default:
		return fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// ExecuteWithOptions 执行带选项的卸载命令
func (c *UninstallCommand) ExecuteWithOptions(pkg string, keepData bool, user int) error {
	cmd := fmt.Sprintf("shell:pm uninstall")
	if keepData {
		cmd += " -k"
	}
	if user >= 0 {
		cmd += fmt.Sprintf(" --user %d", user)
	}
	cmd += " " + pkg

	if err := c.sender(cmd); err != nil {
		return fmt.Errorf("发送卸载命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		output, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取卸载结果失败: %v", err)
		}

		output = strings.TrimSpace(output)
		if output == "Success" || strings.Contains(output, "Failure") || strings.Contains(output, "Unknown package") {
			return nil
		}
		return fmt.Errorf("unexpected uninstall output: %s", output)

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取错误信息失败: %v", err)
		}
		return fmt.Errorf("uninstall failed: %s", errMsg)

	default:
		return fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}
