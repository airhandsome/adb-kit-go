package sync

import (
	"time"
)

// Stats 实现文件统计信息
type Stats struct {
	mode  uint32    // 文件模式
	size  int64     // 文件大小
	mtime time.Time // 修改时间
}

// 文件类型常量
const (
	S_IFMT   = 0o170000 // 文件类型位掩码
	S_IFSOCK = 0o140000 // socket
	S_IFLNK  = 0o120000 // 符号链接
	S_IFREG  = 0o100000 // 普通文件
	S_IFBLK  = 0o060000 // 块设备
	S_IFDIR  = 0o040000 // 目录
	S_IFCHR  = 0o020000 // 字符设备
	S_IFIFO  = 0o010000 // FIFO
	S_ISUID  = 0o004000 // set UID
	S_ISGID  = 0o002000 // set-group-ID
	S_ISVTX  = 0o001000 // sticky bit
	S_IRWXU  = 0o0700   // 用户权限掩码
	S_IRUSR  = 0o0400   // 用户读权限
	S_IWUSR  = 0o0200   // 用户写权限
	S_IXUSR  = 0o0100   // 用户执行权限
	S_IRWXG  = 0o0070   // 组权限掩码
	S_IRGRP  = 0o0040   // 组读权限
)

// NewStats 创建新的Stats实例
func NewStats(mode uint32, size int64, mtime time.Time) *Stats {
	return &Stats{
		mode:  mode,
		size:  size,
		mtime: mtime,
	}
}

// Mode 获取文件模式
func (s *Stats) Mode() uint32 {
	return s.mode
}

// Size 获取文件大小
func (s *Stats) Size() int64 {
	return s.size
}

// ModTime 获取修改时间
func (s *Stats) ModTime() time.Time {
	return s.mtime
}

// IsSocket 判断是否为socket
func (s *Stats) IsSocket() bool {
	return (s.mode & S_IFMT) == S_IFSOCK
}

// IsSymlink 判断是否为符号链接
func (s *Stats) IsSymlink() bool {
	return (s.mode & S_IFMT) == S_IFLNK
}

// IsRegular 判断是否为普通文件
func (s *Stats) IsRegular() bool {
	return (s.mode & S_IFMT) == S_IFREG
}

// IsBlock 判断是否为块设备
func (s *Stats) IsBlock() bool {
	return (s.mode & S_IFMT) == S_IFBLK
}

// IsDir 判断是否为目录
func (s *Stats) IsDir() bool {
	return (s.mode & S_IFMT) == S_IFDIR
}

// IsCharacter 判断是否为字符设备
func (s *Stats) IsCharacter() bool {
	return (s.mode & S_IFMT) == S_IFCHR
}

// IsFifo 判断是否为FIFO
func (s *Stats) IsFifo() bool {
	return (s.mode & S_IFMT) == S_IFIFO
}

// IsSetuid 判断是否设置了setuid位
func (s *Stats) IsSetuid() bool {
	return (s.mode & S_ISUID) != 0
}

// IsSetgid 判断是否设置了setgid位
func (s *Stats) IsSetgid() bool {
	return (s.mode & S_ISGID) != 0
}

// IsSticky 判断是否设置了sticky位
func (s *Stats) IsSticky() bool {
	return (s.mode & S_ISVTX) != 0
}

// UserPermissions 获取用户权限
func (s *Stats) UserPermissions() uint32 {
	return (s.mode & S_IRWXU) >> 6
}

// GroupPermissions 获取组权限
func (s *Stats) GroupPermissions() uint32 {
	return (s.mode & S_IRWXG) >> 3
}

// OtherPermissions 获取其他用户权限
func (s *Stats) OtherPermissions() uint32 {
	return s.mode & 0o007
}

// HasUserRead 判断是否有用户读权限
func (s *Stats) HasUserRead() bool {
	return (s.mode & S_IRUSR) != 0
}

// HasUserWrite 判断是否有用户写权限
func (s *Stats) HasUserWrite() bool {
	return (s.mode & S_IWUSR) != 0
}

// HasUserExecute 判断是否有用户执行权限
func (s *Stats) HasUserExecute() bool {
	return (s.mode & S_IXUSR) != 0
}

// HasGroupRead 判断是否有组读权限
func (s *Stats) HasGroupRead() bool {
	return (s.mode & S_IRGRP) != 0
}

// IsFile 判断是否为普通文件
func (s *Stats) IsFile() bool {
	return s.mode&0x8000 != 0
}

// Permissions 获取权限位
func (s *Stats) Permissions() uint32 {
	return s.mode & 0x1FF
}
