package framebuffer

import (
	"fmt"
	"io"
)

type Meta struct {
	Bpp         int
	RedOffset   int
	GreenOffset int
	BlueOffset  int
	AlphaOffset int
}

type RgbTransform struct {
	meta       Meta
	buffer     []byte
	rPos       int
	gPos       int
	bPos       int
	aPos       int
	pixelBytes int
}

func NewRgbTransform(meta Meta) (*RgbTransform, error) {
	if meta.Bpp != 24 && meta.Bpp != 32 {
		return nil, fmt.Errorf("只支持24位和32位每像素的原始图像（每种颜色8位）")
	}

	return &RgbTransform{
		meta:       meta,
		buffer:     make([]byte, 0),
		rPos:       meta.RedOffset / 8,
		gPos:       meta.GreenOffset / 8,
		bPos:       meta.BlueOffset / 8,
		aPos:       meta.AlphaOffset / 8,
		pixelBytes: meta.Bpp / 8,
	}, nil
}

func (t *RgbTransform) Transform(input []byte) ([]byte, error) {
	// 合并现有buffer和新输入
	t.buffer = append(t.buffer, input...)

	sourceCursor := 0
	targetCursor := 0

	// 计算目标buffer大小
	targetSize := len(t.buffer) / t.pixelBytes * 3
	target := make([]byte, targetSize)

	// 处理每个像素
	for len(t.buffer)-sourceCursor >= t.pixelBytes {
		r := t.buffer[sourceCursor+t.rPos]
		g := t.buffer[sourceCursor+t.gPos]
		b := t.buffer[sourceCursor+t.bPos]

		target[targetCursor+0] = r
		target[targetCursor+1] = g
		target[targetCursor+2] = b

		sourceCursor += t.pixelBytes
		targetCursor += 3
	}

	// 保存未处理的数据
	t.buffer = t.buffer[sourceCursor:]

	return target[:targetCursor], nil
}

// 实现io.Writer接口的写入方法
func (t *RgbTransform) Write(p []byte) (n int, err error) {
	output, err := t.Transform(p)
	if err != nil {
		return 0, err
	}
	// 这里需要实现将转换后的数据写入到某个目标。
	// 具体实现取决于你的需求
	fmt.Println(output)
	return len(p), nil
}

// 如果需要实现完整的流处理，可以添加Read方法
func (t *RgbTransform) Read(p []byte) (n int, err error) {
	// 实现读取逻辑
	return 0, io.EOF
}
