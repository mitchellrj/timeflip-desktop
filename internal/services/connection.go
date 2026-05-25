package services

import (
	"context"
	"sync"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
)

type ConnectionManager struct {
	devices *DeviceService
	config  domain.AppConfig
	store   interface {
		ListDeviceProfiles(context.Context) ([]domain.DeviceProfile, error)
		SaveDeviceState(context.Context, domain.DeviceState) error
	}
	bus           EventBus
	clock         Clock
	mu            sync.Mutex
	failureCounts map[string]int
	firstFailure  map[string]time.Time
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

func NewConnectionManager(devices *DeviceService, store interface {
	ListDeviceProfiles(context.Context) ([]domain.DeviceProfile, error)
	SaveDeviceState(context.Context, domain.DeviceState) error
}, cfg domain.AppConfig, bus EventBus, clock Clock) *ConnectionManager {
	if cfg.ReconnectPolicy.OfflineAfterFailures == 0 {
		cfg.ReconnectPolicy = domain.DefaultReconnectPolicy()
	}
	if bus == nil {
		bus = NoopEventBus{}
	}
	if clock == nil {
		clock = SystemClock{}
	}
	return &ConnectionManager{
		devices:       devices,
		store:         store,
		config:        cfg,
		bus:           bus,
		clock:         clock,
		failureCounts: map[string]int{},
		firstFailure:  map[string]time.Time{},
	}
}

func (m *ConnectionManager) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	profiles, err := m.store.ListDeviceProfiles(ctx)
	if err != nil {
		return err
	}
	for _, profile := range profiles {
		if profile.StoredPassword == "" || profile.PairingState == "unpaired" {
			continue
		}
		m.wg.Add(1)
		go func(deviceID string) {
			defer m.wg.Done()
			m.reconnectLoop(ctx, deviceID)
		}(profile.ID)
	}
	return nil
}

func (m *ConnectionManager) ConnectDevice(ctx context.Context, deviceID string) error {
	if err := m.devices.ConnectDevice(ctx, deviceID); err != nil {
		m.recordFailure(ctx, deviceID)
		return err
	}
	m.mu.Lock()
	delete(m.failureCounts, deviceID)
	delete(m.firstFailure, deviceID)
	m.mu.Unlock()
	return nil
}

func (m *ConnectionManager) HandleDisconnect(deviceID string, reason string) {
	m.bus.Publish(context.Background(), "device.connection", domain.DeviceState{DeviceID: deviceID, ConnectionState: domain.ConnectionReconnecting, SystemStatus: reason, UpdatedAt: m.clock.Now()})
}

func (m *ConnectionManager) ScheduleReconnect(deviceID string) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	first := m.firstFailure[deviceID]
	if first.IsZero() {
		return 0
	}
	elapsed := m.clock.Now().Sub(first)
	if elapsed < 2*time.Minute {
		return m.config.ReconnectPolicy.InitialRetryInterval
	}
	if elapsed < 15*time.Minute {
		return m.config.ReconnectPolicy.MediumRetryInterval
	}
	return m.config.ReconnectPolicy.LongRetryInterval
}

func (m *ConnectionManager) MarkOfflineIfThresholdReached(ctx context.Context, deviceID string) bool {
	m.mu.Lock()
	failures := m.failureCounts[deviceID]
	first := m.firstFailure[deviceID]
	m.mu.Unlock()
	if failures >= m.config.ReconnectPolicy.OfflineAfterFailures || (!first.IsZero() && m.clock.Now().Sub(first) >= m.config.ReconnectPolicy.OfflineAfterDuration) {
		state := domain.DeviceState{DeviceID: deviceID, ConnectionState: domain.ConnectionOffline, UpdatedAt: m.clock.Now()}
		_ = m.store.SaveDeviceState(ctx, state)
		m.bus.Publish(ctx, "device.connection", state)
		return true
	}
	return false
}

func (m *ConnectionManager) Stop(ctx context.Context) {
	if m.cancel != nil {
		m.cancel()
	}
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
	case <-done:
	}
}

func (m *ConnectionManager) reconnectLoop(ctx context.Context, deviceID string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if _, ok := m.devices.currentHandle(deviceID); ok {
			if !sleepContext(ctx, retryIntervalOrDefault(m.config.ReconnectPolicy.InitialRetryInterval)) {
				return
			}
			continue
		}
		if err := m.ConnectDevice(ctx, deviceID); err == nil {
			if !sleepContext(ctx, retryIntervalOrDefault(m.config.ReconnectPolicy.InitialRetryInterval)) {
				return
			}
			continue
		}
		m.MarkOfflineIfThresholdReached(ctx, deviceID)
		delay := m.ScheduleReconnect(deviceID)
		if delay == 0 {
			delay = m.config.ReconnectPolicy.InitialRetryInterval
		}
		if !sleepContext(ctx, retryIntervalOrDefault(delay)) {
			return
		}
	}
}

func retryIntervalOrDefault(delay time.Duration) time.Duration {
	if delay > 0 {
		return delay
	}
	return domain.DefaultReconnectPolicy().InitialRetryInterval
}

func sleepContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (m *ConnectionManager) recordFailure(ctx context.Context, deviceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failureCounts[deviceID]++
	if m.firstFailure[deviceID].IsZero() {
		m.firstFailure[deviceID] = m.clock.Now()
	}
	m.bus.Publish(ctx, "device.connection", domain.DeviceState{DeviceID: deviceID, ConnectionState: domain.ConnectionReconnecting, UpdatedAt: m.clock.Now()})
}
