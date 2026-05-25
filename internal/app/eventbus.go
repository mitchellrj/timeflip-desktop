package app

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type WailsEventBus struct {
	ctx context.Context
}

func NewWailsEventBus() *WailsEventBus {
	return &WailsEventBus{}
}

func (b *WailsEventBus) SetContext(ctx context.Context) {
	b.ctx = ctx
}

func (b *WailsEventBus) Publish(ctx context.Context, name string, payload any) {
	target := ctx
	if target == nil || target.Err() != nil {
		target = b.ctx
	}
	if target == nil {
		return
	}
	runtime.EventsEmit(target, name, payload)
}
