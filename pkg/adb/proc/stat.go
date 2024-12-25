package proc

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// CPUStats 存储CPU统计信息
type CPUStats struct {
	Line      string
	User      uint64
	Nice      uint64
	System    uint64
	Idle      uint64
	IOWait    uint64
	IRQ       uint64
	SoftIRQ   uint64
	Steal     uint64
	Guest     uint64
	GuestNice uint64
	Total     uint64
}

// CPULoad 存储CPU负载百分比
type CPULoad struct {
	User      int
	Nice      int
	System    int
	Idle      int
	IOWait    int
	IRQ       int
	SoftIRQ   int
	Steal     int
	Guest     int
	GuestNice int
	Total     int
}

type ProcStat struct {
	interval time.Duration
	stats    map[string]*CPUStats
	ignore   map[string]string
	done     chan bool
	OnLoad   func(map[string]*CPULoad)
	OnError  func(error)
}

func NewProcStat() *ProcStat {
	return &ProcStat{
		interval: time.Second,
		stats:    make(map[string]*CPUStats),
		ignore:   make(map[string]string),
		done:     make(chan bool),
	}
}

func (p *ProcStat) Start() {
	ticker := time.NewTicker(p.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				p.update()
			case <-p.done:
				ticker.Stop()
				return
			}
		}
	}()
	p.update()
}

func (p *ProcStat) Stop() {
	p.done <- true
}

func (p *ProcStat) update() {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		if p.OnError != nil {
			p.OnError(err)
		}
		return
	}
	p.parse(string(data))
}

func (p *ProcStat) parse(data string) {
	newStats := make(map[string]*CPUStats)
	scanner := bufio.NewScanner(strings.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		cpuID := fields[0]
		if p.ignore[cpuID] == line {
			continue
		}

		stats := &CPUStats{Line: line}
		values := make([]uint64, 10)

		for i := 0; i < 10; i++ {
			val, err := strconv.ParseUint(fields[i+1], 10, 64)
			if err != nil {
				if p.OnError != nil {
					p.OnError(fmt.Errorf("解析CPU数据失败: %v", err))
				}
				return
			}
			values[i] = val
		}

		stats.User = values[0]
		stats.Nice = values[1]
		stats.System = values[2]
		stats.Idle = values[3]
		stats.IOWait = values[4]
		stats.IRQ = values[5]
		stats.SoftIRQ = values[6]
		stats.Steal = values[7]
		stats.Guest = values[8]
		stats.GuestNice = values[9]

		for _, v := range values {
			stats.Total += v
		}

		newStats[cpuID] = stats
	}

	p.calculateLoad(newStats)
}

func (p *ProcStat) calculateLoad(newStats map[string]*CPUStats) {
	loads := make(map[string]*CPULoad)
	found := false

	for id, cur := range newStats {
		old, exists := p.stats[id]
		if !exists {
			continue
		}

		ticks := cur.Total - old.Total
		if ticks > 0 {
			found = true
			m := float64(100) / float64(ticks)

			loads[id] = &CPULoad{
				User:      int(m * float64(cur.User-old.User)),
				Nice:      int(m * float64(cur.Nice-old.Nice)),
				System:    int(m * float64(cur.System-old.System)),
				Idle:      int(m * float64(cur.Idle-old.Idle)),
				IOWait:    int(m * float64(cur.IOWait-old.IOWait)),
				IRQ:       int(m * float64(cur.IRQ-old.IRQ)),
				SoftIRQ:   int(m * float64(cur.SoftIRQ-old.SoftIRQ)),
				Steal:     int(m * float64(cur.Steal-old.Steal)),
				Guest:     int(m * float64(cur.Guest-old.Guest)),
				GuestNice: int(m * float64(cur.GuestNice-old.GuestNice)),
				Total:     100,
			}
		} else {
			p.ignore[id] = cur.Line
			delete(newStats, id)
		}
	}

	if found && p.OnLoad != nil {
		p.OnLoad(loads)
	}
	p.stats = newStats
}
