package store

import (
	"context"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
)

type Store interface {
	Migrate(context.Context) error
	Close() error

	SaveDeviceProfile(context.Context, domain.DeviceProfile) error
	GetDeviceProfile(context.Context, string) (domain.DeviceProfile, error)
	ListDeviceProfiles(context.Context) ([]domain.DeviceProfile, error)

	SaveTask(context.Context, domain.Task) error
	ListTasks(context.Context, bool) ([]domain.Task, error)
	ArchiveTask(context.Context, string) error

	SaveFacetAssignment(context.Context, domain.FacetAssignment) error
	ListFacetAssignments(context.Context, string) ([]domain.FacetAssignment, error)
	GetFacetAssignment(context.Context, string, uint8) (domain.FacetAssignment, error)
	DeleteFacetAssignment(context.Context, string, uint8) error
	DeleteFacetAssignments(context.Context, string) error

	SaveDeviceState(context.Context, domain.DeviceState) error
	GetDeviceState(context.Context, string) (domain.DeviceState, error)
	SaveDeviceTapSettings(context.Context, domain.DeviceTapSettings) error
	GetDeviceTapSettings(context.Context, string) (domain.DeviceTapSettings, error)
	ListDeviceTapSettings(context.Context) ([]domain.DeviceTapSettings, error)
	SaveDeviceLEDSettings(context.Context, domain.DeviceLEDSettings) error
	GetDeviceLEDSettings(context.Context, string) (domain.DeviceLEDSettings, error)
	ListDeviceLEDSettings(context.Context) ([]domain.DeviceLEDSettings, error)

	InsertDeviceEvent(context.Context, domain.DeviceEventRecord) error
	ListDeviceEvents(context.Context, string) ([]domain.DeviceEventRecord, error)

	SaveTaskSession(context.Context, domain.TaskSession) error
	ListTaskSessions(context.Context, domain.TaskSessionFilter) ([]domain.TaskSession, error)
	CountTaskSessions(context.Context, domain.TaskSessionFilter) (int, error)
	GetOpenTaskSession(context.Context, string) (domain.TaskSession, error)

	SaveConfig(context.Context, domain.AppConfig) error
	LoadConfig(context.Context) (domain.AppConfig, error)
}
