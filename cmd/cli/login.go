package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Google OpenID Connect",
	Long: `Login authenticates the user with Google OpenID Connect.
It will open a browser window for authentication and store the credentials locally.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Get OIDC provider
		provider, err := getOIDCProvider(ctx)
		if err != nil {
			return fmt.Errorf("failed to get OIDC provider: %v", err)
		}

		// Get OAuth config
		config := getOAuthConfig()

		// Generate random state
		state := generateRandomState()

		// Create callback server
		callbackChan := make(chan string)
		server := &http.Server{
			Addr: ":8080",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/callback" {
					http.Error(w, "Invalid callback path", http.StatusBadRequest)
					return
				}

				if r.URL.Query().Get("state") != state {
					http.Error(w, "Invalid state", http.StatusBadRequest)
					return
				}

				code := r.URL.Query().Get("code")
				if code == "" {
					http.Error(w, "No code provided", http.StatusBadRequest)
					return
				}

				callbackChan <- code
				w.Write([]byte("Authentication successful! You can close this window."))
			}),
		}

		// Start callback server
		go server.ListenAndServe()

		// Generate auth URL
		authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)
		fmt.Printf("Please visit this URL to authenticate: %s\n", authURL)

		// Wait for callback
		code := <-callbackChan

		// Exchange code for token
		token, err := config.Exchange(ctx, code)
		if err != nil {
			return fmt.Errorf("failed to exchange code for token: %v", err)
		}

		// Get user info
		userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil {
			return fmt.Errorf("failed to get user info: %v", err)
		}

		// Store credentials
		if err := storeCredentials(token, userInfo); err != nil {
			return fmt.Errorf("failed to store credentials: %v", err)
		}

		fmt.Printf("Successfully logged in as %s\n", userInfo.Email)
		return nil
	},
}

// generateRandomState generates a random state string for OAuth flow
func generateRandomState() string {
	// Generate a random 32-byte string
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp if random generation fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// Credentials represents stored OAuth credentials
type Credentials struct {
	Token     *oauth2.Token `json:"token"`
	UserInfo  *UserInfo     `json:"user_info"`
	ExpiresAt time.Time     `json:"expires_at"`
}

// UserInfo represents the user information from OIDC
type UserInfo struct {
	Email     string `json:"email"`
	Subject   string `json:"sub"`
	GivenName string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

// storeCredentials stores the OAuth token and user info locally
func storeCredentials(token *oauth2.Token, userInfo *oidc.UserInfo) error {
	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	// Create .apollo-cli directory if it doesn't exist
	dir := filepath.Join(home, ".apollo-cli")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Create credentials file
	creds := Credentials{
		Token: token,
		UserInfo: &UserInfo{
			Email:     userInfo.Email,
			Subject:   userInfo.Subject,
			GivenName: userInfo.GivenName,
			FamilyName: userInfo.FamilyName,
		},
		ExpiresAt: time.Now().Add(token.Expiry.Sub(time.Now())),
	}

	// Marshal credentials
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %v", err)
	}

	// Write to file
	file := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(file, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %v", err)
	}

	return nil
} 