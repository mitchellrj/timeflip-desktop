package services

import (
	"context"
	"sync"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
)

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type EventBus interface {
	Publish(context.Context, string, any)
}

type NoopEventBus struct{}

func (NoopEventBus) Publish(context.Context, string, any) {}

type MemoryEventBus struct {
	mu     sync.Mutex
	Events []PublishedEvent
}

type PublishedEvent struct {
	Name    string
	Payload any
}

func (b *MemoryEventBus) Publish(_ context.Context, name string, payload any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Events = append(b.Events, PublishedEvent{Name: name, Payload: payload})
}

type AppState struct {
	Config         domain.AppConfig                `json:"config"`
	Devices        []domain.DeviceProfileView      `json:"devices"`
	States         []domain.DeviceState            `json:"states"`
	Tasks          []domain.Task                   `json:"tasks"`
	Sessions       []domain.TaskSession            `json:"sessions"`
	FacetConfigs   []domain.FacetConfigurationView `json:"facetConfigs"`
	CurrentSession *domain.TaskSession             `json:"currentSession,omitempty"`
}
