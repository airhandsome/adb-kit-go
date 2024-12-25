package hosttransport

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

const (
	OKAY = "OKAY"
	FAIL = "FAIL"
)

// ClearCommand 实现清除应用数据命令
type ClearCommand struct {
	sender func(string) error
	reader func(int) (string, error)
}

// NewClearCommand 创建新的清除命令实例
func NewClearCommand(sender func(string) error, reader func(int) (string, error)) *ClearCommand {
	return &ClearCommand{
		sender: sender,
		reader: reader,
	}
}

// Execute 执行清除应用数据命令
func (c *ClearCommand) Execute(pkg string) (bool, error) {
	cmd := fmt.Sprintf("shell:pm clear %s", pkg)
	if err := c.sender(cmd); err != nil {
		return false, fmt.Errorf("发送清除命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return false, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		result, err := c.searchLine(regexp.MustCompile(`^(Success|Failed)$`))
		if err != nil {
			return false, fmt.Errorf("读取结果失败: %v", err)
		}

		switch result {
		case "Success":
			return true, nil
		case "Failed":
			return false, fmt.Errorf("package '%s' could not be cleared", pkg)
		default:
			return false, fmt.Errorf("unexpected result: %s", result)
		}

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return false, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return false, fmt.Errorf(errMsg)

	default:
		return false, fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

// searchLine 在响应中搜索匹配的行
func (c *ClearCommand) searchLine(pattern *regexp.Regexp) (string, error) {
	// 创建一个通道用于传递结果
	resultChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// 启动一个goroutine来读取和搜索响应
	go func() {
		buffer := make([]byte, 1024)
		var builder strings.Builder

		for {
			n, err := c.reader(len(buffer))
			if err != nil {
				errChan <- fmt.Errorf("读取响应失败: %v", err)
				return
			}

			if n == "" {
				break
			}

			builder.WriteString(n)

			// 使用Scanner逐行扫描
			scanner := bufio.NewScanner(strings.NewReader(builder.String()))
			for scanner.Scan() {
				line := scanner.Text()
				if pattern.MatchString(line) {
					resultChan <- line
					return
				}
			}

			if err := scanner.Err(); err != nil {
				errChan <- fmt.Errorf("扫描响应失败: %v", err)
				return
			}
		}

		errChan <- fmt.Errorf("未找到匹配的行")
	}()

	// 等待结果或错误
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return "", err
	}
}

// 清理资源
func (c *ClearCommand) Close() error {
	// 实现任何必要的清理逻辑
	return nil
}
