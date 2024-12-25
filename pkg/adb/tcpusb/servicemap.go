package tcpusb

import (
	"fmt"
	"sync"
)

// ServiceMap 实现服务映射管理
type ServiceMap struct {
	remotes map[uint32]*Service
	count   int
	mu      sync.RWMutex
}

// NewServiceMap 创建新的服务映射实例
func NewServiceMap() *ServiceMap {
	return &ServiceMap{
		remotes: make(map[uint32]*Service),
		count:   0,
	}
}

// End 结束所有服务
func (m *ServiceMap) End() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 结束所有服务
	for _, remote := range m.remotes {
		remote.End()
	}

	// 清空映射
	m.remotes = make(map[uint32]*Service)
	m.count = 0
}

// Insert 插入新服务
func (m *ServiceMap) Insert(remoteId uint32, socket *Service) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查ID是否已存在
	if _, exists := m.remotes[remoteId]; exists {
		return fmt.Errorf("Remote ID %d is already being used", remoteId)
	}

	// 插入新服务
	m.remotes[remoteId] = socket
	m.count++
	return nil
}

// Get 获取服务
func (m *ServiceMap) Get(remoteId uint32) *Service {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.remotes[remoteId]
}

// Remove 移除服务
func (m *ServiceMap) Remove(remoteId uint32) *Service {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取并移除服务
	if remote, exists := m.remotes[remoteId]; exists {
		delete(m.remotes, remoteId)
		m.count--
		return remote
	}

	return nil
}

// Count 获取服务数量
func (m *ServiceMap) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.count
}

// List 列出所有服务
func (m *ServiceMap) List() []uint32 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]uint32, 0, len(m.remotes))
	for id := range m.remotes {
		ids = append(ids, id)
	}
	return ids
}

// HasService 检查服务是否存在
func (m *ServiceMap) HasService(remoteId uint32) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.remotes[remoteId]
	return exists
}

// Clear 清空所有服务
func (m *ServiceMap) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.remotes = make(map[uint32]*Service)
	m.count = 0
}

// ForEach 遍历所有服务
func (m *ServiceMap) ForEach(fn func(uint32, *Service)) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, service := range m.remotes {
		fn(id, service)
	}
}

// EndService 结束指定服务
func (m *ServiceMap) EndService(remoteId uint32) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if service, exists := m.remotes[remoteId]; exists {
		service.End()
		delete(m.remotes, remoteId)
		m.count--
		return true
	}
	return false
}
