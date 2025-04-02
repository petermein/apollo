package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	resourceID string
	level      string
	duration   string
	reason     string
)

var requestCmd = &cobra.Command{
	Use:   "request",
	Short: "Request privilege escalation",
	Long: `Request creates a new privilege escalation request.
It will be reviewed by an operator.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate required flags
		if resourceID == "" {
			return fmt.Errorf("resource-id is required")
		}
		if level == "" {
			return fmt.Errorf("level is required")
		}
		if duration == "" {
			return fmt.Errorf("duration is required")
		}
		if reason == "" {
			return fmt.Errorf("reason is required")
		}

		// Parse duration
		parsedDuration, err := time.ParseDuration(duration)
		if err != nil {
			return fmt.Errorf("invalid duration format: %v", err)
		}

		fmt.Printf("Requesting privilege escalation:\n")
		fmt.Printf("Resource: %s\n", resourceID)
		fmt.Printf("Level: %s\n", level)
		fmt.Printf("Duration: %s\n", parsedDuration)
		fmt.Printf("Reason: %s\n", reason)

		return nil
	},
}

func init() {
	requestCmd.Flags().StringVar(&resourceID, "resource-id", "", "ID of the resource requiring access")
	requestCmd.Flags().StringVar(&level, "level", "", "Required privilege level")
	requestCmd.Flags().StringVar(&duration, "duration", "", "Duration of the privilege grant (e.g., 1h, 30m)")
	requestCmd.Flags().StringVar(&reason, "reason", "", "Reason for privilege escalation")

	// Mark required flags
	requestCmd.MarkFlagRequired("resource-id")
	requestCmd.MarkFlagRequired("level")
	requestCmd.MarkFlagRequired("duration")
	requestCmd.MarkFlagRequired("reason")
}
