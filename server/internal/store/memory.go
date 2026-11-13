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

func (m *Memory) Get(code string) (*model.Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[code]
	return r, ok
}

// Add зберігає кімнату, якщо код вільний. Повертає false, якщо код уже зайнятий.
func (m *Memory) Add(r *model.Room) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, taken := m.rooms[r.Code]; taken {
		return false
	}
	m.rooms[r.Code] = r
	return true
}

func (m *Memory) Delete(code string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rooms, code)
}
