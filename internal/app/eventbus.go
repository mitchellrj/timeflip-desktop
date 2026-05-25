package app

import (
	"context"
	"sync"
)

type WailsEventBus struct {
	mu      sync.RWMutex
	emitter func(string, any)
}

func NewWailsEventBus() *WailsEventBus {
	return &WailsEventBus{}
}

func (b *WailsEventBus) SetEmitter(emitter func(string, any)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.emitter = emitter
}

func (b *WailsEventBus) Publish(_ context.Context, name string, payload any) {
	b.mu.RLock()
	emitter := b.emitter
	b.mu.RUnlock()
	if emitter == nil {
		return
	}
	emitter(name, payload)
}
