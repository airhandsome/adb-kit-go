package hosttransport

import (
	"fmt"
	"regexp"
	"strings"
)

// GetPropertiesCommand 实现获取系统属性命令
type GetPropertiesCommand struct {
	BaseCommand
}

// NewGetPropertiesCommand 创建新的获取系统属性命令实例
func NewGetPropertiesCommand(sender func(string) error, reader func(int) (string, error)) *GetPropertiesCommand {
	return &GetPropertiesCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行获取系统属性命令
func (c *GetPropertiesCommand) Execute() (map[string]string, error) {
	if err := c.sender("shell:getprop"); err != nil {
		return nil, fmt.Errorf("发送获取属性命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		data, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取属性数据失败: %v", err)
		}
		return c.parseProperties(data)

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

// parseProperties 解析系统属性数据
func (c *GetPropertiesCommand) parseProperties(value string) (map[string]string, error) {
	properties := make(map[string]string)

	// 编译正则表达式，匹配属性格式 [key]: [value]
	re := regexp.MustCompile(`^\[([\s\S]*?)\]: \[([\s\S]*?)\]\r?$`)

	// 按行分割
	lines := strings.Split(value, "\n")

	// 处理每一行
	for _, line := range lines {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			key := strings.TrimSpace(matches[1])
			value := strings.TrimSpace(matches[2])
			properties[key] = value
		}
	}

	return properties, nil
}
