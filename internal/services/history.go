package services

import (
	"context"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/device"
	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	"github.com/mitchellrj/timeflip-desktop/internal/store"
)

type HistoryService struct {
	store    store.Store
	client   device.Client
	tracking *TrackingService
}

func NewHistoryService(store store.Store, client device.Client, tracking *TrackingService) *HistoryService {
	return &HistoryService{store: store, client: client, tracking: tracking}
}

func (s *HistoryService) ImportDeviceHistory(ctx context.Context, handle device.Handle) (int, error) {
	if s.client == nil || handle == nil {
		return 0, nil
	}
	events, err := s.client.ReadHistory(ctx, handle, device.HistoryRequest{All: true})
	if err != nil {
		return 0, err
	}
	return s.ReconcileEventsToSessions(ctx, handle.DeviceID(), events)
}

func (s *HistoryService) ReconcileEventsToSessions(ctx context.Context, deviceID string, events []domain.DeviceEventRecord) (int, error) {
	seen, err := s.existingEventKeys(ctx, deviceID)
	if err != nil {
		return 0, err
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].EventNumber == events[j].EventNumber {
			return events[i].OccurredAt.Before(events[j].OccurredAt)
		}
		return events[i].EventNumber < events[j].EventNumber
	})
	imported := 0
	for _, event := range events {
		if event.DeviceID == "" {
			event.DeviceID = deviceID
		}
		key := eventKey(event)
		if seen[key] {
			continue
		}
		if err := s.tracking.ApplyDeviceEvent(ctx, event); err != nil {
			return imported, domain.TrackingError{AppError: domain.NewAppError(domain.ErrHistoryReconciliationFailed, "Could not reconcile device history.", err.Error(), err)}
		}
		seen[key] = true
		imported++
	}
	return imported, nil
}

func (s *HistoryService) ListTaskSessions(ctx context.Context, filter domain.TaskSessionFilter) ([]domain.TaskSession, error) {
	return s.store.ListTaskSessions(ctx, filter)
}

func (s *HistoryService) BuildTimeReport(ctx context.Context, req domain.TimeReportRequest) (domain.TimeReport, error) {
	from, to, now, err := validateTimeReportRequest(req)
	if err != nil {
		return domain.TimeReport{}, err
	}
	sessions, err := s.store.ListTaskSessions(ctx, domain.TaskSessionFilter{From: &from, To: &to, Overlap: true})
	if err != nil {
		return domain.TimeReport{}, err
	}
	type aggregate struct {
		summary domain.TaskTimeSummary
	}
	byTask := map[string]aggregate{}
	var total uint32
	for _, session := range sessions {
		activeSeconds := activeSecondsForPeriod(session, from, to, now)
		if activeSeconds == 0 {
			continue
		}
		key := session.TaskID + "\x00" + session.TaskLabelSnapshot + "\x00" + session.TaskIconSnapshot + "\x00" + session.TaskColorSnapshot
		item := byTask[key]
		if item.summary.Label == "" {
			item.summary = domain.TaskTimeSummary{
				TaskID: session.TaskID,
				Label:  session.TaskLabelSnapshot,
				Icon:   session.TaskIconSnapshot,
				Color:  session.TaskColorSnapshot,
			}
			if item.summary.Label == "" {
				item.summary.Label = "Unnamed task"
			}
		}
		item.summary.ActiveSeconds += activeSeconds
		byTask[key] = item
		total += activeSeconds
	}
	report := domain.TimeReport{
		Period: domain.ReportingPeriod{
			From: from,
			To:   to,
		},
		TotalActiveSeconds: total,
	}
	if total == 0 {
		return report, nil
	}
	var rows []domain.TaskTimeSummary
	var otherSeconds uint32
	for _, item := range byTask {
		row := item.summary
		if float64(row.ActiveSeconds)/float64(total) < 0.015 {
			otherSeconds += row.ActiveSeconds
			continue
		}
		rows = append(rows, row)
	}
	if otherSeconds > 0 {
		rows = append(rows, domain.TaskTimeSummary{
			Label:         "Other",
			Icon:          "layers",
			Color:         "#d8dee9",
			ActiveSeconds: otherSeconds,
			Other:         true,
		})
	}
	for i := range rows {
		rows[i].Share = float64(rows[i].ActiveSeconds) / float64(total)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].ActiveSeconds == rows[j].ActiveSeconds {
			if rows[i].Other != rows[j].Other {
				return !rows[i].Other
			}
			return rows[i].Label < rows[j].Label
		}
		return rows[i].ActiveSeconds > rows[j].ActiveSeconds
	})
	report.Rows = rows
	return report, nil
}

func (s *HistoryService) ListTaskSessionPage(ctx context.Context, req domain.DetailedHistoryRequest) (domain.TaskSessionPage, error) {
	from, to, page, pageSize, err := validateDetailedHistoryRequest(req)
	if err != nil {
		return domain.TaskSessionPage{}, err
	}
	filter := domain.TaskSessionFilter{
		From:    &from,
		To:      &to,
		Overlap: true,
		Limit:   pageSize,
		Offset:  page * pageSize,
	}
	sessions, err := s.store.ListTaskSessions(ctx, filter)
	if err != nil {
		return domain.TaskSessionPage{}, err
	}
	total, err := s.store.CountTaskSessions(ctx, domain.TaskSessionFilter{From: &from, To: &to, Overlap: true})
	if err != nil {
		return domain.TaskSessionPage{}, err
	}
	return domain.TaskSessionPage{
		Sessions:    sessions,
		Page:        page,
		PageSize:    pageSize,
		TotalCount:  total,
		HasNext:     (page+1)*pageSize < total,
		HasPrevious: page > 0,
	}, nil
}

func (s *HistoryService) GetCurrentSession(ctx context.Context, deviceID string) (*domain.TaskSession, error) {
	session, err := s.store.GetOpenTaskSession(ctx, deviceID)
	if err != nil {
		if isStoreNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func validateTimeReportRequest(req domain.TimeReportRequest) (time.Time, time.Time, time.Time, error) {
	if req.From == nil || req.To == nil {
		return time.Time{}, time.Time{}, time.Time{}, validationError("Report period is required.", "report period missing start or end")
	}
	from := req.From.UTC()
	to := req.To.UTC()
	if from.IsZero() || to.IsZero() {
		return time.Time{}, time.Time{}, time.Time{}, validationError("Report period is required.", "report period contains zero time")
	}
	if !from.Before(to) {
		return time.Time{}, time.Time{}, time.Time{}, validationError("Report start must be before report end.", "report period is inverted or zero duration")
	}
	now := time.Now().UTC()
	if req.Now != nil && !req.Now.IsZero() {
		now = req.Now.UTC()
	}
	return from, to, now, nil
}

func validateDetailedHistoryRequest(req domain.DetailedHistoryRequest) (time.Time, time.Time, int, int, error) {
	if req.From == nil || req.To == nil {
		return time.Time{}, time.Time{}, 0, 0, validationError("History period is required.", "history period missing start or end")
	}
	from := req.From.UTC()
	to := req.To.UTC()
	if from.IsZero() || to.IsZero() {
		return time.Time{}, time.Time{}, 0, 0, validationError("History period is required.", "history period contains zero time")
	}
	if !from.Before(to) {
		return time.Time{}, time.Time{}, 0, 0, validationError("History start must be before history end.", "history period is inverted or zero duration")
	}
	if req.Page < 0 {
		return time.Time{}, time.Time{}, 0, 0, validationError("History page is invalid.", "history page is negative")
	}
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 20
	}
	if pageSize < 0 || pageSize > 200 {
		return time.Time{}, time.Time{}, 0, 0, validationError("History page size is invalid.", "history page size outside allowed range")
	}
	return from, to, req.Page, pageSize, nil
}

func activeSecondsForPeriod(session domain.TaskSession, from time.Time, to time.Time, now time.Time) uint32 {
	if session.StartedAt.IsZero() {
		return 0
	}
	sessionStart := session.StartedAt.UTC()
	sessionEnd := now.UTC()
	if session.EndedAt != nil {
		sessionEnd = session.EndedAt.UTC()
	}
	if !sessionEnd.After(sessionStart) {
		return 0
	}
	overlapStart := maxTime(sessionStart, from.UTC())
	overlapEnd := minTime(sessionEnd, to.UTC())
	if !overlapEnd.After(overlapStart) {
		return 0
	}
	overlapSeconds := overlapEnd.Sub(overlapStart).Seconds()
	fullSeconds := sessionEnd.Sub(sessionStart).Seconds()
	if fullSeconds <= 0 || overlapSeconds <= 0 {
		return 0
	}
	pausedSeconds := float64(session.PausedSeconds)
	if overlapStart.After(sessionStart) || overlapEnd.Before(sessionEnd) {
		pausedSeconds *= overlapSeconds / fullSeconds
	}
	if session.PauseStartedAt != nil && session.EndedAt == nil {
		pauseStart := maxTime(session.PauseStartedAt.UTC(), overlapStart)
		pauseEnd := minTime(now.UTC(), overlapEnd)
		if pauseEnd.After(pauseStart) {
			pausedSeconds += pauseEnd.Sub(pauseStart).Seconds()
		}
	}
	active := overlapSeconds - pausedSeconds
	if active <= 0 {
		return 0
	}
	if active > float64(^uint32(0)) {
		return ^uint32(0)
	}
	return uint32(math.Floor(active))
}

func maxTime(left time.Time, right time.Time) time.Time {
	if left.After(right) {
		return left
	}
	return right
}

func minTime(left time.Time, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}

func validationError(message string, diagnostic string) error {
	return domain.ValidationError{AppError: domain.NewAppError(domain.ErrValidation, message, diagnostic, nil)}
}

func (s *HistoryService) existingEventKeys(ctx context.Context, deviceID string) (map[string]bool, error) {
	events, err := s.store.ListDeviceEvents(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(events))
	for _, event := range events {
		out[eventKey(event)] = true
	}
	return out, nil
}

func eventKey(event domain.DeviceEventRecord) string {
	if event.EventNumber > 0 {
		return "number:" + strconv.FormatUint(uint64(event.EventNumber), 10)
	}
	return "event:" + event.Kind + ":" + strconv.Itoa(int(event.Facet)) + ":" + strconv.FormatBool(event.Pause) + ":" + event.OccurredAt.UTC().Format("2006-01-02T15:04:05.000000000Z07:00")
}
