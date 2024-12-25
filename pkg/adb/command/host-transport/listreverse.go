package hosttransport

import (
	"fmt"
	"strings"
)

// Reverse 表示一个反向端口转发配置
type Reverse struct {
	Remote string
	Local  string
}

// ListReversesCommand 实现列出反向端口转发命令
type ListReversesCommand struct {
	BaseCommand
}

// NewListReversesCommand 创建新的列出反向端口转发命令实例
func NewListReversesCommand(sender func(string) error, reader func(int) (string, error)) *ListReversesCommand {
	return &ListReversesCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行列出反向端口转发命令
func (c *ListReversesCommand) Execute() ([]Reverse, error) {
	if err := c.sender("reverse:list-forward"); err != nil {
		return nil, fmt.Errorf("发送列出反向端口转发命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取值
		value, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取反向端口转发列表失败: %v", err)
		}
		return c.parseReverses(value)

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return nil, fmt.Errorf(errMsg)

	default:
		return nil, fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// parseReverses 解析反向端口转发列表
func (c *ListReversesCommand) parseReverses(value string) ([]Reverse, error) {
	reverses := make([]Reverse, 0)

	// 按行分割
	lines := strings.Split(strings.TrimSpace(value), "\n")

	// 处理每一行
	for _, line := range lines {
		// 跳过空行
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		// 分割行内容
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue // 跳过格式不正确的行
		}

		// parts[0] 是序列号，我们不需要它
		// parts[1] 是远程地址
		// parts[2] 是本地地址
		reverse := Reverse{
			Remote: parts[1],
			Local:  parts[2],
		}

		reverses = append(reverses, reverse)
	}

	return reverses, nil
}
