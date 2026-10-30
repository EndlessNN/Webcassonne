// Package store — сховище кімнат у пам'яті процесу (ADR-0004).
package store

import (
	"sync"

	"github.com/EndlessNN/Webcassonne/server/internal/model"
)

type Memory struct {
	mu    sync.RWMutex
	rooms map[string]*model.Room
}

func NewMemory() *Memory {
	return &Memory{rooms: map[string]*model.Room{}}
}

// RoomCount використовується перевіркою здоров'я.
func (m *Memory) RoomCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rooms)
}
