package hosttransport

import (
	"fmt"
	"io"
)

// SyncCommand 实现同步命令
type SyncCommand struct {
	BaseCommand
}

// NewSyncCommand 创建新的同步命令实例
func NewSyncCommand(sender func(string) error, reader func(int) (string, error)) *SyncCommand {
	return &SyncCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行同步命令
func (c *SyncCommand) Execute() (*SyncConnection, error) {
	// 发送同步命令
	if err := c.sender("sync:"); err != nil {
		return nil, fmt.Errorf("发送同步命令失败: %v", err)
	}

	// 读取响应头
	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 创建新的同步连接
		return NewSyncConnection(c.sender, c.reader), nil

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return nil, fmt.Errorf("同步命令失败: %s", errMsg)

	default:
		return nil, fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// SyncConnection 处理同步连接
type SyncConnection struct {
	sender func(string) error
	reader func(int) (string, error)
}

// NewSyncConnection 创建新的同步连接
func NewSyncConnection(sender func(string) error, reader func(int) (string, error)) *SyncConnection {
	return &SyncConnection{
		sender: sender,
		reader: reader,
	}
}

// Push 推送文件到设备
func (c *SyncConnection) Push(reader io.Reader, path string, mode int, mtime int64) error {
	// 发送SEND命令
	cmd := fmt.Sprintf("SEND,%s,%d", path, mode)
	if err := c.sender(cmd); err != nil {
		return fmt.Errorf("发送SEND命令失败: %v", err)
	}

	// 传输文件数据
	buffer := make([]byte, 64*1024) // 64KB buffer
	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("读取文件数据失败: %v", err)
		}

		if n > 0 {
			// 发送数据块
			if err := c.sender(string(buffer[:n])); err != nil {
				return fmt.Errorf("发送文件数据失败: %v", err)
			}
		}

		if err == io.EOF {
			break
		}
	}

	// 发送DONE命令和修改时间
	doneCmd := fmt.Sprintf("DONE,%d", mtime)
	if err := c.sender(doneCmd); err != nil {
		return fmt.Errorf("发送DONE命令失败: %v", err)
	}

	// 检查响应
	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取最终响应失败: %v", err)
	}

	if reply != OKAY {
		return fmt.Errorf("unexpected final response: %s", reply)
	}

	return nil
}

// Pull 从设备拉取文件
func (c *SyncConnection) Pull(path string) (io.Reader, error) {
	// 发送RECV命令
	cmd := fmt.Sprintf("RECV,%s", path)
	if err := c.sender(cmd); err != nil {
		return nil, fmt.Errorf("发送RECV命令失败: %v", err)
	}

	// 创建并返回文件读取器
	return &syncReader{
		connection: c,
		buffer:     make([]byte, 64*1024), // 64KB buffer
	}, nil
}

// syncReader 实现io.Reader接口用于读取同步数据
type syncReader struct {
	connection *SyncConnection
	buffer     []byte
	offset     int
	length     int
}

// Read 实现io.Reader接口
func (r *syncReader) Read(p []byte) (n int, err error) {
	if r.offset >= r.length {
		// 需要读取新的数据块
		data, err := r.connection.reader(len(r.buffer))
		if err != nil {
			return 0, fmt.Errorf("读取同步数据失败: %v", err)
		}

		if len(data) == 0 {
			return 0, io.EOF
		}

		r.offset = 0
		r.length = len(data)
		copy(r.buffer, []byte(data))
	}

	// 复制数据到目标缓冲区
	n = copy(p, r.buffer[r.offset:r.length])
	r.offset += n
	return n, nil
}

// Close 关闭同步连接
func (c *SyncConnection) Close() error {
	// 发送QUIT命令
	if err := c.sender("QUIT"); err != nil {
		return fmt.Errorf("发送QUIT命令失败: %v", err)
	}
	return nil
}
