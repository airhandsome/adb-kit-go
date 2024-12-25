package hosttransport

import (
	"fmt"
	"io"
	"strings"
)

// LogcatCommand 实现logcat命令
type LogcatCommand struct {
	BaseCommand
}

// LogcatOptions 定义logcat命令的选项
type LogcatOptions struct {
	Clear bool // 是否在开始前清除日志
}

// NewLogcatCommand 创建新的logcat命令实例
func NewLogcatCommand(sender func(string) error, reader func(int) (string, error)) *LogcatCommand {
	return &LogcatCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行logcat命令
func (c *LogcatCommand) Execute(options *LogcatOptions) (io.Reader, error) {
	if options == nil {
		options = &LogcatOptions{}
	}

	// 构建命令
	// 注意：LG G Flex需要-B选项带过滤器，虽然实际上不使用
	cmd := "shell:echo && logcat -B *:I 2>/dev/null"
	if options.Clear {
		cmd = "shell:echo && logcat -c 2>/dev/null && " + cmd
	}

	if err := c.sender(cmd); err != nil {
		return nil, fmt.Errorf("发送logcat命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 创建一个转换器来处理日志流
		return c.createLogcatReader(), nil

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

// logcatReader 实现io.Reader接口，用于读取和转换logcat输出
type logcatReader struct {
	command    *LogcatCommand
	buffer     []byte
	lineBuffer string
}

// createLogcatReader 创建新的logcat读取器
func (c *LogcatCommand) createLogcatReader() io.Reader {
	return &logcatReader{
		command:    c,
		buffer:     make([]byte, 4096), // 使用4KB的缓冲区
		lineBuffer: "",
	}
}

// Read 实现io.Reader接口
func (r *logcatReader) Read(p []byte) (n int, err error) {
	// 如果行缓冲区中有数据，先处理它
	if len(r.lineBuffer) > 0 {
		n = copy(p, r.lineBuffer)
		r.lineBuffer = r.lineBuffer[n:]
		return n, nil
	}

	// 从命令读取新数据
	data, err := r.command.reader(len(p))
	if err != nil {
		if err.Error() == "EOF" {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("读取logcat数据失败: %v", err)
	}

	// 如果没有数据返回，表示结束
	if len(data) == 0 {
		return 0, io.EOF
	}

	// 处理数据，确保完整的行
	fullData := r.lineBuffer + data
	lines := strings.Split(fullData, "\n")

	// 保存最后一个不完整的行（如果有）
	if !strings.HasSuffix(fullData, "\n") {
		r.lineBuffer = lines[len(lines)-1]
		lines = lines[:len(lines)-1]
	} else {
		r.lineBuffer = ""
	}

	// 合并完整的行
	processedData := strings.Join(lines, "\n")
	if len(processedData) > 0 {
		processedData += "\n"
	}

	// 复制处理后的数据到目标缓冲区
	return copy(p, []byte(processedData)), nil
}

// Close 关闭logcat读取器（如果需要）
func (r *logcatReader) Close() error {
	// 实现任何必要的清理逻辑
	return nil
}
