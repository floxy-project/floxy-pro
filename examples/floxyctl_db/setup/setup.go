package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	floxy "github.com/rom8726/floxy-pro"
)

type HelloHandler struct{}

func (h *HelloHandler) Name() string {
	return "hello"
}

func (h *HelloHandler) Execute(ctx context.Context, stepCtx floxy.StepContext, input json.RawMessage) (json.RawMessage, error) {
	log.Printf("Hello handler executed for instance %d, step %s", stepCtx.InstanceID(), stepCtx.StepName())
	return input, nil
}

func main() {
	ctx := context.Background()

	connStr := "postgres://floxy:password@localhost:5435/floxy?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	if err := floxy.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	engine := floxy.NewEngine(pool)
	defer engine.Shutdown()

	engine.RegisterHandler(&HelloHandler{})

	workflowDef, err := floxy.NewBuilder("hello-world", 1).
		Step("say-hello", "hello", floxy.WithStepMaxRetries(3)).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	if err := engine.RegisterWorkflow(ctx, workflowDef); err != nil {
		log.Fatalf("Failed to register workflow: %v", err)
	}

	log.Printf("Workflow 'hello-world-v1' registered successfully in database")
	log.Printf("You can now use floxyctl commands to manage workflow instances")
}
