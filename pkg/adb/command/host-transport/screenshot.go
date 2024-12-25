package hosttransport

import (
	"fmt"
	"io"
)

// ScreencapCommand 实现屏幕截图命令
type ScreencapCommand struct {
	BaseCommand
}

// NewScreencapCommand 创建新的屏幕截图命令实例
func NewScreencapCommand(sender func(string) error, reader func(int) (string, error)) *ScreencapCommand {
	return &ScreencapCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行屏幕截图命令
func (c *ScreencapCommand) Execute() (io.Reader, error) {
	// 发送命令，重定向stderr到/dev/null以避免错误信息
	if err := c.sender("shell:echo && screencap -p 2>/dev/null"); err != nil {
		return nil, fmt.Errorf("发送屏幕截图命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取第一个字节来检测行结束符
		firstByte, err := c.reader(1)
		if err != nil {
			if err.Error() == "EOF" {
				return nil, fmt.Errorf("设备不支持screencap命令")
			}
			return nil, fmt.Errorf("读取数据失败: %v", err)
		}

		// 创建新的转换器
		transform := NewLineTransform(firstByte, true)
		return transform, nil

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

// LineTransform 实现行结束符转换
type LineTransform struct {
	command    *ScreencapCommand
	buffer     []byte
	autoDetect bool
	firstByte  string
}

// NewLineTransform 创建新的行转换器
func NewLineTransform(firstByte string, autoDetect bool) *LineTransform {
	return &LineTransform{
		buffer:     make([]byte, 4096),
		autoDetect: autoDetect,
		firstByte:  firstByte,
	}
}

// Read 实现io.Reader接口
func (t *LineTransform) Read(p []byte) (n int, err error) {
	// 如果有第一个字节，先处理它
	if len(t.firstByte) > 0 {
		n = copy(p, []byte(t.firstByte))
		t.firstByte = ""
		return n, nil
	}

	// 从命令读取数据
	data, err := t.command.reader(len(p))
	if err != nil {
		if err.Error() == "EOF" {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("读取数据失败: %v", err)
	}

	// 如果没有数据返回，表示结束
	if len(data) == 0 {
		return 0, io.EOF
	}

	// 如果需要自动检测和转换行结束符
	if t.autoDetect {
		// 转换行结束符
		data = t.convertLineEndings(data)
	}

	// 复制转换后的数据到目标缓冲区
	return copy(p, []byte(data)), nil
}

// convertLineEndings 转换行结束符
func (t *LineTransform) convertLineEndings(data string) string {
	// 根据需要实现行结束符转换逻辑
	// 这里可以添加检测和转换CRLF/LF的代码
	return data
}

// Write 实现写入功能（如果需要）
func (t *LineTransform) Write(p []byte) (n int, err error) {
	// 实现写入逻辑（如果需要）
	return len(p), nil
}

// Close 关闭转换器（如果需要）
func (t *LineTransform) Close() error {
	// 实现清理逻辑（如果需要）
	return nil
}
