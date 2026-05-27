# [WIP] gcal-sync

A CLI tool for syncing specific Google Calendar events to a personal (or specified) calendar.

## Installation

```sh
go install github.com/tingtt/gcal-sync@latest
```

## Usage

### Login

Authenticate with Google before syncing.

```sh
gcal-sync login
```

### Sync

```sh
gcal-sync <src calendar id> [options]
```

#### Options

| Option | Short | Description |
| --- | --- | --- |
| `--dest <calendar id>` | `-d` | Destination calendar ID. Defaults to the primary calendar. |
| `--name <event name>` | | Event name to sync (exact match). If omitted, **all events** are synced — shows a warning and prompts `y/n` to confirm. |
| `--all` | | Sync all matching events. Without this flag, only the nearest upcoming match (from today) is synced. |
| `--month <YYYYMM>` | | Restrict sync to a specific month (e.g. `202506`). |
| `--next-month` | | Shorthand for `--month` set to next month. |

#### Examples

```sh
# Sync the nearest upcoming "Team Standup" from a shared calendar
gcal-sync example@group.calendar.google.com --name "Team Standup"

# Sync all matching events in a specific month
gcal-sync example@group.calendar.google.com --name "Team Standup" --all --month 202506

# Sync all matching events in the next month
gcal-sync example@group.calendar.google.com --name "Team Standup" --all --next-month

# Sync to a specific destination calendar
gcal-sync example@group.calendar.google.com --name "Team Standup" --dest personal@example.com --all --next-month
```
