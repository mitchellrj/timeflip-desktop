package domain

import "time"

const (
	FacetCount     = 12
	DefaultTimeout = 10 * time.Second
)

type ConnectionState string

const (
	ConnectionDisconnected ConnectionState = "disconnected"
	ConnectionScanning     ConnectionState = "scanning"
	ConnectionConnecting   ConnectionState = "connecting"
	ConnectionAuthorizing  ConnectionState = "authorizing"
	ConnectionConnected    ConnectionState = "connected"
	ConnectionReconnecting ConnectionState = "reconnecting"
	ConnectionOffline      ConnectionState = "offline"
	ConnectionError        ConnectionState = "error"
)

type AppConfig struct {
	DatabasePath         string          `json:"databasePath"`
	CommunicationTimeout time.Duration   `json:"communicationTimeout"`
	CommandTimeout       time.Duration   `json:"commandTimeout"`
	ReconnectPolicy      ReconnectPolicy `json:"reconnectPolicy"`
}

type ReconnectPolicy struct {
	InitialRetryInterval time.Duration `json:"initialRetryInterval"`
	MediumRetryInterval  time.Duration `json:"mediumRetryInterval"`
	LongRetryInterval    time.Duration `json:"longRetryInterval"`
	OfflineAfterDuration time.Duration `json:"offlineAfterDuration"`
	OfflineAfterFailures int           `json:"offlineAfterFailures"`
}

func DefaultReconnectPolicy() ReconnectPolicy {
	return ReconnectPolicy{
		InitialRetryInterval: 15 * time.Second,
		MediumRetryInterval:  60 * time.Second,
		LongRetryInterval:    5 * time.Minute,
		OfflineAfterDuration: 2 * time.Minute,
		OfflineAfterFailures: 3,
	}
}

func DefaultAppConfig() AppConfig {
	return AppConfig{
		CommunicationTimeout: DefaultTimeout,
		CommandTimeout:       5 * time.Second,
		ReconnectPolicy:      DefaultReconnectPolicy(),
	}
}

type DeviceProfile struct {
	ID              string    `json:"id"`
	DisplayName     string    `json:"displayName"`
	AdvertisedName  string    `json:"advertisedName"`
	ProtocolVersion string    `json:"protocolVersion"`
	StoredPassword  string    `json:"-"`
	PairingState    string    `json:"pairingState"`
	LastSeenAt      time.Time `json:"lastSeenAt"`
	LastConnectedAt time.Time `json:"lastConnectedAt"`
}

type DeviceProfileView struct {
	ID              string    `json:"id"`
	DisplayName     string    `json:"displayName"`
	AdvertisedName  string    `json:"advertisedName"`
	ProtocolVersion string    `json:"protocolVersion"`
	PairingState    string    `json:"pairingState"`
	LastSeenAt      time.Time `json:"lastSeenAt"`
	LastConnectedAt time.Time `json:"lastConnectedAt"`
	HasPassword     bool      `json:"hasPassword"`
}

func (p DeviceProfile) View() DeviceProfileView {
	return DeviceProfileView{
		ID:              p.ID,
		DisplayName:     p.DisplayName,
		AdvertisedName:  p.AdvertisedName,
		ProtocolVersion: p.ProtocolVersion,
		PairingState:    p.PairingState,
		LastSeenAt:      p.LastSeenAt,
		LastConnectedAt: p.LastConnectedAt,
		HasPassword:     p.StoredPassword != "",
	}
}

type Task struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Icon      string    `json:"icon"`
	Color     string    `json:"color"`
	Archived  bool      `json:"archived"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type FacetAssignment struct {
	ID                   string    `json:"id"`
	DeviceID             string    `json:"deviceID"`
	Facet                uint8     `json:"facet"`
	TaskID               string    `json:"taskID"`
	TaskLabelSnapshot    string    `json:"taskLabelSnapshot"`
	TaskIconSnapshot     string    `json:"taskIconSnapshot"`
	TaskColorSnapshot    string    `json:"taskColorSnapshot"`
	IsPauseAssignment    bool      `json:"isPauseAssignment"`
	PomodoroLimitSeconds uint32    `json:"pomodoroLimitSeconds"`
	EffectiveFrom        time.Time `json:"effectiveFrom"`
	ConfirmedOnDevice    bool      `json:"confirmedOnDevice"`
}

type FacetConfigurationRequest struct {
	DeviceID             string `json:"deviceID"`
	Facet                uint8  `json:"facet"`
	TaskID               string `json:"taskID"`
	Label                string `json:"label"`
	Icon                 string `json:"icon"`
	Color                string `json:"color"`
	IsPauseAssignment    bool   `json:"isPauseAssignment"`
	PomodoroLimitSeconds uint32 `json:"pomodoroLimitSeconds"`
}

type FacetConfigurationView struct {
	Facet                uint8  `json:"facet"`
	TaskID               string `json:"taskID"`
	Label                string `json:"label"`
	Icon                 string `json:"icon"`
	Color                string `json:"color"`
	IsPauseAssignment    bool   `json:"isPauseAssignment"`
	PomodoroLimitSeconds uint32 `json:"pomodoroLimitSeconds"`
	AssignedOnDevice     bool   `json:"assignedOnDevice"`
}

type DeviceState struct {
	DeviceID              string          `json:"deviceID"`
	ConnectionState       ConnectionState `json:"connectionState"`
	CurrentFacet          uint8           `json:"currentFacet"`
	CurrentFacetKnown     bool            `json:"currentFacetKnown"`
	CurrentFacetUndefined bool            `json:"currentFacetUndefined"`
	Paused                bool            `json:"paused"`
	Locked                bool            `json:"locked"`
	BatteryPercent        uint8           `json:"batteryPercent"`
	SystemStatus          string          `json:"systemStatus"`
	UpdatedAt             time.Time       `json:"updatedAt"`
}

type DeviceEventRecord struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"deviceID"`
	Kind        string    `json:"kind"`
	Facet       uint8     `json:"facet"`
	Pause       bool      `json:"pause"`
	EventNumber uint32    `json:"eventNumber"`
	OccurredAt  time.Time `json:"occurredAt"`
	Source      string    `json:"source"`
	RawSummary  string    `json:"rawSummary"`
}

type TaskSession struct {
	ID                string     `json:"id"`
	DeviceID          string     `json:"deviceID"`
	TaskID            string     `json:"taskID"`
	TaskLabelSnapshot string     `json:"taskLabelSnapshot"`
	TaskIconSnapshot  string     `json:"taskIconSnapshot"`
	TaskColorSnapshot string     `json:"taskColorSnapshot"`
	Facet             uint8      `json:"facet"`
	StartedAt         time.Time  `json:"startedAt"`
	EndedAt           *time.Time `json:"endedAt,omitempty"`
	DurationSeconds   uint32     `json:"durationSeconds"`
	Source            string     `json:"source"`
	StartEventNumber  uint32     `json:"startEventNumber"`
	EndEventNumber    uint32     `json:"endEventNumber"`
}

type TaskSessionFilter struct {
	DeviceID string     `json:"deviceID"`
	TaskID   string     `json:"taskID"`
	Facet    *uint8     `json:"facet,omitempty"`
	From     *time.Time `json:"from,omitempty"`
	To       *time.Time `json:"to,omitempty"`
}

type ManualAction struct {
	Kind        string            `json:"kind"`
	Description string            `json:"description"`
	Inputs      map[string]string `json:"inputs,omitempty"`
}

type StageResult struct {
	Stage        string        `json:"stage"`
	Completed    bool          `json:"completed"`
	Error        string        `json:"error,omitempty"`
	ManualAction *ManualAction `json:"manualAction,omitempty"`
}

type PairingWorkflow struct {
	DeviceID     string        `json:"deviceID"`
	CurrentStage string        `json:"currentStage"`
	Completed    bool          `json:"completed"`
	Stages       []StageResult `json:"stages"`
	ManualAction *ManualAction `json:"manualAction,omitempty"`
}

type UnpairingWorkflow struct {
	DeviceID     string        `json:"deviceID"`
	CurrentStage string        `json:"currentStage"`
	Completed    bool          `json:"completed"`
	Stages       []StageResult `json:"stages"`
	ManualAction *ManualAction `json:"manualAction,omitempty"`
}

type DiscoveredDevice struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	RSSI      int               `json:"rssi"`
	Supported bool              `json:"supported"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type DeviceSnapshot struct {
	Profile       DeviceProfile            `json:"profile"`
	State         DeviceState              `json:"state"`
	FacetConfigs  []FacetConfigurationView `json:"facetConfigs"`
	TapConfigured bool                     `json:"tapConfigured"`
}
