package device

import (
	"context"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
)

type Client interface {
	Scan(context.Context, bool) ([]domain.DiscoveredDevice, error)
	Pair(context.Context, PairRequest) (domain.PairingWorkflow, error)
	Unpair(context.Context, UnpairRequest) (domain.UnpairingWorkflow, error)
	Connect(context.Context, ConnectRequest) (Handle, error)
	Authorize(context.Context, Handle, string) error
	ReadDeviceSnapshot(context.Context, Handle) (domain.DeviceSnapshot, error)
	WriteFacetConfiguration(context.Context, Handle, domain.FacetAssignment) (domain.FacetConfigurationView, error)
	SetPause(context.Context, Handle, bool) error
	SetLock(context.Context, Handle, bool) error
	SetAutoPause(context.Context, Handle, uint16) error
	SetTapSettings(context.Context, Handle, TapSettings) error
	ReadHistory(context.Context, Handle, HistoryRequest) ([]domain.DeviceEventRecord, error)
	Events(context.Context, Handle) (<-chan domain.DeviceEventRecord, <-chan error, error)
	Close(context.Context, Handle) error
}

type Handle interface {
	DeviceID() string
}

type PairRequest struct {
	DeviceID       string
	Password       string
	NewPassword    string
	AllowOSPairing bool
	Timeout        time.Duration
}

type UnpairRequest struct {
	DeviceID         string
	Password         string
	FactoryReset     bool
	AllowOSUnpairing bool
	Timeout          time.Duration
}

type ConnectRequest struct {
	DeviceID        string
	AdvertisedName  string
	ProtocolVersion string
	Timeout         time.Duration
}

type HistoryRequest struct {
	StartEvent uint32
	All        bool
}

type TapSettings struct {
	Configured bool
	Threshold  uint8
	Limit      uint8
	Latency    uint8
	Window     uint8
}
