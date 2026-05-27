package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mitchellrj/timeflip-desktop/internal/domain"
	_ "modernc.org/sqlite"
)

func TestListTaskSessionsOverlapFilterIncludesBoundaryCrossingSessions(t *testing.T) {
	st, closeStore := newTestSQLiteStore(t)
	defer closeStore()
	ctx := context.Background()
	base := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	for _, session := range []domain.TaskSession{
		testSession("inside", base.Add(15*time.Minute), base.Add(30*time.Minute)),
		testSession("before", base.Add(-2*time.Hour), base.Add(-time.Hour)),
		testSession("starts-before", base.Add(-30*time.Minute), base.Add(15*time.Minute)),
		testSession("ends-after", base.Add(45*time.Minute), base.Add(90*time.Minute)),
		testSession("contains", base.Add(-time.Hour), base.Add(2*time.Hour)),
		testOpenSession("open", base.Add(50*time.Minute)),
	} {
		if err := st.SaveTaskSession(ctx, session); err != nil {
			t.Fatal(err)
		}
	}

	from := base
	to := base.Add(time.Hour)
	sessions, err := st.ListTaskSessions(ctx, domain.TaskSessionFilter{From: &from, To: &to, Overlap: true})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, session := range sessions {
		got[session.ID] = true
	}
	for _, id := range []string{"inside", "starts-before", "ends-after", "contains", "open"} {
		if !got[id] {
			t.Fatalf("expected overlap result to include %s, got %#v", id, got)
		}
	}
	if got["before"] {
		t.Fatalf("expected non-overlapping session to be excluded, got %#v", got)
	}
	count, err := st.CountTaskSessions(ctx, domain.TaskSessionFilter{From: &from, To: &to, Overlap: true})
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("expected overlap count 5, got %d", count)
	}
}

func TestListTaskSessionsPaginationUsesNewestFirstOrder(t *testing.T) {
	st, closeStore := newTestSQLiteStore(t)
	defer closeStore()
	ctx := context.Background()
	base := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	for i, id := range []string{"a", "b", "c", "d"} {
		started := base.Add(time.Duration(i) * time.Hour)
		if err := st.SaveTaskSession(ctx, testSession(id, started, started.Add(15*time.Minute))); err != nil {
			t.Fatal(err)
		}
	}

	sessions, err := st.ListTaskSessions(ctx, domain.TaskSessionFilter{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 || sessions[0].ID != "c" || sessions[1].ID != "b" {
		t.Fatalf("expected second page in newest-first order, got %#v", sessions)
	}
	count, err := st.CountTaskSessions(ctx, domain.TaskSessionFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("expected total count 4, got %d", count)
	}
}

func newTestSQLiteStore(t *testing.T) (*SQLiteStore, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	st := NewSQLiteStore(db)
	if err := st.Migrate(context.Background()); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	return st, func() { _ = db.Close() }
}

func testSession(id string, started time.Time, ended time.Time) domain.TaskSession {
	duration := uint32(ended.Sub(started).Seconds())
	return domain.TaskSession{
		ID:                id,
		DeviceID:          "d1",
		TaskID:            "task-" + id,
		TaskLabelSnapshot: "Task " + id,
		TaskIconSnapshot:  "code",
		TaskColorSnapshot: "#69d2a5",
		Facet:             1,
		StartedAt:         started,
		EndedAt:           &ended,
		DurationSeconds:   duration,
		Source:            "test",
	}
}

func testOpenSession(id string, started time.Time) domain.TaskSession {
	return domain.TaskSession{
		ID:                id,
		DeviceID:          "d1",
		TaskID:            "task-" + id,
		TaskLabelSnapshot: "Task " + id,
		TaskIconSnapshot:  "code",
		TaskColorSnapshot: "#69d2a5",
		Facet:             1,
		StartedAt:         started,
		Source:            "test",
	}
}
