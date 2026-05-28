# Google OAuth Credential Design

This document describes how `gcal-sync` distributes and handles the Google OAuth Desktop-app credentials.

## Background

`gcal-sync` uses the **Google Calendar API v3** via Google's OAuth 2.0 Desktop application flow (RFC 8252).
A Desktop-app OAuth client requires a `credentials.json` file that contains a `client_id` and a `client_secret`.

Although the `client_secret` sounds sensitive, Google explicitly acknowledges that Desktop-app client secrets cannot truly be kept secret (they must live on the user's device). The secret is therefore treated as a _low-risk identifier_ rather than a password.

## Two Distribution Paths

### 1. GitHub Release Binary (embedded credentials)

| Step | Who | Action |
|------|-----|--------|
| 1 | Developer | Creates a Desktop-app OAuth client in Google Cloud Console and downloads `credentials.json` |
| 2 | CI (GitHub Actions) | Stores the JSON content as a repository secret (`GOOGLE_CREDENTIALS_JSON`) |
| 3 | CI | Before `go build`, writes the secret to `internal/auth/credentials.json` |
| 4 | CI | Runs `go build` — `go:embed` compiles the real credentials into the binary |
| 5 | User | Downloads the binary from GitHub Releases and runs it directly |

The user does **not** need to supply a `credentials.json`; it is embedded in the binary.

**GitHub Actions snippet (conceptual):**

```yaml
- name: Embed credentials
  run: echo "$GOOGLE_CREDENTIALS_JSON" > internal/auth/credentials.json
  env:
    GOOGLE_CREDENTIALS_JSON: ${{ secrets.GOOGLE_CREDENTIALS_JSON }}

- name: Build
  run: go build -o gcal-sync .
```

### 2. `go install` Binary (user-supplied credentials)

When a developer installs `gcal-sync` via `go install github.com/tingtt/gcal-sync@latest`, the compiled binary contains only the placeholder `credentials.json` (`{}`), which is invalid. The user must supply their own credentials:

1. Go to [Google Cloud Console](https://console.cloud.google.com/) → APIs & Services → Credentials.
2. Create an **OAuth 2.0 Client ID** of type **Desktop app**.
3. Download the generated `credentials.json`.
4. Pass it to every `gcal-sync` invocation using the `--credential` / `-c` flag:

```sh
gcal-sync --credential ~/credentials.json login
gcal-sync --credential ~/credentials.json sync <src-calendar-id>
```

## Credential Resolution Order

At runtime, `LoadConfig` in `internal/auth/auth.go` resolves credentials in this order:

1. **`--credential` / `-c` flag** — explicit file path supplied by the user.
2. **Embedded credentials** — compiled in by CI (only valid in release binaries).
3. **Error** — instructs the user to use `--credential`.

```
--credential flag supplied?
    ├─ yes → read that file
    └─ no  → embedded credentials valid?
                 ├─ yes → use embedded
                 └─ no  → error: use --credential
```

## Repository Layout

```
internal/auth/
├── auth.go               # credential loading, token persistence, HTTP client
└── credentials.json      # placeholder {} — replaced by CI for release builds
```

The placeholder `credentials.json` is committed to the repository so that `go:embed` compiles without errors. It is intentionally invalid so that `go install` users are prompted to supply their own credentials.

## Security Considerations

- The committed `credentials.json` is `{}` — it contains no real secrets.
- Real credentials are injected only by CI via a GitHub Actions secret and never pushed to the repository.
- Token files (`~/.config/gcal-sync/token.json`) are written with mode `0600` to prevent other users from reading them.
- Tokens can be revoked at any time via [Google Account Security](https://myaccount.google.com/permissions).
