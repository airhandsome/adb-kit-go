package host

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	OKAY = "OKAY"
	FAIL = "FAIL"
)

// Device 表示一个ADB设备
type Device struct {
	ID   string
	Type string
	Path string // 仅在 devices-l 命令中使用
}

// Command 接口定义了所有ADB主机命令的基本行为
type Command interface {
	Execute() (interface{}, error)
}

// BaseCommand 提供基础功能
type BaseCommand struct {
	sender func(string) error
	reader func(int) (string, error)
}

// DevicesCommand 实现基础的设备列表查询
type DevicesCommand struct {
	BaseCommand
}

// DevicesWithPathsCommand 实现带路径信息的设备列表查询
type DevicesWithPathsCommand struct {
	BaseCommand
}

// KillCommand 实现终止ADB服务器的命令
type KillCommand struct {
	BaseCommand
}
type VersionCommand struct {
	BaseCommand
}

type TransportCommand struct {
	BaseCommand
}

type TrackDevicesCommand struct {
	BaseCommand
	DevicesCommand
	onTrack func([]Device)
}

// Tracker 接口定义设备跟踪器的行为
type Tracker interface {
	Start() error
	Stop() error
}

// 实现 Tracker
type deviceTracker struct {
	cmd      *TrackDevicesCommand
	done     chan bool
	tracking bool
}

func NewConnectCommand(sender func(string) error, reader func(int) (string, error)) *BaseCommand {
	return &BaseCommand{
		sender: sender,
		reader: reader,
	}
}

func NewDevicesCommand(sender func(string) error, reader func(int) (string, error)) *DevicesCommand {
	return &DevicesCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

func NewDevicesWithPathsCommand(sender func(string) error, reader func(int) (string, error)) *DevicesWithPathsCommand {
	return &DevicesWithPathsCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

func NewKillCommand(sender func(string) error, reader func(int) (string, error)) *KillCommand {
	return &KillCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}
func NewVersionCommand(sender func(string) error, reader func(int) (string, error)) *VersionCommand {
	return &VersionCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

func NewTransportCommand(sender func(string) error, reader func(int) (string, error)) *TransportCommand {
	return &TransportCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

func NewTrackDevicesCommand(sender func(string) error, reader func(int) (string, error), onTrack func([]Device)) *TrackDevicesCommand {
	return &TrackDevicesCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
		onTrack: onTrack,
	}
}

// 连接成功的正则表达式
var reOK = regexp.MustCompile(`connected to|already connected`)

// Execute 执行连接命令
// 可能的返回值:
// - "unable to connect to 192.168.2.2:5555"
// - "connected to 192.168.2.2:5555"
// - "already connected to 192.168.2.2:5555"
func (c *BaseCommand) Execute(host string, port string) (string, error) {
	// 发送连接命令
	cmd := fmt.Sprintf("host:connect:%s:%s", host, port)
	if err := c.sender(cmd); err != nil {
		return "", fmt.Errorf("发送命令失败: %v", err)
	}

	// 读取4字节响应
	reply, err := c.reader(4)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取详细响应
		value, err := c.reader(0) // 0表示读取到结束
		if err != nil {
			return "", fmt.Errorf("读取详细信息失败: %v", err)
		}

		if reOK.MatchString(value) {
			return fmt.Sprintf("%s:%s", host, port), nil
		}
		return "", fmt.Errorf(value)

	case FAIL:
		// 读取错误信息
		errMsg, err := c.reader(0)
		if err != nil {
			return "", fmt.Errorf("读取错误信息失败: %v", err)
		}
		return "", fmt.Errorf(errMsg)

	default:
		return "", fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// Execute 执行设备列表查询命令
func (c *DevicesCommand) Execute() (interface{}, error) {
	if err := c.sender("host:devices"); err != nil {
		return nil, fmt.Errorf("发送命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		return c.readDevices()
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

// Execute 执行带路径信息的设备列表查询命令
func (c *DevicesWithPathsCommand) Execute() (interface{}, error) {
	if err := c.sender("host:devices-l"); err != nil {
		return nil, fmt.Errorf("发送命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		return c.readDevices()
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

// Execute 执行终止ADB服务器命令
func (c *KillCommand) Execute() (interface{}, error) {
	if err := c.sender("host:kill"); err != nil {
		return nil, fmt.Errorf("发送终止命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		return true, nil
	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return false, fmt.Errorf(errMsg)
	default:
		return false, fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// 辅助方法
func (c *DevicesCommand) readDevices() ([]Device, error) {
	value, err := c.reader(0)
	if err != nil {
		return nil, fmt.Errorf("读取设备列表失败: %v", err)
	}
	return c.parseDevices(value)
}

func (c *DevicesWithPathsCommand) readDevices() ([]Device, error) {
	value, err := c.reader(0)
	if err != nil {
		return nil, fmt.Errorf("读取设备列表失败: %v", err)
	}
	return c.parseDevices(value)
}

func (c *DevicesCommand) parseDevices(value string) ([]Device, error) {
	devices := make([]Device, 0)
	if value == "" {
		return devices, nil
	}

	lines := strings.Split(value, "\n")
	for _, line := range lines {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			return nil, fmt.Errorf("无效的设备信息格式: %s", line)
		}

		devices = append(devices, Device{
			ID:   parts[0],
			Type: parts[1],
		})
	}
	return devices, nil
}

func (c *DevicesWithPathsCommand) parseDevices(value string) ([]Device, error) {
	devices := make([]Device, 0)
	if value == "" {
		return devices, nil
	}

	lines := strings.Split(value, "\n")
	for _, line := range lines {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			return nil, fmt.Errorf("无效的设备信息格式: %s", line)
		}

		devices = append(devices, Device{
			ID:   parts[0],
			Type: parts[1],
			Path: parts[2],
		})
	}
	return devices, nil
}
func (c *VersionCommand) Execute() (interface{}, error) {
	if err := c.sender("host:version"); err != nil {
		return nil, fmt.Errorf("发送版本查询命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		value, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取版本值失败: %v", err)
		}
		return c.parseVersion(value)
	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return nil, fmt.Errorf(errMsg)
	default:
		// 某些版本的ADB直接返回版本号
		return c.parseVersion(reply)
	}
}

// Execute 执行传输命令
func (c *TransportCommand) Execute(serial string) (interface{}, error) {
	cmd := fmt.Sprintf("host:transport:%s", serial)
	if err := c.sender(cmd); err != nil {
		return nil, fmt.Errorf("发送传输命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		return true, nil
	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return false, fmt.Errorf(errMsg)
	default:
		return false, fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// Execute 执行设备跟踪命令
func (c *TrackDevicesCommand) Execute() (interface{}, error) {
	if err := c.sender("host:track-devices"); err != nil {
		return nil, fmt.Errorf("发送跟踪命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		tracker := &deviceTracker{
			cmd:  c,
			done: make(chan bool),
		}
		return tracker, nil
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

// 辅助方法
func (c *VersionCommand) parseVersion(version string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(version), 16, 64)
}

// Tracker 实现
func (t *deviceTracker) Start() error {
	if t.tracking {
		return fmt.Errorf("already tracking")
	}

	t.tracking = true
	go func() {
		for t.tracking {
			select {
			case <-t.done:
				return
			default:
				if devices, err := t.cmd.readDevices(); err == nil && t.cmd.onTrack != nil {
					t.cmd.onTrack(devices)
				}
			}
		}
	}()

	return nil
}

func (t *deviceTracker) Stop() error {
	if !t.tracking {
		return nil
	}
	t.tracking = false
	t.done <- true
	return nil
}
