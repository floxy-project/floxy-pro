package floxyctl

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

func AbortWorkflow(ctx context.Context, pool *pgxpool.Pool, objectID, requestedBy, reason string) error {
	instanceID, err := strconv.ParseInt(objectID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid instance ID: %w", err)
	}

	engine, err := CreateEngineFromDB(ctx, pool)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}
	defer engine.Shutdown()

	if requestedBy == "" {
		requestedBy = os.Getenv("USER")
		if requestedBy == "" {
			requestedBy = "floxyctl"
		}
	}

	if reason == "" {
		reason = "Aborted via floxyctl"
	}

	if err := engine.AbortWorkflow(ctx, instanceID, requestedBy, reason); err != nil {
		return fmt.Errorf("failed to abort workflow: %w", err)
	}

	fmt.Printf("Workflow instance %d abort requested\n", instanceID)

	return nil
}
