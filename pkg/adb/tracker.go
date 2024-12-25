package adb

import (
	"sync"
)

// Tracker 设备跟踪器
type Tracker struct {
	command    *Command
	deviceList []*Device
	deviceMap  map[string]*Device
	listeners  map[string][]func(interface{})
	mu         sync.RWMutex
}

// ChangeSet 设备变更集
type ChangeSet struct {
	Removed []Device
	Changed []Device
	Added   []Device
}

// NewTracker 创建新的设备跟踪器
func NewTracker(command *Command) *Tracker {
	t := &Tracker{
		command:   command,
		deviceMap: make(map[string]*Device),
		listeners: make(map[string][]func(interface{})),
	}

	// 启动读取循环
	go t.read()

	return t
}

// Track 开始跟踪设备变化
func (t *Tracker) Track() error {
	return t.command.Execute("track-devices")
}

// On 注册事件监听器
func (t *Tracker) On(event string, handler func(interface{})) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.listeners[event] = append(t.listeners[event], handler)
}

// End 结束跟踪
func (t *Tracker) End() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 清理资源
	t.deviceList = nil
	t.deviceMap = make(map[string]*Device)

	// 发送结束事件
	t.emit("end", nil)

	return nil
}

// GetDevices 获取当前设备列表
func (t *Tracker) GetDevices() []*Device {
	t.mu.RLock()
	defer t.mu.RUnlock()

	devices := make([]*Device, len(t.deviceList))
	copy(devices, t.deviceList)
	return devices
}

// read 持续读取设备列表
func (t *Tracker) read() {
	for {
		devices, err := t.command.GetDevices()
		if err != nil {
			t.emit("error", err)
			continue
		}

		t.update(devices)
	}
}

// update 更新设备列表
func (t *Tracker) update(newList []*Device) {
	t.mu.Lock()
	defer t.mu.Unlock()

	changes := ChangeSet{}
	newMap := make(map[string]*Device)

	// 检查新增和变更的设备
	for _, device := range newList {
		newMap[device.ID] = device

		if oldDevice, exists := t.deviceMap[device.ID]; exists {
			// 检查设备状态是否变更
			if oldDevice.State != device.State {
				changes.Changed = append(changes.Changed, *device)
				t.emit("change", device)
			}
		} else {
			// 新增设备
			changes.Added = append(changes.Added, *device)
			t.emit("add", device)
		}
	}

	// 检查移除的设备
	for _, device := range t.deviceList {
		if _, exists := newMap[device.ID]; !exists {
			changes.Removed = append(changes.Removed, *device)
			t.emit("remove", device)
		}
	}

	// 更新设备列表和映射
	t.deviceList = newList
	t.deviceMap = newMap

	// 发送变更集事件
	if len(changes.Added) > 0 || len(changes.Changed) > 0 || len(changes.Removed) > 0 {
		t.emit("changeSet", changes)
	}
}

// emit 发送事件
func (t *Tracker) emit(event string, data interface{}) {
	t.mu.RLock()
	handlers := make([]func(interface{}), len(t.listeners[event]))
	copy(handlers, t.listeners[event])
	t.mu.RUnlock()

	for _, handler := range handlers {
		go handler(data)
	}
}

// Device 设备信息
type Device struct {
	ID    string
	State string
	Props map[string]string
}

// NewDevice 创建新的设备对象
func NewDevice(id string, state string) *Device {
	return &Device{
		ID:    id,
		State: state,
		Props: make(map[string]string),
	}
}

// SetProperty 设置设备属性
func (d *Device) SetProperty(key, value string) {
	d.Props[key] = value
}

// GetProperty 获取设备属性
func (d *Device) GetProperty(key string) string {
	return d.Props[key]
}

// IsOnline 检查设备是否在线
func (d *Device) IsOnline() bool {
	return d.State == "device"
}

// String 设备字符串表示
func (d *Device) String() string {
	return d.ID + " " + d.State
}
