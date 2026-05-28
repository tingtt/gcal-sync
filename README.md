# gcal-sync

A CLI tool for syncing specific Google Calendar events to a personal (or specified) calendar.

## Installation

### GitHub Releases (recommended)

Download a pre-built binary from the [Releases](https://github.com/tingtt/gcal-sync/releases) page.
OAuth credentials are embedded — no extra setup required.

### `go install`

```sh
go install github.com/tingtt/gcal-sync@latest
```

> **Note:** Binaries built via `go install` do not contain embedded OAuth credentials.
> You must obtain a Desktop-app OAuth client JSON from [Google Cloud Console](https://console.cloud.google.com/) and pass it with `--credential` / `-c`.
> See [docs/DesignDoc_Google_OAuth.md](./docs/DesignDoc_Google_OAuth.md) for details.

## Prerequisites

Enable the **Google Calendar API** in your Google Cloud project:
<https://console.developers.google.com/apis/api/calendar-json.googleapis.com/overview>

## Usage

### Login

Authenticate with Google before syncing.

```sh
# Release binary (credentials embedded)
gcal-sync login

# go install binary (supply your own credentials.json)
gcal-sync --credential /path/to/credentials.json login
```

### Sync

```sh
gcal-sync [src-calendar-id] [options]
```

If `src-calendar-id` is omitted, an interactive calendar picker is shown (↑/↓ to navigate, Enter to select).

#### Options

| Option | Short | Description |
| --- | --- | --- |
| `--dest <calendar id>` | `-d` | Destination calendar ID. Defaults to the primary calendar. |
| `--name <string>` | | Search string to filter source events (Google Calendar full-text search). If omitted, all events are included. |
| `--month <YYYYMM>` | | Restrict sync to a specific month — syncs **all** matching events in that month. |
| `--next-month` | | Shorthand for `--month` set to next month. |
| `--prefix <string>` | | Prefix to prepend to the title of each created event. |
| `--color <color>` | | Color for created events. Accepts a number (`1`–`11`) or a name: `tomato`, `flamingo`, `tangerine`, `banana`, `sage`, `basil`, `peacock`, `blueberry`, `lavender`, `grape`, `graphite`. |
| `--credential <path>` | `-c` | Path to Google OAuth `credentials.json` (required for `go install` users). |

#### Sync behaviour

| `--month` specified | Result |
| --- | --- |
| Yes | All matching events in the given month |
| No | Only the nearest single upcoming event (from today) |

#### Examples

```sh
# Interactive calendar picker, nearest upcoming "Team Standup"
gcal-sync --name "Team Standup"

# Sync all matching events in next month, with a prefix and color
gcal-sync example@group.calendar.google.com \
  --name "Team Standup" \
  --next-month \
  --prefix "[Work] " \
  --color tomato

# Sync to a specific destination calendar for a given month
gcal-sync example@group.calendar.google.com \
  --name "Team Standup" \
  --month 202506 \
  --dest personal@example.com

# go install users: provide credentials explicitly
gcal-sync --credential ~/credentials.json \
  example@group.calendar.google.com \
  --name "Team Standup" \
  --next-month
```
