package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// MySQL Commands
var mysqlCmd = &cobra.Command{
	Use:   "mysql",
	Short: "MySQL privilege management",
	Long:  `Manage MySQL database privileges including granting and revoking access.`,
}

var mysqlGrantCmd = &cobra.Command{
	Use:   "grant",
	Short: "Grant MySQL database access",
	Long: `Grant temporary access to a MySQL database with specified privileges.
Example: apollo-cli mysql grant --host db.example.com --database mydb --level read --duration 1h`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement MySQL grant logic
		return nil
	},
}

var mysqlRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke MySQL database access",
	Long:  `Revoke previously granted MySQL database access.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement MySQL revoke logic
		return nil
	},
}

var mysqlPingCmd = &cobra.Command{
	Use:   "ping [server]",
	Short: "Ping a MySQL server",
	Long: `Ping a MySQL server to check its hostname.
Example:
  apollo mysql ping my-server`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		server := args[0]

		// Create API client
		client := NewAPIClient(apiEndpoint)

		// Create ping job
		job, err := client.CreatePingJob(cmd.Context(), server)
		if err != nil {
			return fmt.Errorf("failed to create ping job: %v", err)
		}

		fmt.Printf("Created ping job %s\n", job.ID)

		// Wait for job completion
		job, err = client.WaitForJobCompletion(cmd.Context(), job.ID, time.Second*2)
		if err != nil {
			return fmt.Errorf("failed to complete ping job: %v", err)
		}

		fmt.Printf("Server hostname: %s\n", job.Result)
		return nil
	},
}

var mysqlListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered MySQL servers",
	Long: `List all registered MySQL servers with their connection details.
Example:
  apollo mysql list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create API client
		client := NewAPIClient(apiEndpoint)

		// Get list of servers
		servers, err := client.ListMySQLServers(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to list servers: %v", err)
		}

		// Print servers in a table format
		fmt.Printf("\nRegistered MySQL Servers:\n")
		fmt.Printf("------------------------\n")
		for _, server := range servers {
			fmt.Printf("Name:     %s\n", server.Name)
			fmt.Printf("Host:     %s\n", server.Host)
			fmt.Printf("Port:     %d\n", server.Port)
			fmt.Printf("User:     %s\n", server.User)
			fmt.Printf("Database: %s\n", server.Database)
			fmt.Printf("------------------------\n")
		}

		return nil
	},
}

// MySQL command flags
var (
	mysqlHost     string
	mysqlPort     int
	mysqlDatabase string
	mysqlLevel    string
	mysqlDuration string
	mysqlReason   string
	mysqlServer   string
)

// Kubernetes Commands
var kubernetesCmd = &cobra.Command{
	Use:   "kubernetes",
	Short: "Kubernetes privilege management",
	Long:  `Manage Kubernetes RBAC privileges including role and cluster role bindings.`,
}

var kubernetesGrantCmd = &cobra.Command{
	Use:   "grant",
	Short: "Grant Kubernetes access",
	Long: `Grant temporary Kubernetes access with specified privileges.
Example: apollo-cli kubernetes grant --namespace default --level read --duration 1h`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement Kubernetes grant logic
		return nil
	},
}

var kubernetesRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke Kubernetes access",
	Long:  `Revoke previously granted Kubernetes access.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement Kubernetes revoke logic
		return nil
	},
}

// Kubernetes command flags
var (
	k8sNamespace string
	k8sLevel     string
	k8sDuration  string
	k8sReason    string
)

func init() {
	// Add module commands to root
	rootCmd.AddCommand(mysqlCmd)
	rootCmd.AddCommand(kubernetesCmd)

	// MySQL command setup
	mysqlCmd.AddCommand(mysqlGrantCmd)
	mysqlCmd.AddCommand(mysqlRevokeCmd)
	mysqlCmd.AddCommand(mysqlPingCmd)
	mysqlCmd.AddCommand(mysqlListCmd)

	// MySQL ping command flags
	mysqlPingCmd.Flags().StringVar(&mysqlServer, "server", "", "Name of the registered MySQL server")
	mysqlPingCmd.MarkFlagRequired("server")

	mysqlGrantCmd.Flags().StringVar(&mysqlHost, "host", "", "MySQL server host")
	mysqlGrantCmd.Flags().IntVar(&mysqlPort, "port", 3306, "MySQL server port")
	mysqlGrantCmd.Flags().StringVar(&mysqlDatabase, "database", "", "Target database name")
	mysqlGrantCmd.Flags().StringVar(&mysqlLevel, "level", "", "Access level (read/write/admin)")
	mysqlGrantCmd.Flags().StringVar(&mysqlDuration, "duration", "1h", "Access duration (e.g., 1h, 30m)")
	mysqlGrantCmd.Flags().StringVar(&mysqlReason, "reason", "", "Reason for access request")

	mysqlRevokeCmd.Flags().String("grant-id", "", "ID of the grant to revoke")

	// Kubernetes command setup
	kubernetesCmd.AddCommand(kubernetesGrantCmd)
	kubernetesCmd.AddCommand(kubernetesRevokeCmd)

	kubernetesGrantCmd.Flags().StringVar(&k8sNamespace, "namespace", "", "Target namespace")
	kubernetesGrantCmd.Flags().StringVar(&k8sLevel, "level", "", "Access level (read/write/admin)")
	kubernetesGrantCmd.Flags().StringVar(&k8sDuration, "duration", "1h", "Access duration (e.g., 1h, 30m)")
	kubernetesGrantCmd.Flags().StringVar(&k8sReason, "reason", "", "Reason for access request")

	kubernetesRevokeCmd.Flags().String("grant-id", "", "ID of the grant to revoke")

	// Mark required flags
	mysqlGrantCmd.MarkFlagRequired("host")
	mysqlGrantCmd.MarkFlagRequired("database")
	mysqlGrantCmd.MarkFlagRequired("level")
	mysqlGrantCmd.MarkFlagRequired("reason")

	kubernetesGrantCmd.MarkFlagRequired("namespace")
	kubernetesGrantCmd.MarkFlagRequired("level")
	kubernetesGrantCmd.MarkFlagRequired("reason")
}

// Helper function to validate duration
func validateDuration(duration string) error {
	_, err := time.ParseDuration(duration)
	return err
}

// Helper function to validate access level
func validateAccessLevel(level string) error {
	validLevels := map[string]bool{
		"read":  true,
		"write": true,
		"admin": true,
	}

	if !validLevels[level] {
		return fmt.Errorf("invalid access level: %s. Must be one of: read, write, admin", level)
	}
	return nil
}
