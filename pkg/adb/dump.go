package adb

import (
	"io"
	"os"
	"sync"
)

var (
	dumpEnabled bool
	dumpFile    *os.File
	dumpMutex   sync.Mutex
	dumpWriter  io.Writer
)

func init() {
	// 检查环境变量是否启用dump
	dumpEnabled = os.Getenv("ADBKIT_DUMP") != ""
	if dumpEnabled {
		var err error
		dumpFile, err = os.OpenFile("adbkit.dump", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			// 如果无法创建dump文件，则禁用dump
			dumpEnabled = false
		}
	}
}

// Dump 记录数据
func Dump(data []byte) {
	if dumpEnabled && dumpFile != nil {
		dumpMutex.Lock()
		defer dumpMutex.Unlock()
		if dumpWriter != nil {
			dumpWriter.Write(data)
		} else if dumpWriter == nil {
			dumpFile.Write(data)
		}
	}
}

// DumpReader 创建可以dump的Reader
type DumpReader struct {
	reader io.Reader
}

// NewDumpReader 创建新的DumpReader
func NewDumpReader(reader io.Reader) *DumpReader {
	return &DumpReader{reader: reader}
}

// Read 实现io.Reader接口
func (d *DumpReader) Read(p []byte) (n int, err error) {
	n, err = d.reader.Read(p)
	if n > 0 {
		Dump(p[:n])
	}
	return
}

// DumpWriter 创建可以dump的Writer
type DumpWriter struct {
	writer io.Writer
}

// NewDumpWriter 创建新的DumpWriter
func NewDumpWriter(writer io.Writer) *DumpWriter {
	return &DumpWriter{writer: writer}
}

// Write 实现io.Writer接口
func (d *DumpWriter) Write(p []byte) (n int, err error) {
	Dump(p)
	return d.writer.Write(p)
}

// DumpCloser 关闭dump文件
func DumpCloser() error {
	if dumpEnabled {
		dumpMutex.Lock()
		defer dumpMutex.Unlock()
		if dumpFile != nil {
			err := dumpFile.Close()
			dumpFile = nil
			return err
		}
		dumpWriter = nil
		dumpEnabled = false
	}
	return nil
}

// IsDumpEnabled 检查dump是否启用
func IsDumpEnabled() bool {
	return dumpEnabled
}

// SetDumpFile 设置dump文件
func SetDumpFile(file string) error {
	dumpMutex.Lock()
	defer dumpMutex.Unlock()

	// 关闭现有的dump文件
	if dumpFile != nil {
		dumpFile.Close()
	}

	var err error
	dumpFile, err = os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		dumpEnabled = false
		return err
	}
	dumpWriter = nil
	dumpEnabled = true
	return nil
}

// DumpToWriter 将dump输出重定向到指定的Writer
func DumpToWriter(writer io.Writer) {
	dumpMutex.Lock()
	defer dumpMutex.Unlock()

	if dumpFile != nil {
		dumpFile.Close()
		dumpFile = nil
	}

	if writer != nil {
		dumpWriter = writer
		dumpEnabled = true
	} else {
		dumpWriter = nil
		dumpEnabled = false
	}
}

// writerFile 包装io.Writer以实现类文件接口
type writerFile struct {
	writer io.Writer
}

func (w *writerFile) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}

func (w *writerFile) Close() error {
	if closer, ok := w.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// DumpBuffer 用于缓存dump数据的buffer
type DumpBuffer struct {
	data []byte
}

// NewDumpBuffer 创建新的DumpBuffer
func NewDumpBuffer() *DumpBuffer {
	return &DumpBuffer{
		data: make([]byte, 0),
	}
}

// Write 写入数据到buffer
func (b *DumpBuffer) Write(p []byte) (n int, err error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

// Bytes 获取buffer中的数据
func (b *DumpBuffer) Bytes() []byte {
	return b.data
}

// String 获取buffer中的数据as字符串
func (b *DumpBuffer) String() string {
	return string(b.data)
}

// Reset 重置buffer
func (b *DumpBuffer) Reset() {
	b.data = b.data[:0]
}
