package hosttransport

import (
	"fmt"
	"strings"
)

// IsInstalledCommand 实现检查包是否已安装的命令
type IsInstalledCommand struct {
	BaseCommand
}

// NewIsInstalledCommand 创建新的检查安装命令实例
func NewIsInstalledCommand(sender func(string) error, reader func(int) (string, error)) *IsInstalledCommand {
	return &IsInstalledCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行检查包是否已安装命令
func (c *IsInstalledCommand) Execute(pkg string) (bool, error) {
	// 发送命令，重定向stderr到/dev/null以避免错误信息
	cmd := fmt.Sprintf("shell:pm path %s 2>/dev/null", pkg)
	if err := c.sender(cmd); err != nil {
		return false, fmt.Errorf("发送检查安装命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return false, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 尝试读取"package:"前缀
		prefix, err := c.reader(8)
		if err != nil {
			// 如果读取失败，可能是因为包未安装
			return false, nil
		}

		// 检查前缀是否为"package:"
		if prefix == "package:" {
			return true, nil
		}

		// 如果前缀不匹配，返回意外响应错误
		return false, fmt.Errorf("unexpected response prefix: %s, expected 'package:'", prefix)

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return false, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return false, fmt.Errorf(errMsg)

	default:
		return false, fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// readUntilEOF 读取直到遇到EOF
func (c *IsInstalledCommand) readUntilEOF() (string, error) {
	var builder strings.Builder
	buffer := make([]byte, 1024)

	for {
		n, err := c.reader(len(buffer))
		if err != nil {
			// 如果是EOF，返回已读取的内容
			if err.Error() == "EOF" {
				return builder.String(), nil
			}
			return "", err
		}

		if n == "" {
			break
		}

		builder.WriteString(n)
	}

	return builder.String(), nil
}
