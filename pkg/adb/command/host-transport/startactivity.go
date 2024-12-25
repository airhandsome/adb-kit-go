package hosttransport

import (
	"fmt"
	"strings"
)

// StartActivityCommand 实现启动活动命令
type StartActivityCommand struct {
	BaseCommand
}

// NewStartActivityCommand 创建新的启动活动命令实例
func NewStartActivityCommand(sender func(string) error, reader func(int) (string, error)) *StartActivityCommand {
	return &StartActivityCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行启动活动命令
func (c *StartActivityCommand) Execute(options map[string]interface{}) error {
	args := c.intentArgs(options)

	// 构建启动命令
	cmd := fmt.Sprintf("shell:am start %s", strings.Join(args, " "))

	if err := c.sender(cmd); err != nil {
		return fmt.Errorf("发送启动活动命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取所有剩余数据
		_, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取剩余数据失败: %v", err)
		}
		return nil

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取错误信息失败: %v", err)
		}
		return fmt.Errorf("启动活动失败: %s", errMsg)

	default:
		return fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// intentArgs 生成启动活动的参数
func (c *StartActivityCommand) intentArgs(options map[string]interface{}) []string {
	var args []string

	if extras, ok := options["extras"].(map[string]interface{}); ok {
		args = append(args, c.formatExtras(extras)...)
	}
	if action, ok := options["action"].(string); ok {
		args = append(args, "-a", c.escape(action))
	}
	if data, ok := options["data"].(string); ok {
		args = append(args, "-d", c.escape(data))
	}
	if mimeType, ok := options["mimeType"].(string); ok {
		args = append(args, "-t", c.escape(mimeType))
	}
	if category, ok := options["category"].([]string); ok {
		for _, cat := range category {
			args = append(args, "-c", c.escape(cat))
		}
	}
	if component, ok := options["component"].(string); ok {
		args = append(args, "-n", c.escape(component))
	}
	if flags, ok := options["flags"].(string); ok {
		args = append(args, "-f", c.escape(flags))
	}
	if debug, ok := options["debug"].(bool); ok && debug {
		args = append(args, "-D")
	}
	if wait, ok := options["wait"].(bool); ok && wait {
		args = append(args, "-W")
	}
	if user, ok := options["user"].(int); ok {
		args = append(args, "--user", fmt.Sprintf("%d", user))
	}

	return args
}

// formatExtras 格式化额外参数
func (c *StartActivityCommand) formatExtras(extras map[string]interface{}) []string {
	var args []string
	for key, value := range extras {
		switch v := value.(type) {
		case string:
			args = append(args, fmt.Sprintf("--es %s %s", c.escape(key), c.escape(v)))
		case bool:
			args = append(args, fmt.Sprintf("--ez %s %t", c.escape(key), v))
		case int:
			args = append(args, fmt.Sprintf("--ei %s %d", c.escape(key), v))
		case float64:
			args = append(args, fmt.Sprintf("--ef %s %f", c.escape(key), v))
		case nil:
			args = append(args, fmt.Sprintf("--esn %s", c.escape(key)))
		default:
			// Handle other types or throw an error
		}
	}
	return args
}

// escape 转义命令中的特殊字符
func (c *StartActivityCommand) escape(input string) string {
	// 实现转义逻辑
	// 这里需要根据具体需求实现，例如：
	escaped := strings.ReplaceAll(input, " ", "\\ ")
	escaped = strings.ReplaceAll(escaped, "(", "\\(")
	escaped = strings.ReplaceAll(escaped, ")", "\\)")
	escaped = strings.ReplaceAll(escaped, "[", "\\[")
	escaped = strings.ReplaceAll(escaped, "]", "\\]")
	escaped = strings.ReplaceAll(escaped, "&", "\\&")
	escaped = strings.ReplaceAll(escaped, "|", "\\|")
	escaped = strings.ReplaceAll(escaped, ";", "\\;")
	escaped = strings.ReplaceAll(escaped, "<", "\\<")
	escaped = strings.ReplaceAll(escaped, ">", "\\>")
	escaped = strings.ReplaceAll(escaped, "$", "\\$")
	escaped = strings.ReplaceAll(escaped, "`", "\\`")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")
	return escaped
}
