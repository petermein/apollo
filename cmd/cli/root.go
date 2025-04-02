package main

import (
	"context"
	"fmt"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	clientID     string
	clientSecret string
	apiEndpoint  string
	cfgFile      string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "apollo-cli",
	Short: "Apollo CLI for privilege escalation management",
	Long: `Apollo CLI is a command-line interface for managing privilege escalations.
It provides secure access to temporary elevated privileges through OpenID Connect authentication.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.apollo-cli.yaml)")
	rootCmd.PersistentFlags().StringVar(&clientID, "client-id", "", "Google OAuth client ID")
	rootCmd.PersistentFlags().StringVar(&clientSecret, "client-secret", "", "Google OAuth client secret")
	rootCmd.PersistentFlags().StringVar(&apiEndpoint, "api-endpoint", "http://localhost:8080", "API server endpoint")

	// Mark required flags
	rootCmd.MarkPersistentFlagRequired("client-id")
	rootCmd.MarkPersistentFlagRequired("client-secret")

	// Add commands
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(requestCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".apollo-cli")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// getOIDCProvider returns an initialized OIDC provider
func getOIDCProvider(ctx context.Context) (*oidc.Provider, error) {
	return oidc.NewProvider(ctx, "https://accounts.google.com")
}

// getOAuthConfig returns an initialized OAuth2 config
func getOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8080/callback",
		Scopes: []string{
			oidc.ScopeOpenID,
			"profile",
			"email",
		},
	}
} 