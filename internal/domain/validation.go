package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var colorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func NewID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

func ValidateDeviceProfile(profile DeviceProfile) error {
	if strings.TrimSpace(profile.ID) == "" {
		return ValidationError{NewAppError(ErrValidation, "Device ID is required.", "device id is empty", nil)}
	}
	if profile.StoredPassword != "" && len(profile.StoredPassword) != 6 {
		return ValidationError{NewAppError(ErrValidation, "TimeFlip password must be six characters.", "invalid password length", nil)}
	}
	return nil
}

func ValidateDeviceName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ValidationError{NewAppError(ErrValidation, "Device name is required.", "device name is empty", nil)}
	}
	if len(name) > 18 {
		return ValidationError{NewAppError(ErrValidation, "Device name must be 18 ASCII characters or fewer.", "device name exceeds TimeFlip2 limit", nil)}
	}
	for i := 0; i < len(name); i++ {
		if name[i] < 0x20 || name[i] > 0x7e {
			return ValidationError{NewAppError(ErrValidation, "Device name must use printable ASCII characters.", "device name contains non-ascii or control characters", nil)}
		}
	}
	return nil
}

func ValidateTask(task Task) error {
	if strings.TrimSpace(task.Label) == "" {
		return ValidationError{NewAppError(ErrValidation, "Task label is required.", "task label is empty", nil)}
	}
	if len(task.Icon) > 64 {
		return ValidationError{NewAppError(ErrValidation, "Task icon is too long.", "icon length exceeds 64", nil)}
	}
	if task.Color != "" && !colorPattern.MatchString(task.Color) {
		return ValidationError{NewAppError(ErrValidation, "Task colour must be a hex value.", "invalid task color", nil)}
	}
	return nil
}

func ValidateFacetAssignment(assignment FacetAssignment) error {
	if strings.TrimSpace(assignment.DeviceID) == "" {
		return ValidationError{NewAppError(ErrValidation, "Device ID is required.", "assignment device id is empty", nil)}
	}
	if assignment.Facet < 1 || assignment.Facet > FacetCount {
		return ValidationError{NewAppError(ErrValidation, "Facet must be between 1 and 12.", "facet outside TimeFlip2 range", nil)}
	}
	if assignment.IsPauseAssignment && assignment.TaskID != "" {
		return ValidationError{NewAppError(ErrValidation, "A pause side cannot also be a task.", "pause assignment has task id", nil)}
	}
	if assignment.IsPauseAssignment && assignment.IsPomodoroAssignment {
		return ValidationError{NewAppError(ErrValidation, "A pause side cannot also be a pomodoro side.", "pause assignment marked pomodoro", nil)}
	}
	if !assignment.IsPauseAssignment && strings.TrimSpace(assignment.TaskID) == "" {
		return ValidationError{NewAppError(ErrValidation, "Task assignment is required.", "non-pause assignment has no task id", nil)}
	}
	if assignment.IsPomodoroAssignment && assignment.PomodoroLimitSeconds == 0 {
		return ValidationError{NewAppError(ErrValidation, "Pomodoro duration is required.", "pomodoro assignment has no duration", nil)}
	}
	if !assignment.IsPomodoroAssignment && assignment.PomodoroLimitSeconds != 0 {
		return ValidationError{NewAppError(ErrValidation, "Pomodoro duration only applies to pomodoro sides.", "normal assignment has pomodoro duration", nil)}
	}
	if assignment.PomodoroLimitSeconds > 24*60*60 {
		return ValidationError{NewAppError(ErrValidation, "Pomodoro duration is too long.", "pomodoro exceeds 24 hours", nil)}
	}
	return nil
}

func ValidateDeviceTapSettings(settings DeviceTapSettings) error {
	if strings.TrimSpace(settings.DeviceID) == "" {
		return ValidationError{NewAppError(ErrValidation, "Device ID is required.", "tap settings device id is empty", nil)}
	}
	for label, value := range map[string]uint8{
		"threshold": settings.Threshold,
		"limit":     settings.Limit,
		"latency":   settings.Latency,
		"window":    settings.Window,
	} {
		if value > 255 {
			return ValidationError{NewAppError(ErrValidation, "Tap settings must be byte register values.", label+" outside TimeFlip2 byte range", nil)}
		}
	}
	return nil
}

func ValidateDeviceLEDSettings(settings DeviceLEDSettings) error {
	if strings.TrimSpace(settings.DeviceID) == "" {
		return ValidationError{NewAppError(ErrValidation, "Device ID is required.", "LED settings device id is empty", nil)}
	}
	if settings.BrightnessPercent < 1 || settings.BrightnessPercent > 100 {
		return ValidationError{NewAppError(ErrValidation, "LED brightness must be between 1% and 100%.", "led brightness outside TimeFlip2 range", nil)}
	}
	if settings.BlinkSeconds < 5 || settings.BlinkSeconds > 60 {
		return ValidationError{NewAppError(ErrValidation, "LED blink must be between 5 and 60 seconds.", "led blink outside TimeFlip2 range", nil)}
	}
	return nil
}

func StartTaskSession(deviceID string, assignment FacetAssignment, event DeviceEventRecord) (TaskSession, bool, error) {
	if assignment.IsPauseAssignment {
		return TaskSession{}, false, nil
	}
	if err := ValidateFacetAssignment(assignment); err != nil {
		return TaskSession{}, false, err
	}
	started := event.OccurredAt
	if started.IsZero() {
		started = time.Now().UTC()
	}
	return TaskSession{
		ID:                NewID("session"),
		DeviceID:          deviceID,
		TaskID:            assignment.TaskID,
		TaskLabelSnapshot: assignment.TaskLabelSnapshot,
		TaskIconSnapshot:  assignment.TaskIconSnapshot,
		TaskColorSnapshot: assignment.TaskColorSnapshot,
		Facet:             assignment.Facet,
		StartedAt:         started.UTC(),
		Source:            event.Source,
		StartEventNumber:  event.EventNumber,
	}, true, nil
}

func EndTaskSession(session TaskSession, endedAt time.Time, eventNumber uint32) (TaskSession, bool, error) {
	if endedAt.IsZero() {
		endedAt = time.Now().UTC()
	}
	endedAt = endedAt.UTC()
	if endedAt.Before(session.StartedAt) {
		return TaskSession{}, false, TrackingError{NewAppError(ErrValidation, "Task session cannot end before it starts.", "negative session duration", nil)}
	}
	if session.PauseStartedAt != nil {
		pauseStarted := session.PauseStartedAt.UTC()
		if endedAt.After(pauseStarted) {
			session.PausedSeconds += uint32(endedAt.Sub(pauseStarted).Seconds())
		}
		session.PauseStartedAt = nil
	}
	duration := endedAt.Sub(session.StartedAt)
	if duration <= 0 {
		return session, false, nil
	}
	session.EndedAt = &endedAt
	session.DurationSeconds = uint32(duration.Seconds())
	session.EndEventNumber = eventNumber
	return session, true, nil
}
