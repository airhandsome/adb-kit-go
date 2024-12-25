package tcpusb

// RollingCounter 实现滚动计数器
type RollingCounter struct {
	max int
	min int
	now int
}

// NewRollingCounter 创建新的滚动计数器
func NewRollingCounter(max int, min int) *RollingCounter {
	if min == 0 {
		min = 1 // 默认最小值为1
	}
	return &RollingCounter{
		max: max,
		min: min,
		now: min,
	}
}

// Next 获取下一个计数值
func (c *RollingCounter) Next() int {
	// 如果当前值达到最大值，重置为最小值
	if c.now >= c.max {
		c.now = c.min
	}
	c.now++
	return c.now
}

// Current 获取当前计数值
func (c *RollingCounter) Current() int {
	return c.now
}

// Reset 重置计数器到最小值
func (c *RollingCounter) Reset() {
	c.now = c.min
}

// SetMax 设置最大值
func (c *RollingCounter) SetMax(max int) {
	c.max = max
	if c.now > max {
		c.now = c.min
	}
}

// SetMin 设置最小值
func (c *RollingCounter) SetMin(min int) {
	c.min = min
	if c.now < min {
		c.now = min
	}
}
