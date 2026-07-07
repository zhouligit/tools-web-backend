package store

import (
	"sync"
	"time"

	"github.com/find-work/tools-web-backend/internal/model"
)

type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*model.Task
}

func NewTaskStore() *TaskStore {
	return &TaskStore{tasks: make(map[string]*model.Task)}
}

func (s *TaskStore) Save(task *model.Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
}

func (s *TaskStore) Get(id string) (*model.Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	return task, ok
}

func (s *TaskStore) List(limit int) []model.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		items = append(items, *t)
	}
	if limit > 0 && len(items) > limit {
		return items[:limit]
	}
	return items
}

func (s *TaskStore) Update(id string, fn func(*model.Task)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return false
	}
	fn(task)
	task.UpdatedAt = time.Now()
	return true
}
