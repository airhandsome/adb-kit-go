package adb

// 导出主要的类型和函数
type Client struct {
	host string
	port int
}

// NewClient 创建一个新的 ADB 客户端实例
// 这个函数作为包的主要入口点
func NewClient(host string, port int) *Client {
	if host == "" {
		host = "localhost"
	}
	if port == 0 {
		port = 5037
	}
	return &Client{
		host: host,
		port: port,
	}
}

// CreateClient 是 NewClient 的别名，保持与原始 API 的兼容性
func CreateClient(host string, port int) *Client {
	return NewClient(host, port)
}
