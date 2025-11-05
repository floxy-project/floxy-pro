package floxyctl

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "floxyctl",
		Short: "Floxy CLI tool for running workflows",
		Long:  `floxyctl - command line tool for running and managing workflows from YAML files`,
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run workflow from YAML file",
		Long: `Run workflow from YAML file with given input.

Examples:
  # Run with input file
  floxyctl run -f workflow.yaml -i input.json

  # Run with input from stdin
  echo '{"key": "value"}' | floxyctl run -f workflow.yaml

  # Run with empty input
  floxyctl run -f workflow.yaml`,
		RunE: runCommand,
	}

	runCmd.Flags().StringP("file", "f", "", "YAML file with workflow configuration (required)")
	runCmd.Flags().StringP("input", "i", "", "JSON file with initial input (optional)")
	runCmd.Flags().IntP("workers", "w", 3, "Number of worker pool workers")
	runCmd.Flags().StringP("worker-interval", "", "100ms", "Worker pool polling interval (e.g., 100ms, 1s, 500ms)")
	runCmd.Flags().StringP("completion-timeout", "", "10m", "Timeout for workflow completion (e.g., 10m, 30m, 1h, 5m)")
	runCmd.Flags().StringP("status-check-interval", "", "500ms", "Interval for checking workflow status (e.g., 500ms, 1s)")
	runCmd.Flags().BoolP("debug", "D", false, "Enable debug mode (prints handler input/output)")

	if err := runCmd.MarkFlagRequired("file"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error marking file flag as required: %v\n", err)
		os.Exit(1)
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start workflow instance from database",
		Long: `Start a new workflow instance from a workflow definition stored in the database.

Examples:
  # Start workflow with input file
  floxyctl start -o workflow-definition-id -i input.json --host localhost --port 5432 --user user --database mydb -W

  # Start workflow with input from stdin
  echo '{"key": "value"}' | floxyctl start -o workflow-definition-id --host localhost --port 5432 --user user --database mydb -W`,
		RunE: startCommand,
	}

	addDBFlags(startCmd)
	startCmd.Flags().StringP("object", "o", "", "Workflow definition ID (required)")
	startCmd.Flags().StringP("input", "i", "", "JSON file with initial input (optional)")

	if err := startCmd.MarkFlagRequired("object"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error marking object flag as required: %v\n", err)
		os.Exit(1)
	}

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel workflow instance",
		Long: `Cancel a running workflow instance with rollback.

Examples:
  # Cancel workflow instance
  floxyctl cancel -o 123 --host localhost --port 5432 --user user --database mydb -W

  # Cancel with custom reason
  floxyctl cancel -o 123 --host localhost --port 5432 --user user --database mydb -W --reason "User requested"`,
		RunE: cancelCommand,
	}

	addDBFlags(cancelCmd)
	cancelCmd.Flags().StringP("object", "o", "", "Workflow instance ID (required)")
	cancelCmd.Flags().String("requested-by", "", "User/system requesting cancellation (default: $USER)")
	cancelCmd.Flags().String("reason", "", "Reason for cancellation (default: 'Cancelled via floxyctl')")

	if err := cancelCmd.MarkFlagRequired("object"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error marking object flag as required: %v\n", err)
		os.Exit(1)
	}

	abortCmd := &cobra.Command{
		Use:   "abort",
		Short: "Abort workflow instance",
		Long: `Abort a running workflow instance without rollback.

Examples:
  # Abort workflow instance
  floxyctl abort -o 123 --host localhost --port 5432 --user user --database mydb -W

  # Abort with custom reason
  floxyctl abort -o 123 --host localhost --port 5432 --user user --database mydb -W --reason "Critical error"`,
		RunE: abortCommand,
	}

	addDBFlags(abortCmd)
	abortCmd.Flags().StringP("object", "o", "", "Workflow instance ID (required)")
	abortCmd.Flags().String("requested-by", "", "User/system requesting abort (default: $USER)")
	abortCmd.Flags().String("reason", "", "Reason for abort (default: 'Aborted via floxyctl')")

	if err := abortCmd.MarkFlagRequired("object"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error marking object flag as required: %v\n", err)
		os.Exit(1)
	}

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(cancelCmd)
	rootCmd.AddCommand(abortCmd)

	return rootCmd
}

func addDBFlags(cmd *cobra.Command) {
	cmd.Flags().String("host", "", "Database host (required)")
	cmd.Flags().String("port", "", "Database port (required)")
	cmd.Flags().String("user", "", "Database user (required)")
	cmd.Flags().BoolP("password", "W", false, "Prompt for password")
	cmd.Flags().String("database", "", "Database name (required)")

	_ = cmd.MarkFlagRequired("host")
	_ = cmd.MarkFlagRequired("port")
	_ = cmd.MarkFlagRequired("user")
	_ = cmd.MarkFlagRequired("database")
}

func runCommand(cmd *cobra.Command, _ []string) error {
	yamlFile, err := cmd.Flags().GetString("file")
	if err != nil {
		return fmt.Errorf("failed to get file flag: %w", err)
	}

	inputFile, err := cmd.Flags().GetString("input")
	if err != nil {
		return fmt.Errorf("failed to get input flag: %w", err)
	}

	workers, err := cmd.Flags().GetInt("workers")
	if err != nil {
		return fmt.Errorf("failed to get workers flag: %w", err)
	}

	workerIntervalStr, err := cmd.Flags().GetString("worker-interval")
	if err != nil {
		return fmt.Errorf("failed to get worker-interval flag: %w", err)
	}
	workerInterval, err := parseDuration(workerIntervalStr)
	if err != nil {
		return fmt.Errorf("invalid worker-interval: %w", err)
	}

	completionTimeoutStr, err := cmd.Flags().GetString("completion-timeout")
	if err != nil {
		return fmt.Errorf("failed to get completion-timeout flag: %w", err)
	}
	completionTimeout, err := parseDuration(completionTimeoutStr)
	if err != nil {
		return fmt.Errorf("invalid completion-timeout: %w", err)
	}

	statusCheckIntervalStr, err := cmd.Flags().GetString("status-check-interval")
	if err != nil {
		return fmt.Errorf("failed to get status-check-interval flag: %w", err)
	}
	statusCheckInterval, err := parseDuration(statusCheckIntervalStr)
	if err != nil {
		return fmt.Errorf("invalid status-check-interval: %w", err)
	}

	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return fmt.Errorf("failed to get debug flag: %w", err)
	}

	config := Config{
		PoolWorkers:         workers,
		WorkerInterval:      workerInterval,
		CompletionTimeout:   completionTimeout,
		StatusCheckInterval: statusCheckInterval,
		Debug:               debug,
	}

	return RunWorkflow(cmd.Context(), yamlFile, inputFile, config)
}

func startCommand(cmd *cobra.Command, _ []string) error {
	workflowID, err := cmd.Flags().GetString("object")
	if err != nil {
		return fmt.Errorf("failed to get object flag: %w", err)
	}

	inputFile, err := cmd.Flags().GetString("input")
	if err != nil {
		return fmt.Errorf("failed to get input flag: %w", err)
	}

	dbConfig, err := getDBConfig(cmd)
	if err != nil {
		return err
	}

	pool, err := ConnectDB(cmd.Context(), dbConfig)
	if err != nil {
		return err
	}
	defer pool.Close()

	return StartWorkflow(cmd.Context(), pool, workflowID, inputFile)
}

func cancelCommand(cmd *cobra.Command, _ []string) error {
	objectID, err := cmd.Flags().GetString("object")
	if err != nil {
		return fmt.Errorf("failed to get object flag: %w", err)
	}

	requestedBy, err := cmd.Flags().GetString("requested-by")
	if err != nil {
		return fmt.Errorf("failed to get requested-by flag: %w", err)
	}

	reason, err := cmd.Flags().GetString("reason")
	if err != nil {
		return fmt.Errorf("failed to get reason flag: %w", err)
	}

	dbConfig, err := getDBConfig(cmd)
	if err != nil {
		return err
	}

	pool, err := ConnectDB(cmd.Context(), dbConfig)
	if err != nil {
		return err
	}
	defer pool.Close()

	return CancelWorkflow(cmd.Context(), pool, objectID, requestedBy, reason)
}

func abortCommand(cmd *cobra.Command, _ []string) error {
	objectID, err := cmd.Flags().GetString("object")
	if err != nil {
		return fmt.Errorf("failed to get object flag: %w", err)
	}

	requestedBy, err := cmd.Flags().GetString("requested-by")
	if err != nil {
		return fmt.Errorf("failed to get requested-by flag: %w", err)
	}

	reason, err := cmd.Flags().GetString("reason")
	if err != nil {
		return fmt.Errorf("failed to get reason flag: %w", err)
	}

	dbConfig, err := getDBConfig(cmd)
	if err != nil {
		return err
	}

	pool, err := ConnectDB(cmd.Context(), dbConfig)
	if err != nil {
		return err
	}
	defer pool.Close()

	return AbortWorkflow(cmd.Context(), pool, objectID, requestedBy, reason)
}

func getDBConfig(cmd *cobra.Command) (DBConfig, error) {
	host, err := cmd.Flags().GetString("host")
	if err != nil {
		return DBConfig{}, fmt.Errorf("failed to get host flag: %w", err)
	}

	port, err := cmd.Flags().GetString("port")
	if err != nil {
		return DBConfig{}, fmt.Errorf("failed to get port flag: %w", err)
	}

	user, err := cmd.Flags().GetString("user")
	if err != nil {
		return DBConfig{}, fmt.Errorf("failed to get user flag: %w", err)
	}

	database, err := cmd.Flags().GetString("database")
	if err != nil {
		return DBConfig{}, fmt.Errorf("failed to get database flag: %w", err)
	}

	needPassword, err := cmd.Flags().GetBool("password")
	if err != nil {
		return DBConfig{}, fmt.Errorf("failed to get password flag: %w", err)
	}

	password := ""
	if needPassword {
		password, err = ReadPassword()
		if err != nil {
			return DBConfig{}, err
		}
	}

	return DBConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: database,
	}, nil
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	return time.ParseDuration(s)
}
