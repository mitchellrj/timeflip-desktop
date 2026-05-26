package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/mitchellrj/timeflip-desktop/internal/store"
)

type TaskService struct {
	store store.Store
	clock Clock
}

func NewTaskService(store store.Store, clock Clock) *TaskService {
	if clock == nil {
		clock = SystemClock{}
	}
	return &TaskService{store: store, clock: clock}
}

func (s *TaskService) CreateTask(ctx context.Context, label string, icon string, color string) (domain.Task, error) {
	if existing, ok, err := s.findTaskByLabel(ctx, label, ""); err != nil {
		return domain.Task{}, err
	} else if ok {
		return domain.Task{}, duplicateTaskLabelError(existing.Label)
	}
	task := domain.Task{
		ID:        domain.NewID("task"),
		Label:     label,
		Icon:      icon,
		Color:     color,
		CreatedAt: s.clock.Now(),
		UpdatedAt: s.clock.Now(),
	}
	if err := domain.ValidateTask(task); err != nil {
		return domain.Task{}, err
	}
	return task, s.store.SaveTask(ctx, task)
}

func (s *TaskService) UpdateTask(ctx context.Context, task domain.Task) (domain.Task, error) {
	if existing, ok, err := s.findTaskByLabel(ctx, task.Label, task.ID); err != nil {
		return domain.Task{}, err
	} else if ok {
		return domain.Task{}, duplicateTaskLabelError(existing.Label)
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = s.clock.Now()
	}
	if err := domain.ValidateTask(task); err != nil {
		return domain.Task{}, err
	}
	return task, s.store.SaveTask(ctx, task)
}

func (s *TaskService) ArchiveTask(ctx context.Context, taskID string) error {
	return s.store.ArchiveTask(ctx, taskID)
}

func (s *TaskService) AssignFacet(ctx context.Context, req domain.FacetConfigurationRequest) (domain.FacetAssignment, error) {
	assignment := domain.FacetAssignment{
		ID:                   domain.NewID("assignment"),
		DeviceID:             req.DeviceID,
		Facet:                req.Facet,
		TaskID:               req.TaskID,
		IsPauseAssignment:    req.IsPauseAssignment,
		IsPomodoroAssignment: req.IsPomodoroAssignment,
		PomodoroLimitSeconds: req.PomodoroLimitSeconds,
		EffectiveFrom:        s.clock.Now(),
	}
	if req.IsPauseAssignment {
		assignment.IsPomodoroAssignment = false
		assignment.PomodoroLimitSeconds = 0
		assignment.TaskLabelSnapshot = "Paused"
		assignment.TaskIconSnapshot = "pause"
		assignment.TaskColorSnapshot = "#64748B"
	} else {
		if !req.IsPomodoroAssignment {
			assignment.PomodoroLimitSeconds = 0
		}
		task := domain.Task{ID: req.TaskID, Label: req.Label, Icon: req.Icon, Color: req.Color, CreatedAt: time.Now().UTC()}
		if req.TaskID == "" {
			if existing, ok, err := s.findTaskByLabel(ctx, req.Label, ""); err != nil {
				return domain.FacetAssignment{}, err
			} else if ok {
				task = existing
				assignment.TaskID = existing.ID
			} else {
				created, err := s.CreateTask(ctx, req.Label, req.Icon, req.Color)
				if err != nil {
					return domain.FacetAssignment{}, err
				}
				task = created
				assignment.TaskID = created.ID
			}
		}
		assignment.TaskLabelSnapshot = task.Label
		assignment.TaskIconSnapshot = task.Icon
		assignment.TaskColorSnapshot = task.Color
	}
	if err := domain.ValidateFacetAssignment(assignment); err != nil {
		return domain.FacetAssignment{}, err
	}
	if err := s.store.SaveFacetAssignment(ctx, assignment); err != nil {
		return domain.FacetAssignment{}, err
	}
	return assignment, nil
}

func (s *TaskService) ListFacetConfiguration(ctx context.Context, deviceID string) ([]domain.FacetConfigurationView, error) {
	assignments, err := s.store.ListFacetAssignments(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	byFacet := map[uint8]domain.FacetAssignment{}
	for _, assignment := range assignments {
		byFacet[assignment.Facet] = assignment
	}
	views := make([]domain.FacetConfigurationView, 0, domain.FacetCount)
	for facet := uint8(1); facet <= domain.FacetCount; facet++ {
		view := domain.FacetConfigurationView{DeviceID: deviceID, Facet: facet}
		if assignment, ok := byFacet[facet]; ok {
			view.TaskID = assignment.TaskID
			view.Label = assignment.TaskLabelSnapshot
			view.Icon = assignment.TaskIconSnapshot
			view.Color = assignment.TaskColorSnapshot
			view.IsPauseAssignment = assignment.IsPauseAssignment
			view.IsPomodoroAssignment = assignment.IsPomodoroAssignment
			view.PomodoroLimitSeconds = assignment.PomodoroLimitSeconds
			view.AssignedOnDevice = assignment.ConfirmedOnDevice
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *TaskService) ResetFacetConfiguration(ctx context.Context, deviceID string) ([]domain.FacetConfigurationView, error) {
	if strings.TrimSpace(deviceID) == "" {
		return nil, domain.ValidationError{AppError: domain.NewAppError(domain.ErrValidation, "Device ID is required.", "reset facets device id is empty", nil)}
	}
	if err := s.store.DeleteFacetAssignments(ctx, deviceID); err != nil {
		return nil, err
	}
	return s.ListFacetConfiguration(ctx, deviceID)
}

func (s *TaskService) ClearFacetConfiguration(ctx context.Context, deviceID string, facet uint8) (domain.FacetConfigurationView, error) {
	if strings.TrimSpace(deviceID) == "" {
		return domain.FacetConfigurationView{}, domain.ValidationError{AppError: domain.NewAppError(domain.ErrValidation, "Device ID is required.", "clear facet device id is empty", nil)}
	}
	if facet < 1 || facet > domain.FacetCount {
		return domain.FacetConfigurationView{}, domain.ValidationError{AppError: domain.NewAppError(domain.ErrValidation, "Facet must be between 1 and 12.", "facet out of range", nil)}
	}
	if err := s.store.DeleteFacetAssignment(ctx, deviceID, facet); err != nil {
		return domain.FacetConfigurationView{}, err
	}
	return domain.FacetConfigurationView{DeviceID: deviceID, Facet: facet}, nil
}

func (s *TaskService) SetPomodoroForFacet(ctx context.Context, deviceID string, facet uint8, seconds uint32) (domain.FacetAssignment, error) {
	assignment, err := s.store.GetFacetAssignment(ctx, deviceID, facet)
	if err != nil {
		return domain.FacetAssignment{}, err
	}
	assignment.PomodoroLimitSeconds = seconds
	assignment.IsPomodoroAssignment = seconds > 0
	if err := domain.ValidateFacetAssignment(assignment); err != nil {
		return domain.FacetAssignment{}, err
	}
	return assignment, s.store.SaveFacetAssignment(ctx, assignment)
}

func (s *TaskService) findTaskByLabel(ctx context.Context, label string, excludeID string) (domain.Task, bool, error) {
	needle := normaliseTaskLabel(label)
	if needle == "" {
		return domain.Task{}, false, nil
	}
	tasks, err := s.store.ListTasks(ctx, false)
	if err != nil {
		return domain.Task{}, false, err
	}
	for _, task := range tasks {
		if task.ID == excludeID {
			continue
		}
		if normaliseTaskLabel(task.Label) == needle {
			return task, true, nil
		}
	}
	return domain.Task{}, false, nil
}

func duplicateTaskLabelError(label string) error {
	return domain.ValidationError{AppError: domain.NewAppError(domain.ErrValidation, "A task with this label already exists.", "duplicate task label: "+label, nil)}
}

func normaliseTaskLabel(label string) string {
	return strings.ToLower(strings.Join(strings.Fields(label), " "))
}

func IsNotFound(err error) bool {
	return errors.Is(err, domain.ErrNotFound)
}
