package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	apiEndpoint string
	cfgFile     string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "apollo-cli",
	Short: "Apollo CLI - Privilege Management Tool",
	Long: `Apollo CLI is a tool for managing privileged access across different systems.
It provides a unified interface for requesting and revoking access to various resources.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.apollo-cli.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiEndpoint, "api", "http://localhost:8080", "API server endpoint")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format (text/json)")

	// Add commands
	rootCmd.AddCommand(requestCmd)
	rootCmd.AddCommand(mysqlCmd)
	rootCmd.AddCommand(operatorCmd)
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

	// Set default values
	viper.SetDefault("api.endpoint", "http://localhost:8080")
	viper.SetDefault("api.retry_attempts", 3)
	viper.SetDefault("api.retry_delay", "5s")

	// Read config
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	// Bind flags to viper
	viper.BindPFlag("api.endpoint", rootCmd.PersistentFlags().Lookup("api"))

	// Update variables from viper
	apiEndpoint = viper.GetString("api.endpoint")
}
