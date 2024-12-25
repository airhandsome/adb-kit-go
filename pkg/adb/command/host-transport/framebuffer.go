package hosttransport

import (
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
)

// FrameBufferMeta 存储帧缓冲区元数据
type FrameBufferMeta struct {
	Version     uint32
	Bpp         uint32
	Size        uint32
	Width       uint32
	Height      uint32
	RedOffset   uint32
	RedLength   uint32
	BlueOffset  uint32
	BlueLength  uint32
	GreenOffset uint32
	GreenLength uint32
	AlphaOffset uint32
	AlphaLength uint32
	Format      string
}
type BaseCommand struct {
	sender func(string) error
	reader func(int) (string, error)
}

// FrameBufferCommand 实现帧缓冲区命令
type FrameBufferCommand struct {
	BaseCommand
}

func NewFrameBufferCommand(sender func(string) error, reader func(int) (string, error)) *FrameBufferCommand {
	return &FrameBufferCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

func (c *FrameBufferCommand) Execute(format string) (io.Reader, *FrameBufferMeta, error) {
	if err := c.sender("framebuffer:"); err != nil {
		return nil, nil, fmt.Errorf("发送帧缓冲区命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return nil, nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 读取头部信息
		headerBytes, err := c.reader(52)
		if err != nil {
			return nil, nil, fmt.Errorf("读取头部信息失败: %v", err)
		}

		meta, err := c.parseHeader([]byte(headerBytes))
		if err != nil {
			return nil, nil, fmt.Errorf("解析头部信息失败: %v", err)
		}

		// 根据格式选择输出
		if format == "raw" {
			return c.getRawStream(), meta, nil
		}

		return c.convertStream(meta, format)

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return nil, nil, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return nil, nil, fmt.Errorf(errMsg)

	default:
		return nil, nil, fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

func (c *FrameBufferCommand) parseHeader(header []byte) (*FrameBufferMeta, error) {
	if len(header) < 52 {
		return nil, fmt.Errorf("header too short: %d bytes", len(header))
	}

	meta := &FrameBufferMeta{}

	meta.Version = binary.LittleEndian.Uint32(header[0:4])
	if meta.Version == 16 {
		return nil, fmt.Errorf("old-style raw images are not supported")
	}

	meta.Bpp = binary.LittleEndian.Uint32(header[4:8])
	meta.Size = binary.LittleEndian.Uint32(header[8:12])
	meta.Width = binary.LittleEndian.Uint32(header[12:16])
	meta.Height = binary.LittleEndian.Uint32(header[16:20])
	meta.RedOffset = binary.LittleEndian.Uint32(header[20:24])
	meta.RedLength = binary.LittleEndian.Uint32(header[24:28])
	meta.BlueOffset = binary.LittleEndian.Uint32(header[28:32])
	meta.BlueLength = binary.LittleEndian.Uint32(header[32:36])
	meta.GreenOffset = binary.LittleEndian.Uint32(header[36:40])
	meta.GreenLength = binary.LittleEndian.Uint32(header[40:44])
	meta.AlphaOffset = binary.LittleEndian.Uint32(header[44:48])
	meta.AlphaLength = binary.LittleEndian.Uint32(header[48:52])

	// 设置格式
	if meta.BlueOffset == 0 {
		meta.Format = "bgr"
	} else {
		meta.Format = "rgb"
	}
	if meta.Bpp == 32 || meta.AlphaLength > 0 {
		meta.Format += "a"
	}

	return meta, nil
}

// 辅助方法

type customReader struct {
	readerFunc func(int) (string, error)
	buffer     []byte
	offset     int
}

func (cr *customReader) Read(p []byte) (n int, err error) {
	if cr.offset >= len(cr.buffer) {
		// 需要更多数据
		data, err := cr.readerFunc(len(p))
		if err != nil {
			return 0, err
		}
		cr.buffer = []byte(data)
		cr.offset = 0
	}

	if len(cr.buffer) == 0 {
		return 0, io.EOF
	}

	n = copy(p, cr.buffer[cr.offset:])
	cr.offset += n
	return n, nil
}

func (c *FrameBufferCommand) getRawStream() io.Reader {
	return &customReader{
		readerFunc: c.reader,
	}
}

func (c *FrameBufferCommand) convertStream(meta *FrameBufferMeta, format string) (io.Reader, *FrameBufferMeta, error) {
	// 使用 GraphicsMagick 转换图像格式
	args := []string{
		"convert",
		"-size",
		fmt.Sprintf("%dx%d", meta.Width, meta.Height),
		fmt.Sprintf("%s:-", meta.Format),
		fmt.Sprintf("%s:-", format),
	}

	cmd := exec.Command("gm", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("创建输入管道失败: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("创建输出管道失败: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("启动转换进程失败: %v", err)
	}

	// 启动goroutine来复制数据
	go func() {
		io.Copy(stdin, c.getRawStream())
		stdin.Close()
	}()

	return stdout, meta, nil
}
