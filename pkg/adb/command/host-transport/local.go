package hosttransport

import (
	"fmt"
	"strings"
)

// LocalCommand 实现本地文件系统命令
type LocalCommand struct {
	BaseCommand
}

// NewLocalCommand 创建新的本地命令实例
func NewLocalCommand(sender func(string) error, reader func(int) (string, error)) *LocalCommand {
	return &LocalCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行本地文件系统命令
func (c *LocalCommand) Execute(path string) (string, error) {
	// 构建命令，根据路径格式选择命令
	var cmd string
	if strings.Contains(path, ":") {
		cmd = path
	} else {
		cmd = fmt.Sprintf("localfilesystem:%s", path)
	}

	if err := c.sender(cmd); err != nil {
		return "", fmt.Errorf("发送本地命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取原始数据
		data, err := c.reader(0)
		if err != nil {
			return "", fmt.Errorf("读取数据失败: %v", err)
		}
		return data, nil

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return "", fmt.Errorf("读取错误信息失败: %v", err)
		}
		return "", fmt.Errorf(errMsg)

	default:
		return "", fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}
