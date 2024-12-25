package hosttransport

import (
	"fmt"
	"io"
)

// LogCommand 实现日志命令
type LogCommand struct {
	BaseCommand
}

// NewLogCommand 创建新的日志命令实例
func NewLogCommand(sender func(string) error, reader func(int) (string, error)) *LogCommand {
	return &LogCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行日志命令
func (c *LogCommand) Execute(name string) (io.Reader, error) {
	// 构建日志命令
	cmd := fmt.Sprintf("log:%s", name)

	if err := c.sender(cmd); err != nil {
		return nil, fmt.Errorf("发送日志命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 创建一个读取器来处理日志流
		return c.createLogReader(), nil

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

// logReader 实现io.Reader接口，用于读取日志流
type logReader struct {
	command *LogCommand
	buffer  []byte
}

// createLogReader 创建新的日志读取器
func (c *LogCommand) createLogReader() io.Reader {
	return &logReader{
		command: c,
		buffer:  make([]byte, 4096), // 使用4KB的缓冲区
	}
}

// Read 实现io.Reader接口
func (r *logReader) Read(p []byte) (n int, err error) {
	// 从命令读取数据
	data, err := r.command.reader(len(p))
	if err != nil {
		if err.Error() == "EOF" {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("读取日志数据失败: %v", err)
	}

	// 如果没有数据返回，表示结束
	if len(data) == 0 {
		return 0, io.EOF
	}

	// 复制数据到目标缓冲区
	return copy(p, []byte(data)), nil
}

// Close 关闭日志读取器（如果需要）
func (r *logReader) Close() error {
	// 实现任何必要的清理逻辑
	return nil
}
