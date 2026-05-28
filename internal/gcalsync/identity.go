package gcalsync

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"google.golang.org/api/calendar/v3"
)

const (
	metaMarker      = "gcal-sync-marker"
	metaMarkerValue = "true"
	metaSrcCalID    = "gcal-sync-source-calendar-id"
	metaSrcEventID  = "gcal-sync-source-event-id"
	metaSrcStart    = "gcal-sync-source-start"
	metaSrcEnd      = "gcal-sync-source-end"
	metaFingerprint = "gcal-sync-source-fingerprint"
)

// ManagedMetadata holds the sync metadata embedded in a target event.
type ManagedMetadata struct {
	SourceCalendarID string
	SourceEventID    string
	SourceStart      string
	SourceEnd        string
	Fingerprint      string
}

// Fingerprint returns a stable hash of the event fields that define its identity:
// summary, start, end, and whether it is an all-day event.
func Fingerprint(e *calendar.Event) string {
	var start, end string
	allDay := false
	if e.Start != nil {
		if e.Start.Date != "" {
			start = e.Start.Date
			allDay = true
		} else {
			start = e.Start.DateTime
		}
	}
	if e.End != nil {
		if e.End.Date != "" {
			end = e.End.Date
		} else {
			end = e.End.DateTime
		}
	}
	raw := strings.Join([]string{e.Summary, start, end, fmt.Sprintf("%v", allDay)}, "|")
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h)
}

func eventStartStr(e *calendar.Event) string {
	if e.Start == nil {
		return ""
	}
	if e.Start.Date != "" {
		return e.Start.Date
	}
	return e.Start.DateTime
}

func eventEndStr(e *calendar.Event) string {
	if e.End == nil {
		return ""
	}
	if e.End.Date != "" {
		return e.End.Date
	}
	return e.End.DateTime
}

// IsManagedEvent reports whether the event was created by this tool.
func IsManagedEvent(e *calendar.Event) bool {
	if e.ExtendedProperties == nil || e.ExtendedProperties.Private == nil {
		return false
	}
	return e.ExtendedProperties.Private[metaMarker] == metaMarkerValue
}

// GetManagedMetadata extracts sync metadata from a target event.
// Returns false if the event was not created by this tool.
func GetManagedMetadata(e *calendar.Event) (*ManagedMetadata, bool) {
	if !IsManagedEvent(e) {
		return nil, false
	}
	p := e.ExtendedProperties.Private
	return &ManagedMetadata{
		SourceCalendarID: p[metaSrcCalID],
		SourceEventID:    p[metaSrcEventID],
		SourceStart:      p[metaSrcStart],
		SourceEnd:        p[metaSrcEnd],
		Fingerprint:      p[metaFingerprint],
	}, true
}

// ApplyMetadata embeds sync metadata into targetEvent using extendedProperties
// (authoritative) and a human-readable note appended to the description.
func ApplyMetadata(targetEvent *calendar.Event, srcCalendarID string, srcEvent *calendar.Event) {
	fp := Fingerprint(srcEvent)

	if targetEvent.ExtendedProperties == nil {
		targetEvent.ExtendedProperties = &calendar.EventExtendedProperties{}
	}
	if targetEvent.ExtendedProperties.Private == nil {
		targetEvent.ExtendedProperties.Private = make(map[string]string)
	}
	p := targetEvent.ExtendedProperties.Private
	p[metaMarker] = metaMarkerValue
	p[metaSrcCalID] = srcCalendarID
	p[metaSrcEventID] = srcEvent.Id
	p[metaSrcStart] = eventStartStr(srcEvent)
	p[metaSrcEnd] = eventEndStr(srcEvent)
	p[metaFingerprint] = fp

	note := fmt.Sprintf("[synced by gcal-sync from %s]", srcCalendarID)
	if targetEvent.Description != "" {
		targetEvent.Description = targetEvent.Description + "\n\n" + note
	} else {
		targetEvent.Description = note
	}
}
