package services

import (
	"context"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/mitchellrj/timeflip-desktop/internal/store"
)

type ConfigService struct {
	store store.Store
}

func NewConfigService(store store.Store) *ConfigService {
	return &ConfigService{store: store}
}

func (s *ConfigService) Load(ctx context.Context) (domain.AppConfig, error) {
	if s == nil || s.store == nil {
		return domain.DefaultAppConfig(), nil
	}
	return s.store.LoadConfig(ctx)
}

func (s *ConfigService) Save(ctx context.Context, config domain.AppConfig) error {
	if config.ReconnectPolicy.OfflineAfterFailures == 0 {
		config.ReconnectPolicy = domain.DefaultReconnectPolicy()
	}
	if config.CommunicationTimeout == 0 {
		config.CommunicationTimeout = domain.DefaultTimeout
	}
	if config.CommandTimeout == 0 {
		config.CommandTimeout = 5 * time.Second
	}
	return s.store.SaveConfig(ctx, config)
}
