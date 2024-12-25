package sync

import (
	"io"
)

// PullTransfer 实现文件拉取传输
type PullTransfer struct {
	stats struct {
		BytesTransferred int64
	}
	reader   io.Reader
	writer   io.Writer
	handlers map[string][]func(interface{})
}

// NewPullTransfer 创建新的拉取传输实例
func NewPullTransfer() *PullTransfer {
	return &PullTransfer{
		handlers: make(map[string][]func(interface{})),
	}
}

// Write 实现io.Writer接口
func (t *PullTransfer) Write(p []byte) (n int, err error) {
	// 更新传输字节数
	t.stats.BytesTransferred += int64(len(p))

	// 触发进度事件
	t.emit("progress", t.stats)

	// 如果设置了writer，写入数据
	if t.writer != nil {
		return t.writer.Write(p)
	}
	return len(p), nil
}

// Read 实现io.Reader接口
func (t *PullTransfer) Read(p []byte) (n int, err error) {
	if t.reader == nil {
		return 0, io.EOF
	}
	return t.reader.Read(p)
}

// Cancel 取消传输
func (t *PullTransfer) Cancel() {
	t.emit("cancel", nil)
}

// On 注册事件处理器
func (t *PullTransfer) On(event string, handler func(interface{})) {
	if t.handlers[event] == nil {
		t.handlers[event] = make([]func(interface{}), 0)
	}
	t.handlers[event] = append(t.handlers[event], handler)
}

// emit 触发事件
func (t *PullTransfer) emit(event string, data interface{}) {
	if handlers, ok := t.handlers[event]; ok {
		for _, handler := range handlers {
			handler(data)
		}
	}
}

// SetReader 设置读取器
func (t *PullTransfer) SetReader(reader io.Reader) {
	t.reader = reader
}

// SetWriter 设置写入器
func (t *PullTransfer) SetWriter(writer io.Writer) {
	t.writer = writer
}

// Stats 获取传输统计信息
func (t *PullTransfer) Stats() interface{} {
	return t.stats
}

// BytesTransferred 获取已传输字节数
func (t *PullTransfer) BytesTransferred() int64 {
	return t.stats.BytesTransferred
}
