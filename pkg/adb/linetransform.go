package adb

import (
	"bytes"
	"io"
)

// LineTransform 实现行转换
type LineTransform struct {
	savedR          []byte
	autoDetect      bool
	transformNeeded bool
	skipBytes       int
}

// NewLineTransform 创建新的行转换器
func NewLineTransform(options map[string]interface{}) *LineTransform {
	lt := &LineTransform{
		transformNeeded: true,
	}

	// 检查是否启用自动检测
	if autoDetect, ok := options["autoDetect"].(bool); ok {
		lt.autoDetect = autoDetect
	}

	return lt
}

// Transform 转换数据
func (lt *LineTransform) Transform(chunk []byte) ([]byte, error) {
	if len(chunk) == 0 {
		return chunk, nil
	}

	// 如果启用了自动检测，检查第一个字节
	if lt.autoDetect {
		if chunk[0] == 0x0a {
			lt.transformNeeded = false
			lt.skipBytes = 1
		} else {
			lt.skipBytes = 2
		}
		lt.autoDetect = false
	}

	// 跳过指定的字节数
	if lt.skipBytes > 0 {
		skip := min(len(chunk), lt.skipBytes)
		chunk = chunk[skip:]
		lt.skipBytes -= skip
	}

	// 如果chunk为空，直接返回
	if len(chunk) == 0 {
		return chunk, nil
	}

	// 如果不需要转换，直接返回
	if !lt.transformNeeded {
		return chunk, nil
	}

	// 开始转换
	var result bytes.Buffer
	lo := 0
	hi := 0

	// 处理上一次保存的\r
	if lt.savedR != nil {
		if chunk[0] != 0x0a {
			result.Write(lt.savedR)
		}
		lt.savedR = nil
	}

	// 处理当前数据
	last := len(chunk) - 1
	for hi <= last {
		if chunk[hi] == 0x0d {
			if hi == last {
				// 保存最后的\r
				lt.savedR = chunk[last:]
				break
			} else if chunk[hi+1] == 0x0a {
				// 找到\r\n，只保留\n
				result.Write(chunk[lo:hi])
				lo = hi + 1
			}
		}
		hi++
	}

	// 写入剩余数据
	if hi > lo {
		result.Write(chunk[lo:hi])
	}

	return result.Bytes(), nil
}

// Flush 刷新缓存的数据
func (lt *LineTransform) Flush() ([]byte, error) {
	if lt.savedR != nil {
		result := lt.savedR
		lt.savedR = nil
		return result, nil
	}
	return nil, nil
}

// TransformReader 包装Reader以实现行转换
type TransformReader struct {
	reader    io.Reader
	transform *LineTransform
	buffer    []byte
}

// NewTransformReader 创建新的转换Reader
func NewTransformReader(reader io.Reader, options map[string]interface{}) *TransformReader {
	return &TransformReader{
		reader:    reader,
		transform: NewLineTransform(options),
		buffer:    make([]byte, 4096),
	}
}

// Read 实现io.Reader接口
func (tr *TransformReader) Read(p []byte) (n int, err error) {
	// 读取原始数据
	n, err = tr.reader.Read(tr.buffer)
	if err != nil && err != io.EOF {
		return 0, err
	}

	// 转换数据
	transformed, err := tr.transform.Transform(tr.buffer[:n])
	if err != nil {
		return 0, err
	}

	// 复制转换后的数据到输出buffer
	copy(p, transformed)

	// 如果是EOF，还需要刷新可能存在的缓存数据
	if err == io.EOF {
		flushed, ferr := tr.transform.Flush()
		if ferr != nil {
			return len(transformed), ferr
		}
		if len(flushed) > 0 {
			copy(p[len(transformed):], flushed)
			return len(transformed) + len(flushed), io.EOF
		}
	}

	return len(transformed), err
}

// TransformWriter 包装Writer以实现行转换
type TransformWriter struct {
	writer    io.Writer
	transform *LineTransform
}

// NewTransformWriter 创建新的转换Writer
func NewTransformWriter(writer io.Writer, options map[string]interface{}) *TransformWriter {
	return &TransformWriter{
		writer:    writer,
		transform: NewLineTransform(options),
	}
}

// Write 实现io.Writer接口
func (tw *TransformWriter) Write(p []byte) (n int, err error) {
	// 转换数据
	transformed, err := tw.transform.Transform(p)
	if err != nil {
		return 0, err
	}

	// 写入转换后的数据
	return tw.writer.Write(transformed)
}

// Close 关闭Writer并刷新缓存
func (tw *TransformWriter) Close() error {
	// 刷新可能存在的缓存数据
	flushed, err := tw.transform.Flush()
	if err != nil {
		return err
	}

	if len(flushed) > 0 {
		_, err = tw.writer.Write(flushed)
		if err != nil {
			return err
		}
	}

	// 如果底层Writer实现了Closer接口，调用其Close方法
	if closer, ok := tw.writer.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
