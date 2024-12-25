package hosttransport

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// MonkeyCommand 实现Monkey测试命令
type MonkeyCommand struct {
	BaseCommand
}

// NewMonkeyCommand 创建新的Monkey命令实例
func NewMonkeyCommand(sender func(string) error, reader func(int) (string, error)) *MonkeyCommand {
	return &MonkeyCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行Monkey命令
func (c *MonkeyCommand) Execute(port int) (io.Reader, error) {
	// 设置环境变量以修复一些设备上的/sdcard问题
	// 一些设备的/sdcard路径有问题（如/mnt/sdcard），monkey会尝试在那里写入日志
	// 通过设置EXTERNAL_STORAGE环境变量，我们可以改变日志写入位置
	cmd := fmt.Sprintf("shell:EXTERNAL_STORAGE=/data/local/tmp monkey --port %d -v", port)

	if err := c.sender(cmd); err != nil {
		return nil, fmt.Errorf("发送monkey命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 创建一个reader来处理monkey输出
		return c.createMonkeyReader()

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

// monkeyReader 实现io.Reader接口，用于读取monkey输出
type monkeyReader struct {
	command     *MonkeyCommand
	buffer      []byte
	initialized bool
}

// createMonkeyReader 创建新的monkey读取器
func (c *MonkeyCommand) createMonkeyReader() (io.Reader, error) {
	reader := &monkeyReader{
		command:     c,
		buffer:      make([]byte, 4096), // 使用4KB的缓冲区
		initialized: false,
	}

	// 等待初始化消息
	err := reader.waitForInitialization()
	if err != nil {
		return nil, err
	}

	return reader, nil
}

// waitForInitialization 等待monkey初始化
func (r *monkeyReader) waitForInitialization() error {
	// 创建一个通道用于超时控制
	done := make(chan error, 1)

	go func() {
		var buffer string
		for {
			// 读取数据直到发现初始化标记
			data, err := r.command.reader(1024)
			if err != nil {
				done <- fmt.Errorf("读取初始化数据失败: %v", err)
				return
			}

			buffer += data
			if len(buffer) > 0 {
				// 检查是否包含初始化标记 ":Monkey:"
				if strings.Contains(buffer, ":Monkey:") {
					done <- nil
					return
				}
			}
		}
	}()

	// 设置1秒超时
	select {
	case err := <-done:
		return err
	case <-time.After(time.Second):
		// 某些设备（如富士通F-08D）在任何情况下都不会输出monkey信息
		// 使用超时作为后备方案
		return nil
	}
}

// Read 实现io.Reader接口
func (r *monkeyReader) Read(p []byte) (n int, err error) {
	// 从命令读取数据
	data, err := r.command.reader(len(p))
	if err != nil {
		if err.Error() == "EOF" {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("读取monkey数据失败: %v", err)
	}

	// 如果没有数据返回，表示结束
	if len(data) == 0 {
		return 0, io.EOF
	}

	// 复制数据到目标缓冲区
	return copy(p, []byte(data)), nil
}

// Close 关闭monkey读取器（如果需要）
func (r *monkeyReader) Close() error {
	// 实现任何必要的清理逻辑
	return nil
}
