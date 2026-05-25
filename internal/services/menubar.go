package services

import (
	"context"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
)

type MenuBarService struct {
	bus   EventBus
	state domain.DeviceState
}

func NewMenuBarService(bus EventBus) *MenuBarService {
	if bus == nil {
		bus = NoopEventBus{}
	}
	return &MenuBarService{bus: bus}
}

func (s *MenuBarService) UpdateMenuBarState(ctx context.Context, state domain.DeviceState) {
	s.state = state
	s.bus.Publish(ctx, "menubar.state", state)
}

func (s *MenuBarService) OpenMainWindow(ctx context.Context) {
	s.bus.Publish(ctx, "window.open", nil)
}

func (s *MenuBarService) PauseOrResumeCurrentDevice(ctx context.Context) {
	s.bus.Publish(ctx, "menubar.pauseToggle", s.state)
}
