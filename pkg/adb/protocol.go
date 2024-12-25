package adb

import (
	"bytes"
	"fmt"
	"strconv"
)

// Protocol ADB协议常量和工具
type Protocol struct{}

// 协议常量
const (
	OKAY = "OKAY"
	FAIL = "FAIL"
	STAT = "STAT"
	LIST = "LIST"
	DENT = "DENT"
	RECV = "RECV"
	DATA = "DATA"
	DONE = "DONE"
	SEND = "SEND"
	QUIT = "QUIT"
)

// DecodeLength 解码长度值（从16进制字符串）
func (p *Protocol) DecodeLength(length string) (int, error) {
	val, err := strconv.ParseInt(length, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to decode length: %v", err)
	}
	return int(val), nil
}

// EncodeLength 编码长度值（到16进制字符串）
func (p *Protocol) EncodeLength(length int) string {
	return fmt.Sprintf("%04X", length)
}

// EncodeData 编码数据（添加长度前缀）
func (p *Protocol) EncodeData(data []byte) []byte {
	if data == nil {
		data = []byte{}
	}

	// 创建长度前缀
	lengthPrefix := []byte(p.EncodeLength(len(data)))

	// 合并长度前缀和数据
	return append(lengthPrefix, data...)
}

// DecodeData 解码数据（解析长度前缀）
func (p *Protocol) DecodeData(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("data too short for protocol decode")
	}

	// 解析长度
	length, err := p.DecodeLength(string(data[:4]))
	if err != nil {
		return nil, err
	}

	// 验证数据长度
	if len(data) < 4+length {
		return nil, fmt.Errorf("incomplete data: expected %d bytes, got %d", length, len(data)-4)
	}

	// 返回实际数据
	return data[4 : 4+length], nil
}

// EncodeMessage 编码消息（包括命令和参数）
func (p *Protocol) EncodeMessage(cmd string, args ...string) []byte {
	var buffer bytes.Buffer

	// 写入命令
	buffer.WriteString(cmd)

	// 写入参数
	for _, arg := range args {
		buffer.WriteByte(':')
		buffer.WriteString(arg)
	}

	return p.EncodeData(buffer.Bytes())
}

// ValidateResponse 验证响应
func (p *Protocol) ValidateResponse(response []byte, expected string) error {
	if len(response) < 4 {
		return fmt.Errorf("response too short")
	}

	if string(response[:4]) != expected {
		return fmt.Errorf("unexpected response: got %s, want %s",
			string(response[:4]), expected)
	}

	return nil
}

// FormatSync 格式化同步命令
func (p *Protocol) FormatSync(cmd string, length int) []byte {
	// 同步命令固定4字节命令+4字节长度
	message := make([]byte, 8)
	copy(message[:4], cmd)
	copy(message[4:], []byte(fmt.Sprintf("%04x", length)))
	return message
}

// ParseSyncResponse 解析同步响应
func (p *Protocol) ParseSyncResponse(response []byte) (string, int, error) {
	if len(response) < 8 {
		return "", 0, fmt.Errorf("sync response too short")
	}

	cmd := string(response[:4])
	length, err := p.DecodeLength(string(response[4:8]))
	if err != nil {
		return "", 0, fmt.Errorf("invalid sync response length: %v", err)
	}

	return cmd, length, nil
}

// FormatSyncRequest 格式化同步请求
func (p *Protocol) FormatSyncRequest(cmd string, path string) []byte {
	// 计算总长度
	length := len(path)

	// 创建请求
	message := p.FormatSync(cmd, length)
	return append(message, []byte(path)...)
}

// EncodeString 编码字符串（用于传输）
func (p *Protocol) EncodeString(s string) []byte {
	return p.EncodeData([]byte(s))
}

// DecodeString 解码字符串
func (p *Protocol) DecodeString(data []byte) (string, error) {
	decoded, err := p.DecodeData(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// NewProtocol 创建新的协议实例
func NewProtocol() *Protocol {
	return &Protocol{}
}

// 使用示例：
func ExampleProtocol() {
	p := NewProtocol()

	// 编码数据
	encoded := p.EncodeData([]byte("hello"))
	fmt.Printf("Encoded: %x\n", encoded)

	// 解码数据
	decoded, _ := p.DecodeData(encoded)
	fmt.Printf("Decoded: %s\n", decoded)

	// 编码消息
	message := p.EncodeMessage("shell", "ls", "-l")
	fmt.Printf("Message: %s\n", message)

	// 格式化同步请求
	syncReq := p.FormatSyncRequest("SEND", "/sdcard/file.txt")
	fmt.Printf("Sync Request: %x\n", syncReq)
}
