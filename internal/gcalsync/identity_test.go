package gcalsync_test

import (
	"testing"

	"github.com/tingtt/gcal-sync/internal/gcalsync"
	"google.golang.org/api/calendar/v3"
)

func makeEvent(id, summary, start, end string) *calendar.Event {
	return &calendar.Event{
		Id:      id,
		Summary: summary,
		Start:   &calendar.EventDateTime{DateTime: start},
		End:     &calendar.EventDateTime{DateTime: end},
	}
}

func makeManagedEvent(targetID, srcCalID, srcEventID, fingerprint string) *calendar.Event {
	return &calendar.Event{
		Id:      targetID,
		Summary: "managed",
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				"gcal-sync-marker":              "true",
				"gcal-sync-source-calendar-id":  srcCalID,
				"gcal-sync-source-event-id":     srcEventID,
				"gcal-sync-source-fingerprint":  fingerprint,
			},
		},
	}
}

func TestFingerprint_Deterministic(t *testing.T) {
	e := makeEvent("1", "Stand-up", "2025-06-01T09:00:00+09:00", "2025-06-01T09:15:00+09:00")
	fp1 := gcalsync.Fingerprint(e)
	fp2 := gcalsync.Fingerprint(e)
	if fp1 != fp2 {
		t.Errorf("fingerprint is not deterministic: %q vs %q", fp1, fp2)
	}
}

func TestFingerprint_DiffersOnChange(t *testing.T) {
	e1 := makeEvent("1", "Stand-up", "2025-06-01T09:00:00+09:00", "2025-06-01T09:15:00+09:00")
	e2 := makeEvent("1", "Stand-up", "2025-06-01T10:00:00+09:00", "2025-06-01T10:15:00+09:00")
	if gcalsync.Fingerprint(e1) == gcalsync.Fingerprint(e2) {
		t.Error("fingerprints should differ when start time changes")
	}
}

func TestIsManagedEvent_True(t *testing.T) {
	e := makeManagedEvent("t1", "cal", "s1", "fp")
	if !gcalsync.IsManagedEvent(e) {
		t.Error("expected IsManagedEvent to return true")
	}
}

func TestIsManagedEvent_False_NoProps(t *testing.T) {
	e := makeEvent("1", "Manual event", "2025-06-01T09:00:00Z", "2025-06-01T10:00:00Z")
	if gcalsync.IsManagedEvent(e) {
		t.Error("expected IsManagedEvent to return false for user-created event")
	}
}

func TestGetManagedMetadata(t *testing.T) {
	fp := "abc123"
	e := makeManagedEvent("t1", "src-cal", "src-event-1", fp)
	meta, ok := gcalsync.GetManagedMetadata(e)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if meta.SourceCalendarID != "src-cal" {
		t.Errorf("SourceCalendarID = %q, want %q", meta.SourceCalendarID, "src-cal")
	}
	if meta.SourceEventID != "src-event-1" {
		t.Errorf("SourceEventID = %q, want %q", meta.SourceEventID, "src-event-1")
	}
	if meta.Fingerprint != fp {
		t.Errorf("Fingerprint = %q, want %q", meta.Fingerprint, fp)
	}
}

func TestApplyMetadata_SetsMarker(t *testing.T) {
	src := makeEvent("s1", "Standup", "2025-06-01T09:00:00Z", "2025-06-01T09:15:00Z")
	target := &calendar.Event{Summary: "Standup"}

	gcalsync.ApplyMetadata(target, "my-cal", src)

	if !gcalsync.IsManagedEvent(target) {
		t.Error("target should be identified as managed after ApplyMetadata")
	}
	meta, ok := gcalsync.GetManagedMetadata(target)
	if !ok {
		t.Fatal("expected metadata after ApplyMetadata")
	}
	if meta.SourceEventID != "s1" {
		t.Errorf("SourceEventID = %q, want %q", meta.SourceEventID, "s1")
	}
	if meta.Fingerprint == "" {
		t.Error("Fingerprint should not be empty")
	}
}

func TestApplyMetadata_AppendsDescription(t *testing.T) {
	src := makeEvent("s1", "Standup", "2025-06-01T09:00:00Z", "2025-06-01T09:15:00Z")
	src.Description = "Original description."
	target := &calendar.Event{Summary: "Standup", Description: src.Description}

	gcalsync.ApplyMetadata(target, "my-cal", src)

	if target.Description == "Original description." {
		t.Error("description should have sync note appended")
	}
	if len(target.Description) < len("Original description.") {
		t.Error("description should not be shorter than original")
	}
}
