package sync

import (
	"io"
)

// PushTransfer 实现文件推送传输
type PushTransfer struct {
	stats struct {
		BytesTransferred int64
	}
	reader   io.Reader
	writer   io.Writer
	handlers map[string][]func(interface{})
}

// NewPushTransfer 创建新的推送传输实例
func NewPushTransfer() *PushTransfer {
	return &PushTransfer{
		handlers: make(map[string][]func(interface{})),
	}
}

// Write 实现io.Writer接口
func (t *PushTransfer) Write(p []byte) (n int, err error) {
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
func (t *PushTransfer) Read(p []byte) (n int, err error) {
	if t.reader == nil {
		return 0, io.EOF
	}
	return t.reader.Read(p)
}

// Cancel 取消传输
func (t *PushTransfer) Cancel() {
	t.emit("cancel", nil)
}

// On 注册事件处理器
func (t *PushTransfer) On(event string, handler func(interface{})) {
	if t.handlers[event] == nil {
		t.handlers[event] = make([]func(interface{}), 0)
	}
	t.handlers[event] = append(t.handlers[event], handler)
}

// emit 触发事件
func (t *PushTransfer) emit(event string, data interface{}) {
	if handlers, ok := t.handlers[event]; ok {
		for _, handler := range handlers {
			handler(data)
		}
	}
}

// SetReader 设置读取器
func (t *PushTransfer) SetReader(reader io.Reader) {
	t.reader = reader
}

// SetWriter 设置写入器
func (t *PushTransfer) SetWriter(writer io.Writer) {
	t.writer = writer
}

// Stats 获取传输统计信息
func (t *PushTransfer) Stats() interface{} {
	return t.stats
}

// BytesTransferred 获取已传输字节数
func (t *PushTransfer) BytesTransferred() int64 {
	return t.stats.BytesTransferred
}
