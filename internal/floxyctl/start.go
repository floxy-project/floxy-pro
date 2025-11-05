package floxyctl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func StartWorkflow(ctx context.Context, pool *pgxpool.Pool, workflowID, inputFile string) error {
	engine, err := CreateEngineFromDB(ctx, pool)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}
	defer engine.Shutdown()

	var input json.RawMessage
	if inputFile != "" {
		inputData, err := os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read input file: %w", err)
		}

		if !json.Valid(inputData) {
			return fmt.Errorf("input file is not valid JSON")
		}

		input = inputData
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			inputData, err := io.ReadAll(os.Stdin)
			if err != nil && err != io.EOF {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}

			if len(inputData) > 0 {
				if !json.Valid(inputData) {
					return fmt.Errorf("stdin input is not valid JSON")
				}
				input = inputData
			} else {
				input = json.RawMessage("{}")
			}
		} else {
			input = json.RawMessage("{}")
		}
	}

	instanceID, err := engine.Start(ctx, workflowID, input)
	if err != nil {
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	fmt.Printf("Workflow instance started with ID: %d\n", instanceID)

	return nil
}
