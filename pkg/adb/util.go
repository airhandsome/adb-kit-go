package adb

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Util ADB工具类
type Util struct {
	parser *Parser
	auth   *Auth
}

// NewUtil 创建新的工具实例
func NewUtil() *Util {
	return &Util{
		auth: NewAuth(),
	}
}

// ReadAll 读取所有数据
func (u *Util) ReadAll(stream io.Reader) ([]byte, error) {
	parser := NewParser(stream)
	return parser.ReadAll()
}

// ParsePublicKey 解析公钥
func (u *Util) ParsePublicKey(keyString string) ([]byte, error) {
	return u.auth.ParsePublicKey(keyString), nil
}

// WriteFile 写入文件
func (u *Util) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// ReadFile 读取文件
func (u *Util) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Exists 检查文件是否存在
func (u *Util) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir 检查是否是目录
func (u *Util) IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// MkdirAll 创建目录
func (u *Util) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Remove 删除文件
func (u *Util) Remove(path string) error {
	return os.Remove(path)
}

// RemoveAll 删除目录及其内容
func (u *Util) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// CopyFile 复制文件
func (u *Util) CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// CopyDir 复制目录
func (u *Util) CopyDir(src, dst string) error {
	// 获取源目录信息
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// 创建目标目录
	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	// 读取源目录内容
	dir, err := os.Open(src)
	if err != nil {
		return err
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	// 复制每个文件/目录
	for _, file := range files {
		srcPath := src + "/" + file.Name()
		dstPath := dst + "/" + file.Name()

		if file.IsDir() {
			err = u.CopyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err = u.CopyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// FormatSize 格式化文件大小
func (u *Util) FormatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// IsValidPath 检查路径是否有效
func (u *Util) IsValidPath(path string) bool {
	if path == "" {
		return false
	}

	// 检查路径是否包含非法字符
	for _, r := range path {
		if r == 0 {
			return false
		}
	}

	return true
}

// SanitizePath 清理路径
func (u *Util) SanitizePath(path string) string {
	return filepath.Clean(path)
}

// GetHomeDir 获取用户主目录
func (u *Util) GetHomeDir() (string, error) {
	return os.UserHomeDir()
}

// GetTempDir 获取临时目录
func (u *Util) GetTempDir() string {
	return os.TempDir()
}

// GetWorkDir 获取当前工作目录
func (u *Util) GetWorkDir() (string, error) {
	return os.Getwd()
}

// SetWorkDir 设置当前工作目录
func (u *Util) SetWorkDir(dir string) error {
	return os.Chdir(dir)
}
