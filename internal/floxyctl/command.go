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

	rootCmd.AddCommand(runCmd)

	return rootCmd
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

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	return time.ParseDuration(s)
}
