// Package registry 管理房间 → 会话的路由。
// M1 为进程内实现；M2 增加 Redis 实现（room_id → {gateway_node, worker_id, ttl}），
// 支撑跨实例重连重挂。接口保持稳定，业务代码不感知实现。
package registry

import (
	"fmt"
	"sync"

	"media-gateway/internal/relay"
)

// Registry 会话注册表
type Registry interface {
	// Bind 注册会话；房间已存在且未关闭时返回错误（防重复接管）
	Bind(roomID string, s *relay.Session) error
	// Lookup 查找活跃会话（重连 resume 用）
	Lookup(roomID string) (*relay.Session, bool)
	// Unbind 注销
	Unbind(roomID string)
}

// LocalRegistry 进程内实现（M1）
type LocalRegistry struct {
	mu      sync.RWMutex
	sessions map[string]*relay.Session
}

func NewLocal() *LocalRegistry {
	return &LocalRegistry{sessions: make(map[string]*relay.Session)}
}

func (r *LocalRegistry) Bind(roomID string, s *relay.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if exist, ok := r.sessions[roomID]; ok {
		if closed, _ := exist.Closed(); !closed {
			return fmt.Errorf("房间已有活跃会话: %s", roomID)
		}
	}
	r.sessions[roomID] = s
	return nil
}

func (r *LocalRegistry) Lookup(roomID string) (*relay.Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sessions[roomID]
	return s, ok
}

func (r *LocalRegistry) Unbind(roomID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, roomID)
}
