package hosttransport

import (
	"fmt"
	"regexp"
	"strings"
)

// GetPackagesCommand 实现获取包列表命令
type GetPackagesCommand struct {
	BaseCommand
}

// NewGetPackagesCommand 创建新的获取包列表命令实例
func NewGetPackagesCommand(sender func(string) error, reader func(int) (string, error)) *GetPackagesCommand {
	return &GetPackagesCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行获取包列表命令
func (c *GetPackagesCommand) Execute() ([]string, error) {
	// 发送命令，重定向stderr到/dev/null以避免错误信息
	if err := c.sender("shell:pm list packages 2>/dev/null"); err != nil {
		return nil, fmt.Errorf("发送获取包列表命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取所有数据
		data, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取包列表数据失败: %v", err)
		}
		return c.parsePackages(data)

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

// parsePackages 解析包列表数据
func (c *GetPackagesCommand) parsePackages(value string) ([]string, error) {
	packages := make([]string, 0)

	// 编译正则表达式
	re := regexp.MustCompile(`^package:(.*?)\r?$`)

	// 按行分割
	lines := strings.Split(value, "\n")

	// 处理每一行
	for _, line := range lines {
		// 跳过空行
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		// 匹配包名
		matches := re.FindStringSubmatch(line)
		if len(matches) == 2 {
			// matches[1] 包含包名
			packageName := strings.TrimSpace(matches[1])
			if packageName != "" {
				packages = append(packages, packageName)
			}
		}
	}

	return packages, nil
}
