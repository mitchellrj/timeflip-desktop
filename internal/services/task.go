package services

import (
	"context"
	"errors"
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
		PomodoroLimitSeconds: req.PomodoroLimitSeconds,
		EffectiveFrom:        s.clock.Now(),
	}
	if req.IsPauseAssignment {
		assignment.TaskLabelSnapshot = "Paused"
		assignment.TaskIconSnapshot = "pause"
		assignment.TaskColorSnapshot = "#64748B"
	} else {
		task := domain.Task{ID: req.TaskID, Label: req.Label, Icon: req.Icon, Color: req.Color, CreatedAt: time.Now().UTC()}
		if req.TaskID == "" {
			created, err := s.CreateTask(ctx, req.Label, req.Icon, req.Color)
			if err != nil {
				return domain.FacetAssignment{}, err
			}
			task = created
			assignment.TaskID = created.ID
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
		view := domain.FacetConfigurationView{Facet: facet}
		if assignment, ok := byFacet[facet]; ok {
			view.TaskID = assignment.TaskID
			view.Label = assignment.TaskLabelSnapshot
			view.Icon = assignment.TaskIconSnapshot
			view.Color = assignment.TaskColorSnapshot
			view.IsPauseAssignment = assignment.IsPauseAssignment
			view.PomodoroLimitSeconds = assignment.PomodoroLimitSeconds
			view.AssignedOnDevice = assignment.ConfirmedOnDevice
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *TaskService) SetPomodoroForFacet(ctx context.Context, deviceID string, facet uint8, seconds uint32) (domain.FacetAssignment, error) {
	assignment, err := s.store.GetFacetAssignment(ctx, deviceID, facet)
	if err != nil {
		return domain.FacetAssignment{}, err
	}
	assignment.PomodoroLimitSeconds = seconds
	if err := domain.ValidateFacetAssignment(assignment); err != nil {
		return domain.FacetAssignment{}, err
	}
	return assignment, s.store.SaveFacetAssignment(ctx, assignment)
}

func IsNotFound(err error) bool {
	return errors.Is(err, domain.ErrNotFound)
}
