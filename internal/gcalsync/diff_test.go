package gcalsync_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/tingtt/gcal-sync/internal/gcalsync"
	"google.golang.org/api/calendar/v3"
)

// buildManagedFromSource creates a target managed event mirroring a source event.
func buildManagedFromSource(targetID, srcCalID string, src *calendar.Event) *calendar.Event {
	fp := gcalsync.Fingerprint(src)
	return makeManagedEvent(targetID, srcCalID, src.Id, fp)
}

func TestPlanDiff_AddNewEvent(t *testing.T) {
	src := makeEvent("s1", "Meeting", "2025-06-01T09:00:00Z", "2025-06-01T10:00:00Z")
	plan := gcalsync.PlanDiff([]*calendar.Event{src}, nil)

	if len(plan.Add) != 1 {
		t.Errorf("Add count = %d, want 1", len(plan.Add))
	}
	if len(plan.Delete) != 0 {
		t.Errorf("Delete count = %d, want 0", len(plan.Delete))
	}
}

func TestPlanDiff_DeleteOrphanedManagedEvent(t *testing.T) {
	managed := makeManagedEvent("t1", "src-cal", "deleted-event", "somefp")
	plan := gcalsync.PlanDiff(nil, []*calendar.Event{managed})

	if len(plan.Delete) != 1 {
		t.Errorf("Delete count = %d, want 1", len(plan.Delete))
	}
	if len(plan.Add) != 0 {
		t.Errorf("Add count = %d, want 0", len(plan.Add))
	}
}

func TestPlanDiff_KeepUnchangedEvent(t *testing.T) {
	src := makeEvent("s1", "Meeting", "2025-06-01T09:00:00Z", "2025-06-01T10:00:00Z")
	managed := buildManagedFromSource("t1", "src-cal", src)

	plan := gcalsync.PlanDiff([]*calendar.Event{src}, []*calendar.Event{managed})

	if len(plan.Add) != 0 {
		t.Errorf("Add count = %d, want 0 (event unchanged)", len(plan.Add))
	}
	if len(plan.Delete) != 0 {
		t.Errorf("Delete count = %d, want 0 (event unchanged)", len(plan.Delete))
	}
}

func TestPlanDiff_ReplaceChangedEvent(t *testing.T) {
	// Original source event
	srcOld := makeEvent("s1", "Meeting", "2025-06-01T09:00:00Z", "2025-06-01T10:00:00Z")
	// Updated source event (same ID, different time)
	srcNew := makeEvent("s1", "Meeting", "2025-06-01T10:00:00Z", "2025-06-01T11:00:00Z")

	// Target has the old version
	managed := buildManagedFromSource("t1", "src-cal", srcOld)

	plan := gcalsync.PlanDiff([]*calendar.Event{srcNew}, []*calendar.Event{managed})

	if len(plan.Delete) != 1 {
		t.Errorf("Delete count = %d, want 1 (old version should be deleted)", len(plan.Delete))
	}
	if len(plan.Add) != 1 {
		t.Errorf("Add count = %d, want 1 (new version should be added)", len(plan.Add))
	}
}

func TestPlanDiff_UserCreatedEventNotDeleted(t *testing.T) {
	// A user-created event without managed metadata must NOT appear in Delete.
	userEvent := makeEvent("u1", "My personal event", "2025-06-01T09:00:00Z", "2025-06-01T10:00:00Z")

	plan := gcalsync.PlanDiff(nil, []*calendar.Event{userEvent})

	if len(plan.Delete) != 0 {
		t.Errorf("Delete count = %d, want 0 (user events must not be deleted)", len(plan.Delete))
	}
}

func TestNearestEvent_ReturnsClosest(t *testing.T) {
	now := time.Now()
	// Two future events; the second starts later.
	soon := now.Add(24 * time.Hour).Format(time.RFC3339)
	later := now.Add(48 * time.Hour).Format(time.RFC3339)
	end := now.Add(25 * time.Hour).Format(time.RFC3339)

	e1 := makeEvent("1", "Soon", soon, end)
	e2 := makeEvent("2", "Later", later, end)

	result := gcalsync.NearestEvent([]*calendar.Event{e2, e1}) // intentionally unordered
	if len(result) != 1 {
		t.Fatalf("got %d events, want 1", len(result))
	}
	if result[0].Id != "1" {
		t.Errorf("got event %q, want %q (the nearer event)", result[0].Id, "1")
	}
}

func TestNearestEvent_SkipsPastEvents(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)

	e := makeEvent("1", "Past", past, end)
	result := gcalsync.NearestEvent([]*calendar.Event{e})
	if len(result) != 0 {
		t.Errorf("got %d events, want 0 (past events must be skipped)", len(result))
	}
}

func TestNearestEvent_SingleFutureEvent(t *testing.T) {
	future := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	end := time.Now().Add(2 * time.Hour).Format(time.RFC3339)

	e := makeEvent("1", "Future", future, end)
	result := gcalsync.NearestEvent([]*calendar.Event{e})
	if len(result) != 1 {
		t.Fatalf("got %d events, want 1", len(result))
	}
}

func TestNearestEvent_AllEventsUsedWhenAll(t *testing.T) {
	// When --all is set the caller does NOT call NearestEvent; verify PlanDiff
	// handles multiple source events correctly.
	now := time.Now()
	events := make([]*calendar.Event, 3)
	for i := range events {
		start := now.Add(time.Duration(i+1) * 24 * time.Hour).Format(time.RFC3339)
		end := now.Add(time.Duration(i+1)*24*time.Hour + time.Hour).Format(time.RFC3339)
		events[i] = makeEvent(fmt.Sprintf("s%d", i+1), "Meeting", start, end)
	}

	plan := gcalsync.PlanDiff(events, nil)
	if len(plan.Add) != 3 {
		t.Errorf("Add count = %d, want 3", len(plan.Add))
	}
}
