package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	if path == "" {
		path = "timeflip-desktop.sqlite"
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, domain.PersistenceError{AppError: domain.NewAppError(domain.ErrStorage, "Could not open local database.", err.Error(), err)}
	}
	db.SetMaxOpenConns(1)
	return &SQLiteStore{db: db}, nil
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) Migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS app_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			database_path TEXT NOT NULL,
			communication_timeout_ns INTEGER NOT NULL,
			command_timeout_ns INTEGER NOT NULL,
			initial_retry_interval_ns INTEGER NOT NULL,
			medium_retry_interval_ns INTEGER NOT NULL,
			long_retry_interval_ns INTEGER NOT NULL,
			offline_after_duration_ns INTEGER NOT NULL,
			offline_after_failures INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS device_profiles (
			id TEXT PRIMARY KEY,
			display_name TEXT NOT NULL,
			advertised_name TEXT NOT NULL,
			protocol_version TEXT NOT NULL,
			stored_password TEXT NOT NULL,
			pairing_state TEXT NOT NULL,
			last_seen_at TEXT NOT NULL,
			last_connected_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			label TEXT NOT NULL,
			icon TEXT NOT NULL,
			color TEXT NOT NULL,
			archived INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS facet_assignments (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			facet INTEGER NOT NULL,
			task_id TEXT,
			task_label_snapshot TEXT NOT NULL,
			task_icon_snapshot TEXT NOT NULL,
			task_color_snapshot TEXT NOT NULL,
			is_pause_assignment INTEGER NOT NULL,
			pomodoro_limit_seconds INTEGER NOT NULL,
			effective_from TEXT NOT NULL,
			confirmed_on_device INTEGER NOT NULL,
			UNIQUE(device_id, facet)
		)`,
		`CREATE TABLE IF NOT EXISTS device_states (
			device_id TEXT PRIMARY KEY,
			connection_state TEXT NOT NULL,
			current_facet INTEGER NOT NULL,
			current_facet_known INTEGER NOT NULL,
			current_facet_undefined INTEGER NOT NULL,
			paused INTEGER NOT NULL,
			locked INTEGER NOT NULL,
			battery_percent INTEGER NOT NULL,
			system_status TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS device_events (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			facet INTEGER NOT NULL,
			pause INTEGER NOT NULL,
			event_number INTEGER NOT NULL,
			occurred_at TEXT NOT NULL,
			source TEXT NOT NULL,
			raw_summary TEXT NOT NULL,
			UNIQUE(device_id, event_number, kind, occurred_at)
		)`,
		`CREATE TABLE IF NOT EXISTS task_sessions (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			task_id TEXT NOT NULL,
			task_label_snapshot TEXT NOT NULL,
			task_icon_snapshot TEXT NOT NULL,
			task_color_snapshot TEXT NOT NULL,
			facet INTEGER NOT NULL,
			started_at TEXT NOT NULL,
			ended_at TEXT,
			duration_seconds INTEGER NOT NULL,
			source TEXT NOT NULL,
			start_event_number INTEGER NOT NULL,
			end_event_number INTEGER NOT NULL
		)`,
	}
	for i, stmt := range statements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return domain.PersistenceError{AppError: domain.NewAppError(domain.ErrStorage, "Could not migrate local database.", fmt.Sprintf("migration %d failed: %v", i, err), err)}
		}
	}
	return nil
}

func (s *SQLiteStore) SaveDeviceProfile(ctx context.Context, profile domain.DeviceProfile) error {
	if err := domain.ValidateDeviceProfile(profile); err != nil {
		return err
	}
	now := time.Now().UTC()
	if profile.LastSeenAt.IsZero() {
		profile.LastSeenAt = now
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO device_profiles
		(id, display_name, advertised_name, protocol_version, stored_password, pairing_state, last_seen_at, last_connected_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			display_name=excluded.display_name,
			advertised_name=excluded.advertised_name,
			protocol_version=excluded.protocol_version,
			stored_password=excluded.stored_password,
			pairing_state=excluded.pairing_state,
			last_seen_at=excluded.last_seen_at,
			last_connected_at=excluded.last_connected_at`,
		profile.ID, profile.DisplayName, profile.AdvertisedName, profile.ProtocolVersion, profile.StoredPassword, profile.PairingState,
		formatTime(profile.LastSeenAt), formatTime(profile.LastConnectedAt))
	return wrapStoreErr("Could not save device profile.", err)
}

func (s *SQLiteStore) GetDeviceProfile(ctx context.Context, id string) (domain.DeviceProfile, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, display_name, advertised_name, protocol_version, stored_password, pairing_state, last_seen_at, last_connected_at FROM device_profiles WHERE id = ?`, id)
	return scanDeviceProfile(row)
}

func (s *SQLiteStore) ListDeviceProfiles(ctx context.Context) ([]domain.DeviceProfile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, display_name, advertised_name, protocol_version, stored_password, pairing_state, last_seen_at, last_connected_at FROM device_profiles ORDER BY last_seen_at DESC, display_name ASC`)
	if err != nil {
		return nil, wrapStoreErr("Could not list device profiles.", err)
	}
	defer rows.Close()
	var out []domain.DeviceProfile
	for rows.Next() {
		p, err := scanDeviceProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, wrapStoreErr("Could not list device profiles.", rows.Err())
}

func (s *SQLiteStore) SaveTask(ctx context.Context, task domain.Task) error {
	if task.ID == "" {
		task.ID = domain.NewID("task")
	}
	now := time.Now().UTC()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	if err := domain.ValidateTask(task); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO tasks (id, label, icon, color, archived, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET label=excluded.label, icon=excluded.icon, color=excluded.color, archived=excluded.archived, updated_at=excluded.updated_at`,
		task.ID, task.Label, task.Icon, task.Color, boolInt(task.Archived), formatTime(task.CreatedAt), formatTime(task.UpdatedAt))
	return wrapStoreErr("Could not save task.", err)
}

func (s *SQLiteStore) ListTasks(ctx context.Context, includeArchived bool) ([]domain.Task, error) {
	query := `SELECT id, label, icon, color, archived, created_at, updated_at FROM tasks`
	if !includeArchived {
		query += ` WHERE archived = 0`
	}
	query += ` ORDER BY archived ASC, label ASC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, wrapStoreErr("Could not list tasks.", err)
	}
	defer rows.Close()
	var out []domain.Task
	for rows.Next() {
		var t domain.Task
		var archived int
		var created, updated string
		if err := rows.Scan(&t.ID, &t.Label, &t.Icon, &t.Color, &archived, &created, &updated); err != nil {
			return nil, wrapStoreErr("Could not list tasks.", err)
		}
		t.Archived = archived != 0
		t.CreatedAt = parseTime(created)
		t.UpdatedAt = parseTime(updated)
		out = append(out, t)
	}
	return out, wrapStoreErr("Could not list tasks.", rows.Err())
}

func (s *SQLiteStore) ArchiveTask(ctx context.Context, taskID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET archived = 1, updated_at = ? WHERE id = ?`, formatTime(time.Now().UTC()), taskID)
	return wrapStoreErr("Could not archive task.", err)
}

func (s *SQLiteStore) SaveFacetAssignment(ctx context.Context, a domain.FacetAssignment) error {
	if a.ID == "" {
		a.ID = domain.NewID("assignment")
	}
	if a.EffectiveFrom.IsZero() {
		a.EffectiveFrom = time.Now().UTC()
	}
	if err := domain.ValidateFacetAssignment(a); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO facet_assignments
		(id, device_id, facet, task_id, task_label_snapshot, task_icon_snapshot, task_color_snapshot, is_pause_assignment, pomodoro_limit_seconds, effective_from, confirmed_on_device)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(device_id, facet) DO UPDATE SET
			id=excluded.id,
			task_id=excluded.task_id,
			task_label_snapshot=excluded.task_label_snapshot,
			task_icon_snapshot=excluded.task_icon_snapshot,
			task_color_snapshot=excluded.task_color_snapshot,
			is_pause_assignment=excluded.is_pause_assignment,
			pomodoro_limit_seconds=excluded.pomodoro_limit_seconds,
			effective_from=excluded.effective_from,
			confirmed_on_device=excluded.confirmed_on_device`,
		a.ID, a.DeviceID, a.Facet, a.TaskID, a.TaskLabelSnapshot, a.TaskIconSnapshot, a.TaskColorSnapshot,
		boolInt(a.IsPauseAssignment), a.PomodoroLimitSeconds, formatTime(a.EffectiveFrom), boolInt(a.ConfirmedOnDevice))
	return wrapStoreErr("Could not save facet assignment.", err)
}

func (s *SQLiteStore) ListFacetAssignments(ctx context.Context, deviceID string) ([]domain.FacetAssignment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, device_id, facet, task_id, task_label_snapshot, task_icon_snapshot, task_color_snapshot, is_pause_assignment, pomodoro_limit_seconds, effective_from, confirmed_on_device FROM facet_assignments WHERE device_id = ? ORDER BY facet ASC`, deviceID)
	if err != nil {
		return nil, wrapStoreErr("Could not list facet assignments.", err)
	}
	defer rows.Close()
	var out []domain.FacetAssignment
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, wrapStoreErr("Could not list facet assignments.", rows.Err())
}

func (s *SQLiteStore) GetFacetAssignment(ctx context.Context, deviceID string, facet uint8) (domain.FacetAssignment, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, device_id, facet, task_id, task_label_snapshot, task_icon_snapshot, task_color_snapshot, is_pause_assignment, pomodoro_limit_seconds, effective_from, confirmed_on_device FROM facet_assignments WHERE device_id = ? AND facet = ?`, deviceID, facet)
	return scanAssignment(row)
}

func (s *SQLiteStore) SaveDeviceState(ctx context.Context, state domain.DeviceState) error {
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO device_states
		(device_id, connection_state, current_facet, current_facet_known, current_facet_undefined, paused, locked, battery_percent, system_status, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(device_id) DO UPDATE SET
			connection_state=excluded.connection_state,
			current_facet=excluded.current_facet,
			current_facet_known=excluded.current_facet_known,
			current_facet_undefined=excluded.current_facet_undefined,
			paused=excluded.paused,
			locked=excluded.locked,
			battery_percent=excluded.battery_percent,
			system_status=excluded.system_status,
			updated_at=excluded.updated_at`,
		state.DeviceID, string(state.ConnectionState), state.CurrentFacet, boolInt(state.CurrentFacetKnown), boolInt(state.CurrentFacetUndefined),
		boolInt(state.Paused), boolInt(state.Locked), state.BatteryPercent, state.SystemStatus, formatTime(state.UpdatedAt))
	return wrapStoreErr("Could not save device state.", err)
}

func (s *SQLiteStore) GetDeviceState(ctx context.Context, deviceID string) (domain.DeviceState, error) {
	row := s.db.QueryRowContext(ctx, `SELECT device_id, connection_state, current_facet, current_facet_known, current_facet_undefined, paused, locked, battery_percent, system_status, updated_at FROM device_states WHERE device_id = ?`, deviceID)
	var state domain.DeviceState
	var connection string
	var known, undefined, paused, locked int
	var updated string
	err := row.Scan(&state.DeviceID, &connection, &state.CurrentFacet, &known, &undefined, &paused, &locked, &state.BatteryPercent, &state.SystemStatus, &updated)
	if err != nil {
		return domain.DeviceState{}, wrapScanNotFound("Could not load device state.", err)
	}
	state.ConnectionState = domain.ConnectionState(connection)
	state.CurrentFacetKnown = known != 0
	state.CurrentFacetUndefined = undefined != 0
	state.Paused = paused != 0
	state.Locked = locked != 0
	state.UpdatedAt = parseTime(updated)
	return state, nil
}

func (s *SQLiteStore) InsertDeviceEvent(ctx context.Context, event domain.DeviceEventRecord) error {
	if event.ID == "" {
		event.ID = domain.NewID("event")
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO device_events
		(id, device_id, kind, facet, pause, event_number, occurred_at, source, raw_summary)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.DeviceID, event.Kind, event.Facet, boolInt(event.Pause), event.EventNumber, formatTime(event.OccurredAt), event.Source, event.RawSummary)
	return wrapStoreErr("Could not save device event.", err)
}

func (s *SQLiteStore) ListDeviceEvents(ctx context.Context, deviceID string) ([]domain.DeviceEventRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, device_id, kind, facet, pause, event_number, occurred_at, source, raw_summary FROM device_events WHERE device_id = ? ORDER BY event_number ASC, occurred_at ASC`, deviceID)
	if err != nil {
		return nil, wrapStoreErr("Could not list device events.", err)
	}
	defer rows.Close()
	var out []domain.DeviceEventRecord
	for rows.Next() {
		var e domain.DeviceEventRecord
		var pause int
		var occurred string
		if err := rows.Scan(&e.ID, &e.DeviceID, &e.Kind, &e.Facet, &pause, &e.EventNumber, &occurred, &e.Source, &e.RawSummary); err != nil {
			return nil, wrapStoreErr("Could not list device events.", err)
		}
		e.Pause = pause != 0
		e.OccurredAt = parseTime(occurred)
		out = append(out, e)
	}
	return out, wrapStoreErr("Could not list device events.", rows.Err())
}

func (s *SQLiteStore) SaveTaskSession(ctx context.Context, session domain.TaskSession) error {
	if session.ID == "" {
		session.ID = domain.NewID("session")
	}
	var ended any
	if session.EndedAt != nil {
		ended = formatTime(*session.EndedAt)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO task_sessions
		(id, device_id, task_id, task_label_snapshot, task_icon_snapshot, task_color_snapshot, facet, started_at, ended_at, duration_seconds, source, start_event_number, end_event_number)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			ended_at=excluded.ended_at,
			duration_seconds=excluded.duration_seconds,
			end_event_number=excluded.end_event_number`,
		session.ID, session.DeviceID, session.TaskID, session.TaskLabelSnapshot, session.TaskIconSnapshot, session.TaskColorSnapshot, session.Facet,
		formatTime(session.StartedAt), ended, session.DurationSeconds, session.Source, session.StartEventNumber, session.EndEventNumber)
	return wrapStoreErr("Could not save task session.", err)
}

func (s *SQLiteStore) ListTaskSessions(ctx context.Context, filter domain.TaskSessionFilter) ([]domain.TaskSession, error) {
	query := `SELECT id, device_id, task_id, task_label_snapshot, task_icon_snapshot, task_color_snapshot, facet, started_at, ended_at, duration_seconds, source, start_event_number, end_event_number FROM task_sessions WHERE 1=1`
	var args []any
	if filter.DeviceID != "" {
		query += ` AND device_id = ?`
		args = append(args, filter.DeviceID)
	}
	if filter.TaskID != "" {
		query += ` AND task_id = ?`
		args = append(args, filter.TaskID)
	}
	if filter.Facet != nil {
		query += ` AND facet = ?`
		args = append(args, *filter.Facet)
	}
	if filter.From != nil {
		query += ` AND started_at >= ?`
		args = append(args, formatTime(*filter.From))
	}
	if filter.To != nil {
		query += ` AND started_at <= ?`
		args = append(args, formatTime(*filter.To))
	}
	query += ` ORDER BY started_at DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, wrapStoreErr("Could not list task sessions.", err)
	}
	defer rows.Close()
	var out []domain.TaskSession
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, session)
	}
	return out, wrapStoreErr("Could not list task sessions.", rows.Err())
}

func (s *SQLiteStore) GetOpenTaskSession(ctx context.Context, deviceID string) (domain.TaskSession, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, device_id, task_id, task_label_snapshot, task_icon_snapshot, task_color_snapshot, facet, started_at, ended_at, duration_seconds, source, start_event_number, end_event_number FROM task_sessions WHERE device_id = ? AND (ended_at IS NULL OR ended_at = '') ORDER BY started_at DESC LIMIT 1`, deviceID)
	return scanSession(row)
}

func (s *SQLiteStore) SaveConfig(ctx context.Context, config domain.AppConfig) error {
	if config.ReconnectPolicy.OfflineAfterFailures == 0 {
		config.ReconnectPolicy = domain.DefaultReconnectPolicy()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO app_config
		(id, database_path, communication_timeout_ns, command_timeout_ns, initial_retry_interval_ns, medium_retry_interval_ns, long_retry_interval_ns, offline_after_duration_ns, offline_after_failures)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			database_path=excluded.database_path,
			communication_timeout_ns=excluded.communication_timeout_ns,
			command_timeout_ns=excluded.command_timeout_ns,
			initial_retry_interval_ns=excluded.initial_retry_interval_ns,
			medium_retry_interval_ns=excluded.medium_retry_interval_ns,
			long_retry_interval_ns=excluded.long_retry_interval_ns,
			offline_after_duration_ns=excluded.offline_after_duration_ns,
			offline_after_failures=excluded.offline_after_failures`,
		config.DatabasePath, int64(config.CommunicationTimeout), int64(config.CommandTimeout), int64(config.ReconnectPolicy.InitialRetryInterval),
		int64(config.ReconnectPolicy.MediumRetryInterval), int64(config.ReconnectPolicy.LongRetryInterval), int64(config.ReconnectPolicy.OfflineAfterDuration),
		config.ReconnectPolicy.OfflineAfterFailures)
	return wrapStoreErr("Could not save app settings.", err)
}

func (s *SQLiteStore) LoadConfig(ctx context.Context) (domain.AppConfig, error) {
	row := s.db.QueryRowContext(ctx, `SELECT database_path, communication_timeout_ns, command_timeout_ns, initial_retry_interval_ns, medium_retry_interval_ns, long_retry_interval_ns, offline_after_duration_ns, offline_after_failures FROM app_config WHERE id = 1`)
	cfg := domain.DefaultAppConfig()
	var communication, command, initial, medium, long, offline int64
	err := row.Scan(&cfg.DatabasePath, &communication, &command, &initial, &medium, &long, &offline, &cfg.ReconnectPolicy.OfflineAfterFailures)
	if errors.Is(err, sql.ErrNoRows) {
		return cfg, nil
	}
	if err != nil {
		return domain.AppConfig{}, wrapStoreErr("Could not load app settings.", err)
	}
	cfg.CommunicationTimeout = time.Duration(communication)
	cfg.CommandTimeout = time.Duration(command)
	cfg.ReconnectPolicy.InitialRetryInterval = time.Duration(initial)
	cfg.ReconnectPolicy.MediumRetryInterval = time.Duration(medium)
	cfg.ReconnectPolicy.LongRetryInterval = time.Duration(long)
	cfg.ReconnectPolicy.OfflineAfterDuration = time.Duration(offline)
	return cfg, nil
}

type scanner interface {
	Scan(...any) error
}

func scanDeviceProfile(row scanner) (domain.DeviceProfile, error) {
	var p domain.DeviceProfile
	var seen, connected string
	err := row.Scan(&p.ID, &p.DisplayName, &p.AdvertisedName, &p.ProtocolVersion, &p.StoredPassword, &p.PairingState, &seen, &connected)
	if err != nil {
		return domain.DeviceProfile{}, wrapScanNotFound("Could not load device profile.", err)
	}
	p.LastSeenAt = parseTime(seen)
	p.LastConnectedAt = parseTime(connected)
	return p, nil
}

func scanAssignment(row scanner) (domain.FacetAssignment, error) {
	var a domain.FacetAssignment
	var pause, confirmed int
	var effective string
	err := row.Scan(&a.ID, &a.DeviceID, &a.Facet, &a.TaskID, &a.TaskLabelSnapshot, &a.TaskIconSnapshot, &a.TaskColorSnapshot, &pause, &a.PomodoroLimitSeconds, &effective, &confirmed)
	if err != nil {
		return domain.FacetAssignment{}, wrapScanNotFound("Could not load facet assignment.", err)
	}
	a.IsPauseAssignment = pause != 0
	a.ConfirmedOnDevice = confirmed != 0
	a.EffectiveFrom = parseTime(effective)
	return a, nil
}

func scanSession(row scanner) (domain.TaskSession, error) {
	var session domain.TaskSession
	var started string
	var ended sql.NullString
	err := row.Scan(&session.ID, &session.DeviceID, &session.TaskID, &session.TaskLabelSnapshot, &session.TaskIconSnapshot, &session.TaskColorSnapshot, &session.Facet, &started, &ended, &session.DurationSeconds, &session.Source, &session.StartEventNumber, &session.EndEventNumber)
	if err != nil {
		return domain.TaskSession{}, wrapScanNotFound("Could not load task session.", err)
	}
	session.StartedAt = parseTime(started)
	if ended.Valid && ended.String != "" {
		t := parseTime(ended.String)
		session.EndedAt = &t
	}
	return session, nil
}

func wrapScanNotFound(message string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PersistenceError{AppError: domain.NewAppError(domain.ErrDeviceNotFound, message, domain.ErrNotFound.Error(), domain.ErrNotFound)}
	}
	return wrapStoreErr(message, err)
}

func wrapStoreErr(message string, err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return nil
	}
	return domain.PersistenceError{AppError: domain.NewAppError(domain.ErrStorage, message, err.Error(), err)}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339Nano, value)
	return t.UTC()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
