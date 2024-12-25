package hosttransport

import (
	"fmt"
	"io"
	"strings"
)

// ShellCommand 实现shell命令
type ShellCommand struct {
	BaseCommand
}

// NewShellCommand 创建新的shell命令实例
func NewShellCommand(sender func(string) error, reader func(int) (string, error)) *ShellCommand {
	return &ShellCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行shell命令
func (c *ShellCommand) Execute(command interface{}) (io.Reader, error) {
	var cmd string

	// 检查命令是否为字符串数组
	switch v := command.(type) {
	case []string:
		// 转义并连接命令
		escapedCommands := make([]string, len(v))
		for i, part := range v {
			escapedCommands[i] = c.escape(part)
		}
		cmd = strings.Join(escapedCommands, " ")
	case string:
		cmd = v
	default:
		return nil, fmt.Errorf("不支持的命令类型: %T", command)
	}

	// 发送shell命令
	if err := c.sender(fmt.Sprintf("shell:%s", cmd)); err != nil {
		return nil, fmt.Errorf("发送shell命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 返回一个读取器来处理shell输出
		return c.createShellReader(), nil

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return nil, fmt.Errorf("shell命令失败: %s", errMsg)

	default:
		return nil, fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// shellReader 实现io.Reader接口，用于读取shell输出
type shellReader struct {
	command *ShellCommand
	buffer  []byte
}

// createShellReader 创建新的shell读取器
func (c *ShellCommand) createShellReader() io.Reader {
	return &shellReader{
		command: c,
		buffer:  make([]byte, 4096), // 使用4KB的缓冲区
	}
}

// Read 实现io.Reader接口
func (r *shellReader) Read(p []byte) (n int, err error) {
	// 从命令读取数据
	data, err := r.command.reader(len(p))
	if err != nil {
		if err.Error() == "EOF" {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("读取shell数据失败: %v", err)
	}

	// 如果没有数据返回，表示结束
	if len(data) == 0 {
		return 0, io.EOF
	}

	// 复制数据到目标缓冲区
	return copy(p, []byte(data)), nil
}

// escape 转义命令中的特殊字符
func (c *ShellCommand) escape(input string) string {
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
