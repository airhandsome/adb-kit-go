package adb

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Sync ADB同步管理器
type Sync struct {
	conn     *Connection
	parser   *Parser
	protocol *Protocol
}

// 常量定义
const (
	TEMP_PATH       = "/data/local/tmp"
	DEFAULT_CHMOD   = 0644
	DATA_MAX_LENGTH = 65536
)

// NewSync 创建新的同步管理器
func NewSync(conn *Connection) *Sync {
	return &Sync{
		conn:     conn,
		parser:   conn.GetParser(),
		protocol: NewProtocol(),
	}
}

// TempFile 生成临时文件路径
func (s *Sync) TempFile(path string) string {
	return filepath.Join(TEMP_PATH, filepath.Base(path))
}

// Stat 获取文件状态
func (s *Sync) Stat(path string) (*Stats, error) {
	// 发送STAT命令
	err := s.sendCommandWithArg(STAT, path)
	if err != nil {
		return nil, err
	}

	// 读取响应
	reply, err := s.parser.ReadAscii(4)
	if err != nil {
		return nil, err
	}

	switch reply {
	case STAT:
		// 读取文件状态信息
		statData, err := s.parser.ReadBytes(12)
		if err != nil {
			return nil, err
		}

		mode := binary.LittleEndian.Uint32(statData[0:4])
		size := binary.LittleEndian.Uint32(statData[4:8])
		mtime := binary.LittleEndian.Uint32(statData[8:12])

		if mode == 0 {
			return nil, s.enoent(path)
		}

		return NewStats(mode, size, mtime), nil

	case FAIL:
		return nil, s.readError()

	default:
		return nil, s.parser.Unexpected([]byte(reply), "STAT or FAIL")
	}
}

// Push 推送文件或流到设备
func (s *Sync) Push(src interface{}, destPath string, mode os.FileMode) (*PushTransfer, error) {
	if mode == 0 {
		mode = DEFAULT_CHMOD
	}

	switch v := src.(type) {
	case string:
		return s.PushFile(v, destPath, mode)
	case io.Reader:
		return s.PushStream(v, destPath, mode)
	default:
		return nil, fmt.Errorf("unsupported source type")
	}
}

// PushFile 推送文件到设备
func (s *Sync) PushFile(srcPath, destPath string, mode os.FileMode) (*PushTransfer, error) {
	file, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}

	return s.PushStream(file, destPath, mode)
}

// PushStream 推送数据流到设备
func (s *Sync) PushStream(stream io.Reader, destPath string, mode os.FileMode) (*PushTransfer, error) {
	// 设置文件模式
	mode |= Stats.S_IFREG

	// 发送SEND命令
	err := s.sendCommandWithArg(SEND, fmt.Sprintf("%s,%d", destPath, mode))
	if err != nil {
		return nil, err
	}

	// 创建传输对象
	transfer := NewPushTransfer()

	// 开始数据传输
	go s.writeData(stream, time.Now().Unix(), transfer)

	return transfer, nil
}

// Pull 从设备拉取文件
func (s *Sync) Pull(path string) (*PullTransfer, error) {
	// 发送RECV命令
	err := s.sendCommandWithArg(RECV, path)
	if err != nil {
		return nil, err
	}

	// 创建传输对象
	transfer := NewPullTransfer()

	// 开始数据传输
	go s.readData(transfer)

	return transfer, nil
}

// writeData 写入数据到设备
func (s *Sync) writeData(stream io.Reader, timestamp int64, transfer *PushTransfer) {
	buffer := make([]byte, DATA_MAX_LENGTH)

	for {
		// 读取数据块
		n, err := stream.Read(buffer)
		if err != nil && err != io.EOF {
			transfer.EmitError(err)
			return
		}

		if n > 0 {
			// 发送DATA命令
			err = s.sendCommandWithLength(DATA, n)
			if err != nil {
				transfer.EmitError(err)
				return
			}

			// 发送数据
			transfer.Push(n)
			_, err = s.conn.Write(buffer[:n])
			if err != nil {
				transfer.EmitError(err)
				return
			}
		}

		if err == io.EOF {
			break
		}
	}

	// 发送DONE命令
	err := s.sendCommandWithLength(DONE, int(timestamp))
	if err != nil {
		transfer.EmitError(err)
		return
	}

	// 等待确认
	reply, err := s.parser.ReadAscii(4)
	if err != nil {
		transfer.EmitError(err)
		return
	}

	if reply != OKAY {
		transfer.EmitError(fmt.Errorf("unexpected reply: %s", reply))
		return
	}

	transfer.End()
}

// readData 从设备读取数据
func (s *Sync) readData(transfer *PullTransfer) {
	for {
		// 读取命令
		cmd, err := s.parser.ReadAscii(4)
		if err != nil {
			transfer.EmitError(err)
			return
		}

		switch cmd {
		case DATA:
			// 读取数据长度
			lenData, err := s.parser.ReadBytes(4)
			if err != nil {
				transfer.EmitError(err)
				return
			}
			length := binary.LittleEndian.Uint32(lenData)

			// 读取数据
			err = s.parser.ReadByteFlow(int(length), transfer)
			if err != nil {
				transfer.EmitError(err)
				return
			}

		case DONE:
			// 读取时间戳
			_, err := s.parser.ReadBytes(4)
			if err != nil {
				transfer.EmitError(err)
				return
			}
			transfer.End()
			return

		case FAIL:
			transfer.EmitError(s.readError())
			return

		default:
			transfer.EmitError(s.parser.Unexpected([]byte(cmd), "DATA, DONE or FAIL"))
			return
		}
	}
}

// 辅助方法
func (s *Sync) sendCommandWithLength(cmd string, length int) error {
	data := s.protocol.FormatSync(cmd, length)
	_, err := s.conn.Write(data)
	return err
}

func (s *Sync) sendCommandWithArg(cmd, arg string) error {
	data := s.protocol.FormatSyncRequest(cmd, arg)
	_, err := s.conn.Write(data)
	return err
}

func (s *Sync) readError() error {
	return s.parser.ReadError()
}

func (s *Sync) enoent(path string) error {
	return &os.PathError{
		Op:   "stat",
		Path: path,
		Err:  os.ErrNotExist,
	}
}
