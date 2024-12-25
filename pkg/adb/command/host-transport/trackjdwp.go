package hosttransport

import (
	"fmt"
	"strings"
)

// TrackJdwpCommand 实现JDWP跟踪命令
type TrackJdwpCommand struct {
	BaseCommand
}

// NewTrackJdwpCommand 创建新的JDWP跟踪命令实例
func NewTrackJdwpCommand(sender func(string) error, reader func(int) (string, error)) *TrackJdwpCommand {
	return &TrackJdwpCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行JDWP跟踪命令
func (c *TrackJdwpCommand) Execute() (*JdwpTracker, error) {
	if err := c.sender("track-jdwp"); err != nil {
		return nil, fmt.Errorf("发送JDWP跟踪命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		return NewJdwpTracker(c.sender, c.reader), nil
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

// JdwpTracker 处理JDWP跟踪
type JdwpTracker struct {
	sender   func(string) error
	reader   func(int) (string, error)
	pids     []string
	pidMap   map[string]bool
	handlers map[string][]func(string)
}

// NewJdwpTracker 创建新的JDWP跟踪器
func NewJdwpTracker(sender func(string) error, reader func(int) (string, error)) *JdwpTracker {
	t := &JdwpTracker{
		sender:   sender,
		reader:   reader,
		pids:     make([]string, 0),
		pidMap:   make(map[string]bool),
		handlers: make(map[string][]func(string)),
	}
	go t.readLoop()
	return t
}

// readLoop 持续读取JDWP数据
func (t *JdwpTracker) readLoop() {
	for {
		data, err := t.reader(0)
		if err != nil {
			t.emit("error", err.Error())
			t.emit("end", "")
			return
		}

		newPids := strings.Split(strings.TrimSpace(data), "\n")
		t.update(newPids)
	}
}

// update 更新PID列表
func (t *JdwpTracker) update(newPids []string) {
	changes := struct {
		Added   []string
		Removed []string
	}{
		Added:   make([]string, 0),
		Removed: make([]string, 0),
	}

	newPidMap := make(map[string]bool)
	for _, pid := range newPids {
		newPidMap[pid] = true
		if !t.pidMap[pid] {
			changes.Added = append(changes.Added, pid)
			t.emit("add", pid)
		}
	}

	for _, pid := range t.pids {
		if !newPidMap[pid] {
			changes.Removed = append(changes.Removed, pid)
			t.emit("remove", pid)
		}
	}

	t.pids = newPids
	t.pidMap = newPidMap
	t.emit("changeSet", fmt.Sprintf("%v", changes))
}

// On 注册事件处理器
func (t *JdwpTracker) On(event string, handler func(string)) {
	if t.handlers[event] == nil {
		t.handlers[event] = make([]func(string), 0)
	}
	t.handlers[event] = append(t.handlers[event], handler)
}

// emit 触发事件
func (t *JdwpTracker) emit(event string, data string) {
	if handlers, ok := t.handlers[event]; ok {
		for _, handler := range handlers {
			handler(data)
		}
	}
}

// End 结束跟踪
func (t *JdwpTracker) End() {
	t.emit("end", "")
}
