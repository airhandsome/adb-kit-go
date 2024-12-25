package hosttransport

import (
	"fmt"
	"regexp"
	"strings"
)

// InstallCommand 实现APK安装命令
type InstallCommand struct {
	BaseCommand
}

// InstallError 定义安装错误
type InstallError struct {
	Apk  string
	Code string
	err  string
}

func (e *InstallError) Error() string {
	return fmt.Sprintf("%s could not be installed [%s]", e.Apk, e.Code)
}

// NewInstallCommand 创建新的安装命令实例
func NewInstallCommand(sender func(string) error, reader func(int) (string, error)) *InstallCommand {
	return &InstallCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

// Execute 执行APK安装命令
func (c *InstallCommand) Execute(apk string) error {
	// 转义路径并构建命令
	escapedPath := c.escapeCompat(apk)
	cmd := fmt.Sprintf("shell:pm install -r %s", escapedPath)

	if err := c.sender(cmd); err != nil {
		return fmt.Errorf("发送安装命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		// 搜索安装结果
		result, err := c.searchInstallResult()
		if err != nil {
			return err
		}

		// 检查安装结果
		if result.success {
			return nil
		}
		return &InstallError{
			Apk:  apk,
			Code: result.code,
			err:  fmt.Sprintf("%s could not be installed [%s]", apk, result.code),
		}

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return fmt.Errorf("读取错误信息失败: %v", err)
		}
		return fmt.Errorf(errMsg)

	default:
		return fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

type installResult struct {
	success bool
	code    string
}

// searchInstallResult 搜索安装结果
func (c *InstallCommand) searchInstallResult() (*installResult, error) {
	re := regexp.MustCompile(`^(Success|Failure \[(.*?)\])$`)

	// 创建结果通道
	resultChan := make(chan *installResult, 1)
	errChan := make(chan error, 1)

	go func() {
		var buffer strings.Builder

		for {
			data, err := c.reader(1024) // 读取块
			if err != nil {
				errChan <- fmt.Errorf("读取安装结果失败: %v", err)
				return
			}

			buffer.WriteString(data)
			lines := strings.Split(buffer.String(), "\n")

			for _, line := range lines {
				if matches := re.FindStringSubmatch(strings.TrimSpace(line)); matches != nil {
					if matches[1] == "Success" {
						resultChan <- &installResult{success: true}
						return
					} else {
						resultChan <- &installResult{
							success: false,
							code:    matches[2],
						}
						return
					}
				}
			}
		}
	}()

	// 等待结果或错误
	select {
	case result := <-resultChan:
		// 消费剩余内容以自然关闭连接
		go func() {
			for {
				_, err := c.reader(1024)
				if err != nil {
					break
				}
			}
		}()
		return result, nil
	case err := <-errChan:
		return nil, err
	}
}

// escapeCompat 转义路径中的特殊字符
func (c *InstallCommand) escapeCompat(path string) string {
	// 实现路径转义逻辑
	// 这里需要根据具体需求实现，例如：
	escaped := strings.ReplaceAll(path, " ", "\\ ")
	escaped = strings.ReplaceAll(escaped, "(", "\\(")
	escaped = strings.ReplaceAll(escaped, ")", "\\)")
	escaped = strings.ReplaceAll(escaped, "[", "\\[")
	escaped = strings.ReplaceAll(escaped, "]", "\\]")
	escaped = strings.ReplaceAll(escaped, "&", "\\&")
	escaped = strings.ReplaceAll(escaped, "|", "\\|")
	escaped = strings.ReplaceAll(escaped, ";", "\\;")
	escaped = strings.ReplaceAll(escaped, "<", "\\<")
	escaped = strings.ReplaceAll(escaped, ">", "\\>")
	escaped = strings.ReplaceAll(escaped, "$", "\\$")
	escaped = strings.ReplaceAll(escaped, "`", "\\`")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")
	return escaped
}
