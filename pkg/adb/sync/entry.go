package sync

import (
	"time"
)

// Entry 表示文件系统条目
type Entry struct {
	Stats        // 继承Stats结构体
	name  string // 文件名
}

// NewEntry 创建新的Entry实例
func NewEntry(name string, mode uint32, size int64, mtime time.Time) *Entry {
	return &Entry{
		Stats: Stats{
			mode:  mode,
			size:  size,
			mtime: mtime,
		},
		name: name,
	}
}

// Name 获取文件名
func (e *Entry) Name() string {
	return e.name
}

// String 实现Stringer接口
func (e *Entry) String() string {
	return e.name
}
