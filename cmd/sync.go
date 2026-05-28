package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/tingtt/gcal-sync/internal/auth"
	"github.com/tingtt/gcal-sync/internal/gcalsync"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var syncCmd = &cobra.Command{
	Use:   "sync <src-calendar-id>",
	Short: "Sync events from a source calendar to the personal calendar",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSync,
}

var (
	flagDest      string
	flagName      string
	flagMonth     string
	flagNextMonth bool
	flagPrefix    string
	flagColor     string
)

func init() {
	syncCmd.Flags().StringVarP(&flagDest, "dest", "d", "", "destination calendar ID (default: primary calendar)")
	syncCmd.Flags().StringVar(&flagName, "name", "", "search string to filter events (Google Calendar full-text search)")
	syncCmd.Flags().StringVar(&flagMonth, "month", "", "restrict sync to this month (YYYYMM); syncs all matching events in the month")
	syncCmd.Flags().BoolVar(&flagNextMonth, "next-month", false, "shorthand for --month set to next month")
	syncCmd.Flags().StringVar(&flagPrefix, "prefix", "", "prefix to prepend to the summary of created events")
	syncCmd.Flags().StringVar(&flagColor, "color", "", "color for created events (1–11 or name: tomato, flamingo, tangerine, banana, sage, basil, peacock, blueberry, lavender, grape, graphite)")
}

func runSync(_ *cobra.Command, args []string) error {
	// Build Google Calendar service first (needed for interactive picker)
	ctx := context.Background()
	httpClient, err := auth.NewClient(ctx, flagCredential)
	if err != nil {
		return err
	}
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("failed to create calendar service: %w", err)
	}

	// Resolve source calendar ID — interactive picker if not provided
	srcCalendarID := ""
	if len(args) > 0 {
		srcCalendarID = args[0]
	} else {
		srcCalendarID, err = pickCalendar(svc)
		if err != nil {
			return err
		}
	}

	destCalendarID := flagDest
	if destCalendarID == "" {
		destCalendarID = "primary"
	}

	// Validation
	// (no required flags — all combinations are valid)

	// Build sync window
	window, err := gcalsync.BuildWindow(flagMonth, flagNextMonth)
	if err != nil {
		return err
	}

	// Fetch source events
	sourceEvents, err := listSourceEvents(svc, srcCalendarID, window, flagName)
	if err != nil {
		return fmt.Errorf("failed to fetch source events: %w", err)
	}

	// --month / --next-month → all matching events in the month
	// no --month              → nearest single event from today onwards
	if window.End == nil {
		sourceEvents = gcalsync.NearestEvent(sourceEvents)
	}

	// Fetch target managed events (only those created by this tool for this source)
	targetEvents, err := listTargetManagedEvents(svc, destCalendarID, srcCalendarID, window)
	if err != nil {
		return fmt.Errorf("failed to fetch target managed events: %w", err)
	}

	// Plan diff
	plan := gcalsync.PlanDiff(sourceEvents, targetEvents)

	// Always confirm before applying
	if err := confirmSync(srcCalendarID, window, plan); err != nil {
		return err
	}

	// Apply: delete first, then add
	deleted, added := 0, 0
	for _, te := range plan.Delete {
		if err := svc.Events.Delete(destCalendarID, te.Id).Do(); err != nil {
			return fmt.Errorf("failed to delete event %s: %w", te.Id, err)
		}
		deleted++
	}
	for _, se := range plan.Add {
		newEvent := buildTargetEvent(se, srcCalendarID, flagPrefix, flagColor)
		if _, err := svc.Events.Insert(destCalendarID, newEvent).Do(); err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
		added++
	}

	fmt.Printf("Sync complete: %d added, %d deleted.\n", added, deleted)
	return nil
}

// listSourceEvents fetches events from the source calendar within the window,
// filtering by exact name when name is non-empty.
func listSourceEvents(svc *calendar.Service, calendarID string, window *gcalsync.Window, name string) ([]*calendar.Event, error) {
	call := svc.Events.List(calendarID).
		TimeMin(window.Start.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		MaxResults(2500)
	if window.End != nil {
		call = call.TimeMax(window.End.Format(time.RFC3339))
	}
	if name != "" {
		call = call.Q(name)
	}

	var events []*calendar.Event
	for {
		resp, err := call.Do()
		if err != nil {
			return nil, err
		}
		for _, e := range resp.Items {
			events = append(events, e)
		}
		if resp.NextPageToken == "" {
			break
		}
		call = call.PageToken(resp.NextPageToken)
	}
	return events, nil
}

// listTargetManagedEvents fetches events in the destination calendar that were
// created by this tool for the given source calendar.
func listTargetManagedEvents(svc *calendar.Service, calendarID, srcCalendarID string, window *gcalsync.Window) ([]*calendar.Event, error) {
	call := svc.Events.List(calendarID).
		TimeMin(window.Start.Format(time.RFC3339)).
		PrivateExtendedProperty("gcal-sync-marker=true").
		SingleEvents(true).
		MaxResults(2500)
	if window.End != nil {
		call = call.TimeMax(window.End.Format(time.RFC3339))
	}

	var events []*calendar.Event
	for {
		resp, err := call.Do()
		if err != nil {
			return nil, err
		}
		for _, e := range resp.Items {
			meta, ok := gcalsync.GetManagedMetadata(e)
			if !ok || meta.SourceCalendarID != srcCalendarID {
				continue
			}
			events = append(events, e)
		}
		if resp.NextPageToken == "" {
			break
		}
		call = call.PageToken(resp.NextPageToken)
	}
	return events, nil
}

// colorNameToID maps human-friendly color names to Google Calendar colorId values.
var colorNameToID = map[string]string{
	"tomato":     "11",
	"flamingo":   "4",
	"tangerine":  "6",
	"banana":     "5",
	"sage":       "2",
	"basil":      "10",
	"peacock":    "7",
	"blueberry":  "9",
	"lavender":   "1",
	"grape":      "3",
	"graphite":   "8",
}

// resolveColorID converts a color name or numeric string to a Google Calendar colorId.
// Returns the input unchanged if it is already numeric or empty.
func resolveColorID(color string) string {
	if color == "" {
		return ""
	}
	if id, ok := colorNameToID[strings.ToLower(color)]; ok {
		return id
	}
	return color // assume already a numeric ID
}

// buildTargetEvent creates a new calendar event based on the source event,
// embedding sync metadata and applying optional prefix / color overrides.
func buildTargetEvent(src *calendar.Event, srcCalendarID, prefix, color string) *calendar.Event {
	summary := src.Summary
	if prefix != "" {
		summary = prefix + summary
	}
	colorID := src.ColorId
	if color != "" {
		colorID = resolveColorID(color)
	}
	newEvent := &calendar.Event{
		Summary:  summary,
		Location: src.Location,
		Start:    src.Start,
		End:      src.End,
		ColorId:  colorID,
	}
	if src.Description != "" {
		newEvent.Description = src.Description
	}
	gcalsync.ApplyMetadata(newEvent, srcCalendarID, src)
	return newEvent
}

// confirmSync displays the planned changes and asks the user to confirm.
func confirmSync(srcCalendarID string, window *gcalsync.Window, plan *gcalsync.DiffPlan) error {
	var monthLabel string
	if window.End == nil {
		monthLabel = window.Start.Format("2006-01-02") + " onwards (nearest 1 event)"
	} else {
		monthLabel = window.Start.Format("2006-01")
	}

	fmt.Println("This operation will sync events from:")
	fmt.Printf("calendar: %s\n", srcCalendarID)
	fmt.Printf("month:    %s\n", monthLabel)
	fmt.Println()
	fmt.Println("Planned changes:")
	fmt.Printf("+ add:    %d events\n", len(plan.Add))
	fmt.Printf("- delete: %d events\n", len(plan.Delete))
	fmt.Println()
	fmt.Print("Are you sure? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" {
		fmt.Println("Aborted.")
		return fmt.Errorf("sync cancelled by user")
	}
	return nil
}

// pickCalendar fetches the user's calendar list and presents an interactive
// selector (↑/↓ to navigate, Enter to confirm). Returns the selected calendar ID.
func pickCalendar(svc *calendar.Service) (string, error) {
	resp, err := svc.CalendarList.List().Do()
	if err != nil {
		return "", fmt.Errorf("failed to list calendars: %w", err)
	}
	if len(resp.Items) == 0 {
		return "", fmt.Errorf("no calendars found in your account")
	}

	labels := make([]string, len(resp.Items))
	for i, c := range resp.Items {
		label := c.Summary
		if c.Description != "" {
			label += " — " + c.Description
		}
		label += " (" + c.Id + ")"
		labels[i] = label
	}

	prompt := promptui.Select{
		Label: "Select source calendar",
		Items: labels,
		Size:  10,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("calendar selection cancelled: %w", err)
	}
	return resp.Items[idx].Id, nil
}
