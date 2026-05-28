package gcalsync

import (
	"fmt"
	"time"

	"google.golang.org/api/calendar/v3"
)

// DiffPlan holds the add and delete sets produced by PlanDiff.
type DiffPlan struct {
	Add    []*calendar.Event // source events to copy into the target calendar
	Delete []*calendar.Event // target managed events to remove
}

// PlanDiff compares source events against the target calendar's managed events
// and returns the set of events to add and delete.
//
// An event is kept unchanged only when a target managed event exists whose
// sourceEventID and fingerprint both match the source event.
// A changed source event (same ID, different fingerprint) results in a
// delete-then-add replacement.
func PlanDiff(sourceEvents, targetManagedEvents []*calendar.Event) *DiffPlan {
	plan := &DiffPlan{}

	// Index target managed events by sourceEventID|fingerprint.
	targetIndex := make(map[string]*calendar.Event, len(targetManagedEvents))
	for _, te := range targetManagedEvents {
		meta, ok := GetManagedMetadata(te)
		if !ok {
			continue
		}
		key := meta.SourceEventID + "|" + meta.Fingerprint
		targetIndex[key] = te
	}

	// Determine which source events need to be added.
	sourceKeys := make(map[string]bool, len(sourceEvents))
	for _, se := range sourceEvents {
		fp := Fingerprint(se)
		key := se.Id + "|" + fp
		sourceKeys[key] = true
		if _, exists := targetIndex[key]; !exists {
			plan.Add = append(plan.Add, se)
		}
	}

	// Determine which target managed events need to be deleted.
	for key, te := range targetIndex {
		if !sourceKeys[key] {
			plan.Delete = append(plan.Delete, te)
		}
	}

	return plan
}

// NearestEvent returns a slice containing the single event whose start time
// is on or after today and is closest to today.
// Returns nil if no such event exists.
func NearestEvent(events []*calendar.Event) []*calendar.Event {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var nearest *calendar.Event
	var nearestStart time.Time

	for _, e := range events {
		start, err := parseEventStart(e)
		if err != nil {
			continue
		}
		if start.Before(today) {
			continue
		}
		if nearest == nil || start.Before(nearestStart) {
			nearest = e
			nearestStart = start
		}
	}

	if nearest == nil {
		return nil
	}
	return []*calendar.Event{nearest}
}

func parseEventStart(e *calendar.Event) (time.Time, error) {
	if e.Start == nil {
		return time.Time{}, fmt.Errorf("event %s has no start", e.Id)
	}
	if e.Start.Date != "" {
		return time.ParseInLocation("2006-01-02", e.Start.Date, time.Local)
	}
	return time.Parse(time.RFC3339, e.Start.DateTime)
}
