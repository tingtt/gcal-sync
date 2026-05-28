package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/tingtt/gcal-sync/internal/auth"
	"golang.org/x/oauth2"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Google (opens browser)",
	RunE:  runLogin,
}

func runLogin(_ *cobra.Command, _ []string) error {
	config, err := auth.LoadConfig(flagCredential)
	if err != nil {
		return err
	}

	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("failed to generate state token: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	server := &http.Server{Addr: ":8080", Handler: mux}
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("OAuth callback: invalid state parameter")
			fmt.Fprintln(w, "Authentication failed — invalid state.")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("OAuth callback missing code parameter")
			fmt.Fprintln(w, "Authentication failed — no code received.")
			return
		}
		codeCh <- code
		fmt.Fprintln(w, "Authentication successful. You can close this tab.")
	})

	config.RedirectURL = "http://localhost:8080/callback"
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	fmt.Println("Opening browser for Google authentication...")
	fmt.Printf("If the browser does not open, visit:\n  %s\n\n", authURL)
	openBrowser(authURL)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	var code string
	select {
	case code = <-codeCh:
	case err = <-errCh:
		_ = server.Shutdown(context.Background())
		return err
	}
	_ = server.Shutdown(context.Background())

	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		return fmt.Errorf("failed to exchange authorization code: %w", err)
	}
	if err := auth.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("Login successful!")
	return nil
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}
