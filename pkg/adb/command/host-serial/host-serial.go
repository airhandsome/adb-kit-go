package hostserial

import (
	"fmt"
	"strings"
)

const (
	OKAY = "OKAY"
	FAIL = "FAIL"
)

// BaseCommand 提供基础功能
type BaseCommand struct {
	sender func(string) error
	reader func(int) (string, error)
}

// GetDevicePathCommand 实现获取设备路径命令
type GetDevicePathCommand struct {
	BaseCommand
}

// ForwardCommand 实现端口转发命令
type ForwardCommand struct {
	BaseCommand
}

// GetSerialNoCommand 实现获取序列号命令
type GetSerialNoCommand struct {
	BaseCommand
}

// ListForwardsCommand 实现列出转发配置命令
type ListForwardsCommand struct {
	BaseCommand
}
type WaitForDeviceCommand struct {
	BaseCommand
}

// Forward 表示一个端口转发配置
type Forward struct {
	Serial string
	Local  string
	Remote string
}

// 工厂函数
func NewGetDevicePathCommand(sender func(string) error, reader func(int) (string, error)) *GetDevicePathCommand {
	return &GetDevicePathCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

func NewForwardCommand(sender func(string) error, reader func(int) (string, error)) *ForwardCommand {
	return &ForwardCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// 工厂函数
func NewGetSerialNoCommand(sender func(string) error, reader func(int) (string, error)) *GetSerialNoCommand {
	return &GetSerialNoCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

func NewListForwardsCommand(sender func(string) error, reader func(int) (string, error)) *ListForwardsCommand {
	return &ListForwardsCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}
func NewWaitForDeviceCommand(sender func(string) error, reader func(int) (string, error)) *WaitForDeviceCommand {
	return &WaitForDeviceCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行获取设备路径命令
func (c *GetDevicePathCommand) Execute(serial string) (string, error) {
	cmd := fmt.Sprintf("host-serial:%s:get-devpath", serial)
	if err := c.sender(cmd); err != nil {
		return "", fmt.Errorf("发送获取设备路径命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		value, err := c.reader(0)
		if err != nil {
			return "", fmt.Errorf("读取设备路径失败: %v", err)
		}
		return value, nil
	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return "", fmt.Errorf("读取错误信息失败: %v", err)
		}
		return "", fmt.Errorf(errMsg)
	default:
		return "", fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// Execute 执行端口转发命令
func (c *ForwardCommand) Execute(serial, local, remote string) (bool, error) {
	cmd := fmt.Sprintf("host-serial:%s:forward:%s;%s", serial, local, remote)
	if err := c.sender(cmd); err != nil {
		return false, fmt.Errorf("发送端口转发命令失败: %v", err)
	}

	// 第一次读取响应
	reply, err := c.reader(4)
	if err != nil {
		return false, fmt.Errorf("读取第一次响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 第二次读取响应
		reply, err = c.reader(4)
		if err != nil {
			return false, fmt.Errorf("读取第二次响应失败: %v", err)
		}

		switch reply {
		case OKAY:
			return true, nil
		case FAIL:
			errMsg, err := c.reader(0)
			if err != nil {
				return false, fmt.Errorf("读取错误信息失败: %v", err)
			}
			return false, fmt.Errorf(errMsg)
		default:
			return false, fmt.Errorf("unexpected second response: %s, expected OKAY or FAIL", reply)
		}
	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return false, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return false, fmt.Errorf(errMsg)
	default:
		return false, fmt.Errorf("unexpected first response: %s, expected OKAY or FAIL", reply)
	}
}

func (c *GetSerialNoCommand) Execute(serial string) (string, error) {
	cmd := fmt.Sprintf("host-serial:%s:get-serialno", serial)
	if err := c.sender(cmd); err != nil {
		return "", fmt.Errorf("发送获取序列号命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		value, err := c.reader(0)
		if err != nil {
			return "", fmt.Errorf("读取序列号失败: %v", err)
		}
		return value, nil
	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return "", fmt.Errorf("读取错误信息失败: %v", err)
		}
		return "", fmt.Errorf(errMsg)
	default:
		return "", fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// Execute 执行列出转发配置命令
func (c *ListForwardsCommand) Execute(serial string) ([]Forward, error) {
	cmd := fmt.Sprintf("host-serial:%s:list-forward", serial)
	if err := c.sender(cmd); err != nil {
		return nil, fmt.Errorf("发送列出转发配置命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		value, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取转发配置失败: %v", err)
		}
		return c.parseForwards(value)
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

// parseForwards 解析转发配置列表
func (c *ListForwardsCommand) parseForwards(value string) ([]Forward, error) {
	forwards := make([]Forward, 0)
	lines := strings.Split(strings.TrimSpace(value), "\n")

	for _, line := range lines {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid forward format: %s", line)
		}

		forwards = append(forwards, Forward{
			Serial: parts[0],
			Local:  parts[1],
			Remote: parts[2],
		})
	}

	return forwards, nil
}

// Execute 执行等待设备命令
func (c *WaitForDeviceCommand) Execute(serial string) (string, error) {
	cmd := fmt.Sprintf("host-serial:%s:wait-for-any", serial)
	if err := c.sender(cmd); err != nil {
		return "", fmt.Errorf("发送等待设备命令失败: %v", err)
	}

	// 第一次读取响应
	reply, err := c.reader(4)
	if err != nil {
		return "", fmt.Errorf("读取第一次响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 第二次读取响应
		reply, err = c.reader(4)
		if err != nil {
			return "", fmt.Errorf("读取第二次响应失败: %v", err)
		}

		switch reply {
		case OKAY:
			return serial, nil
		case FAIL:
			errMsg, err := c.reader(0)
			if err != nil {
				return "", fmt.Errorf("读取错误信息失败: %v", err)
			}
			return "", fmt.Errorf(errMsg)
		default:
			return "", fmt.Errorf("unexpected second response: %s, expected OKAY or FAIL", reply)
		}
	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return "", fmt.Errorf("读取错误信息失败: %v", err)
		}
		return "", fmt.Errorf(errMsg)
	default:
		return "", fmt.Errorf("unexpected first response: %s, expected OKAY or FAIL", reply)
	}
}
