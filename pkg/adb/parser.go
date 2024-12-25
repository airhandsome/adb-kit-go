package adb

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
)

// Parser ADB数据解析器
type Parser struct {
	stream io.ReadCloser
	ended  bool
}

// NewParser 创建新的解析器
func NewParser(stream io.ReadCloser) *Parser {
	return &Parser{
		stream: stream,
		ended:  false,
	}
}

// End 结束解析
func (p *Parser) End() error {
	if p.ended {
		return nil
	}

	p.ended = true
	return p.stream.Close()
}

// Raw 获取原始流
func (p *Parser) Raw() io.ReadCloser {
	return p.stream
}

// ReadAll 读取所有数据
func (p *Parser) ReadAll() ([]byte, error) {
	var buffer bytes.Buffer

	// 读取数据直到流结束
	_, err := io.Copy(&buffer, p.stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read all data: %v", err)
	}

	p.ended = true
	return buffer.Bytes(), nil
}

// ReadAscii 读取指定长度的ASCII字符串
func (p *Parser) ReadAscii(length int) (string, error) {
	data, err := p.ReadBytes(length)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ReadBytes 读取指定长度的字节
func (p *Parser) ReadBytes(length int) ([]byte, error) {
	if length == 0 {
		return []byte{}, nil
	}

	buffer := make([]byte, length)
	n, err := io.ReadFull(p.stream, buffer)

	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil, &PrematureEOFError{
				MissingBytes: length - n,
			}
		}
		return nil, fmt.Errorf("failed to read bytes: %v", err)
	}

	return buffer, nil
}

// ReadByteFlow 读取指定长度的字节流到目标Writer
func (p *Parser) ReadByteFlow(length int, target io.Writer) error {
	if length == 0 {
		return nil
	}

	n, err := io.CopyN(target, p.stream, int64(length))
	if err != nil {
		if err == io.EOF {
			return &PrematureEOFError{
				MissingBytes: length - int(n),
			}
		}
		return fmt.Errorf("failed to copy bytes: %v", err)
	}

	return nil
}

// ReadError 读取错误信息
func (p *Parser) ReadError() error {
	value, err := p.ReadValue()
	if err != nil {
		return err
	}
	return &FailError{Message: string(value)}
}

// ReadValue 读取值
func (p *Parser) ReadValue() ([]byte, error) {
	// 读取4字节的长度
	lenBytes, err := p.ReadAscii(4)
	if err != nil {
		return nil, err
	}

	// 解码长度
	pro := &Protocol{}
	length, err := pro.DecodeLength(lenBytes)
	if err != nil {
		return nil, err
	}

	// 读取数据
	return p.ReadBytes(length)
}

// ReadUntil 读取直到指定字节
func (p *Parser) ReadUntil(code byte) ([]byte, error) {
	var buffer bytes.Buffer

	for {
		b, err := p.ReadBytes(1)
		if err != nil {
			return nil, err
		}

		if b[0] == code {
			return buffer.Bytes(), nil
		}

		buffer.Write(b)
	}
}

// SearchLine 搜索匹配正则表达式的行
func (p *Parser) SearchLine(re *regexp.Regexp) ([]string, error) {
	for {
		line, err := p.ReadLine()
		if err != nil {
			return nil, err
		}

		if matches := re.FindStringSubmatch(string(line)); matches != nil {
			return matches, nil
		}
	}
}

// ReadLine 读取一行
func (p *Parser) ReadLine() ([]byte, error) {
	line, err := p.ReadUntil(0x0a) // '\n'
	if err != nil {
		return nil, err
	}

	// 移除行尾的'\r'
	if len(line) > 0 && line[len(line)-1] == 0x0d {
		line = line[:len(line)-1]
	}

	return line, nil
}

// Unexpected 生成意外数据错误
func (p *Parser) Unexpected(data []byte, expected string) error {
	return &UnexpectedDataError{
		Unexpected: string(data),
		Expected:   expected,
	}
}

// Error 类型定义
type (
	// FailError 失败错误
	FailError struct {
		Message string
	}

	// PrematureEOFError 过早EOF错误
	PrematureEOFError struct {
		MissingBytes int
	}

	// UnexpectedDataError 意外数据错误
	UnexpectedDataError struct {
		Unexpected string
		Expected   string
	}
)

// Error 实现
func (e *FailError) Error() string {
	return fmt.Sprintf("Failure: '%s'", e.Message)
}

func (e *PrematureEOFError) Error() string {
	return fmt.Sprintf("Premature end of stream, needed %d more bytes", e.MissingBytes)
}

func (e *UnexpectedDataError) Error() string {
	return fmt.Sprintf("Unexpected '%s', was expecting %s", e.Unexpected, e.Expected)
}
