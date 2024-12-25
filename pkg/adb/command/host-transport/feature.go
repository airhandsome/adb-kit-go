package hosttransport

import (
	"fmt"
	"regexp"
	"strings"
)

// GetFeaturesCommand 实现获取特性命令
type GetFeaturesCommand struct {
	BaseCommand
}

type Feature struct {
	Name  string
	Value interface{}
}

func NewGetFeaturesCommand(sender func(string) error, reader func(int) (string, error)) *GetFeaturesCommand {
	return &GetFeaturesCommand{
		BaseCommand: BaseCommand{
			sender: sender,
			reader: reader,
		},
	}
}

func (c *GetFeaturesCommand) Execute() (map[string]interface{}, error) {
	if err := c.sender("shell:pm list features 2>/dev/null"); err != nil {
		return nil, fmt.Errorf("发送获取特性命令失败: %v", err)
	}

	reply, err := c.reader(4)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	switch reply {
	case OKAY:
		data, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取特性数据失败: %v", err)
		}
		return c.parseFeatures(data)

	case FAIL:
		errMsg, err := c.reader(0)
		if err != nil {
			return nil, fmt.Errorf("读取错误信息失败: %v", err)
		}
		return nil, fmt.Errorf(errMsg)

	default:
		return nil, fmt.Errorf("unexpected response: %s, expected OKAY or FAIL", reply)
	}
}

func (c *GetFeaturesCommand) parseFeatures(value string) (map[string]interface{}, error) {
	features := make(map[string]interface{})
	re := regexp.MustCompile(`^feature:(.*?)(?:=(.*?))?\r?$`)

	lines := strings.Split(value, "\n")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			name := matches[1]
			if len(matches) == 3 && matches[2] != "" {
				features[name] = matches[2]
			} else {
				features[name] = true
			}
		}
	}

	return features, nil
}
