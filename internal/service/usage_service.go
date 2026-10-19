package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/find-work/tools-web-backend/internal/store"
)

type UsageService struct {
	store *store.UsageStore
}

func NewUsageService(st *store.UsageStore) *UsageService {
	return &UsageService{store: st}
}

func (s *UsageService) Get(ctx context.Context, toolKey string) (int64, error) {
	if s == nil || s.store == nil {
		return 0, fmt.Errorf("mysql is not configured")
	}
	toolKey = strings.TrimSpace(toolKey)
	if toolKey == "" {
		return 0, fmt.Errorf("tool key is required")
	}
	return s.store.Get(ctx, toolKey)
}

func (s *UsageService) Increment(ctx context.Context, toolKey string) (int64, error) {
	if s == nil || s.store == nil {
		return 0, fmt.Errorf("mysql is not configured")
	}
	toolKey = strings.TrimSpace(toolKey)
	if toolKey == "" {
		return 0, fmt.Errorf("tool key is required")
	}
	return s.store.Increment(ctx, toolKey)
}
