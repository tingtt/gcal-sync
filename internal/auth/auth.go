package auth

import (
	"context"
	"encoding/json"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// embeddedCredentials is compiled into the binary when the developer places a
// real credentials.json in this directory before running go build.
// The committed placeholder ({}) is intentionally invalid; the runtime treats
// any parse failure as "no embedded credentials".
//
//go:embed credentials.json
var embeddedCredentials []byte

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gcal-sync"), nil
}

func tokenPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token.json"), nil
}

// LoadConfig returns an OAuth2 config using the following priority:
//
//  1. credentialPath (non-empty) — explicit file supplied via --credential / -c
//  2. Embedded credentials compiled into the binary (GitHub release build)
//  3. Error — instructs the user to use --credential
//
// For GitHub release builds, replace internal/auth/credentials.json with a
// real Desktop-app OAuth client JSON before running go build (done via CI secret).
// Users who installed via `go install` must pass --credential / -c.
func LoadConfig(credentialPath string) (*oauth2.Config, error) {
	// 1. Explicit file.
	if credentialPath != "" {
		b, err := os.ReadFile(credentialPath)
		if err != nil {
			return nil, fmt.Errorf("cannot read credentials file %q: %w", credentialPath, err)
		}
		config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
		if err != nil {
			return nil, fmt.Errorf("invalid credentials file %q: %w", credentialPath, err)
		}
		return config, nil
	}

	// 2. Embedded credentials (valid only when built with a real credentials.json).
	if config, err := google.ConfigFromJSON(embeddedCredentials, calendar.CalendarScope); err == nil {
		return config, nil
	}

	// 3. No credentials available.
	return nil, fmt.Errorf(
		"no credentials found.\n" +
			"Obtain a Desktop-app OAuth client JSON from Google Cloud Console and pass it with:\n" +
			"  gcal-sync --credential /path/to/credentials.json login",
	)
}

// SaveToken persists an OAuth2 token to ~/.config/gcal-sync/token.json.
func SaveToken(token *oauth2.Token) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path, err := tokenPath()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

// LoadToken reads the saved OAuth2 token.
func LoadToken() (*oauth2.Token, error) {
	path, err := tokenPath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("not logged in — run 'gcal-sync login' first: %w", err)
	}
	defer f.Close()
	token := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, err
	}
	return token, nil
}

// NewClient returns an HTTP client authorized with the saved token.
// credentialPath is forwarded to LoadConfig; see its documentation for priority rules.
func NewClient(ctx context.Context, credentialPath string) (*http.Client, error) {
	config, err := LoadConfig(credentialPath)
	if err != nil {
		return nil, err
	}
	token, err := LoadToken()
	if err != nil {
		return nil, err
	}
	return config.Client(ctx, token), nil
}
