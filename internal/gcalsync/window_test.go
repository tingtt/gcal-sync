package gcalsync_test

import (
	"testing"
	"time"

	"github.com/tingtt/gcal-sync/internal/gcalsync"
)

func TestBuildWindow_NoMonth(t *testing.T) {
	window, err := gcalsync.BuildWindow("", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if !window.Start.Equal(today) {
		t.Errorf("Start = %v, want %v", window.Start, today)
	}
	if window.End != nil {
		t.Errorf("End = %v, want nil (no upper bound)", *window.End)
	}
}

func TestBuildWindow_Month(t *testing.T) {
	window, err := gcalsync.BuildWindow("202506", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loc := time.Local
	wantStart := time.Date(2025, 6, 1, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2025, 7, 1, 0, 0, 0, 0, loc)

	if !window.Start.Equal(wantStart) {
		t.Errorf("Start = %v, want %v", window.Start, wantStart)
	}
	if window.End == nil {
		t.Fatal("End is nil, want a value")
	}
	if !window.End.Equal(wantEnd) {
		t.Errorf("End = %v, want %v", *window.End, wantEnd)
	}
}

func TestBuildWindow_NextMonth(t *testing.T) {
	window, err := gcalsync.BuildWindow("", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	now := time.Now()
	loc := now.Location()
	wantStart := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, loc)
	wantEnd := wantStart.AddDate(0, 1, 0)

	if !window.Start.Equal(wantStart) {
		t.Errorf("Start = %v, want %v", window.Start, wantStart)
	}
	if window.End == nil {
		t.Fatal("End is nil, want a value")
	}
	if !window.End.Equal(wantEnd) {
		t.Errorf("End = %v, want %v", *window.End, wantEnd)
	}
}

func TestBuildWindow_InvalidMonth(t *testing.T) {
	_, err := gcalsync.BuildWindow("invalid", false)
	if err == nil {
		t.Error("expected error for invalid month format, got nil")
	}
}
