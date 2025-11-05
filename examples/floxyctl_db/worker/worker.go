package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connStr := "postgres://floxy:password@localhost:5435/floxy?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	engine := floxy.NewEngine(pool)
	defer engine.Shutdown()

	engine.RegisterHandler(&HelloHandler{})

	workerPool := floxy.NewWorkerPool(engine, 2, 1*time.Second)

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	workerPool.Start(workerCtx)

	log.Println("Worker started. Processing workflow steps...")
	log.Println("Press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Println("Shutting down worker...")

	workerPool.Stop()
	workerCancel()
}
