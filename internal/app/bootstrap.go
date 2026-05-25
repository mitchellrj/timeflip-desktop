package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mitchellrj/timeflip-desktop/internal/device"
	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/mitchellrj/timeflip-desktop/internal/services"
	"github.com/mitchellrj/timeflip-desktop/internal/store"
)

func BuildController(ctx context.Context) (*Controller, *WailsEventBus, error) {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		dataDir = "."
	}
	appDir := filepath.Join(dataDir, "timeflip-desktop")
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return nil, nil, err
	}
	dbPath := filepath.Join(appDir, "timeflip-desktop.sqlite")
	st, err := store.OpenSQLite(dbPath)
	if err != nil {
		return nil, nil, err
	}
	if err := st.Migrate(ctx); err != nil {
		_ = st.Close()
		return nil, nil, err
	}
	cfgSvc := services.NewConfigService(st)
	cfg, err := cfgSvc.Load(ctx)
	if err != nil {
		_ = st.Close()
		return nil, nil, err
	}
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = dbPath
		_ = cfgSvc.Save(ctx, cfg)
	}
	bus := NewWailsEventBus()
	clock := services.SystemClock{}
	devClient, err := device.NewNativeDeviceClient(domain.DefaultTimeout)
	if err != nil {
		_ = st.Close()
		return nil, nil, err
	}
	taskSvc := services.NewTaskService(st, clock)
	trackingSvc := services.NewTrackingService(st, clock, bus)
	historySvc := services.NewHistoryService(st, devClient, trackingSvc)
	deviceSvc := services.NewDeviceService(devClient, st, taskSvc, trackingSvc, historySvc, bus, clock)
	connection := services.NewConnectionManager(deviceSvc, st, cfg, bus, clock)
	return NewController(st, cfgSvc, deviceSvc, taskSvc, historySvc, connection), bus, nil
}
