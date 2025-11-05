package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"

	"github.com/rom8726/floxy-pro"
	"github.com/rom8726/floxy-pro/internal/floxyd"
	"github.com/rom8726/floxy-pro/internal/handlers"
)

func main() {
	printBanner()

	if len(os.Args) < 2 {
		log.Fatal("Usage: floxyd <yaml-file>")
	}

	yamlFile := os.Args[1]

	config, err := floxyd.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := connectDB(ctx, config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := floxy.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	engine := floxy.NewEngine(pool)
	defer engine.Shutdown()

	yamlData, err := os.ReadFile(yamlFile)
	if err != nil {
		log.Fatalf("Failed to read YAML file: %v", err)
	}

	var yamlRoot floxy.YamlRoot
	if err := yaml.Unmarshal(yamlData, &yamlRoot); err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	workflowVersion := 1
	defs, _, err := floxy.ParseWorkflowYAML(yamlData, workflowVersion)
	if err != nil {
		log.Fatalf("Failed to parse workflow YAML: %v", err)
	}

	for name, def := range defs {
		if err := engine.RegisterWorkflow(ctx, def); err != nil {
			log.Fatalf("Failed to register workflow %q: %v", name, err)
		}
		log.Printf("Registered workflow: %s (version %d, id: %s)", name, def.Version, def.ID)
	}

	for _, handlerDef := range yamlRoot.Handlers {
		if handlerDef.Name == "" {
			log.Fatalf("Handler missing name")
		}
		if handlerDef.Exec == "" {
			log.Fatalf("Handler %q missing exec", handlerDef.Name)
		}

		handler, err := handlers.CreateHandler(
			handlerDef.Name,
			handlerDef.Exec,
			yamlRoot.TLS,
			handlerDef.TLS,
			false,
		)
		if err != nil {
			log.Fatalf("Failed to create handler %q: %v", handlerDef.Name, err)
		}

		engine.RegisterHandler(handler)
		log.Printf("Registered handler: %s", handlerDef.Name)
	}

	workerPool := floxy.NewWorkerPool(engine, config.Workers, config.WorkerInterval)

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	workerPool.Start(workerCtx)

	statsCtx, statsCancel := context.WithCancel(ctx)
	defer statsCancel()

	go printStats(statsCtx, pool)

	log.Printf("Floxyd started with %d workers", config.Workers)
	log.Println("Press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Println("Shutting down...")

	workerPool.Stop()
	workerCancel()
	statsCancel()
	cancel()

	log.Println("Floxyd stopped")
}

func connectDB(ctx context.Context, config *floxyd.Config) (*pgxpool.Pool, error) {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(config.DBUser, config.DBPassword),
		Host:   fmt.Sprintf("%s:%s", config.DBHost, config.DBPort),
		Path:   "/" + config.DBName,
	}

	q := u.Query()
	q.Set("sslmode", "disable")
	q.Set("search_path", "workflows")
	u.RawQuery = q.Encode()

	connStr := u.String()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

func printStats(ctx context.Context, pool *pgxpool.Pool) {
	store := floxy.NewStore(pool)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			statsCtx, cancel := context.WithTimeout(ctx, 10*time.Second)

			stats, err := store.GetSummaryStats(statsCtx)
			if err != nil {
				if ctx.Err() == nil {
					log.Printf("[STATS] Failed to get summary stats: %v", err)
				}
				cancel()
				continue
			}

			activeInstances, err := store.GetActiveInstances(statsCtx)
			if err != nil {
				if ctx.Err() == nil {
					log.Printf("[STATS] Failed to get active instances: %v", err)
				}
				cancel()
				continue
			}
			cancel()

			fmt.Println("\n=== Workflow Statistics ===")
			fmt.Printf("Total: %d | Completed: %d | Failed: %d | Running: %d | Pending: %d | Active: %d\n",
				stats.TotalWorkflows, stats.CompletedWorkflows, stats.FailedWorkflows,
				stats.RunningWorkflows, stats.PendingWorkflows, stats.ActiveWorkflows)

			if len(activeInstances) > 0 {
				fmt.Println("\n--- Active Workflow Instances ---")
				for _, inst := range activeInstances {
					runtime := time.Since(inst.StartedAt).Round(time.Second)
					currentStep := inst.CurrentStep
					if currentStep == "" {
						currentStep = "N/A"
					}
					fmt.Printf("  ID: %d | Workflow: %s | Status: %s | Step: %s | Steps: %d/%d | Rolled back: %d | Runtime: %s\n",
						inst.ID, inst.WorkflowName, inst.Status, currentStep,
						inst.CompletedSteps, inst.TotalSteps, inst.RolledBackSteps, runtime)
				}
			} else {
				fmt.Println("\n--- No active workflow instances ---")
			}
			fmt.Println()
		}
	}
}

func printBanner() {
	banner := `
███████╗██╗      ██████╗ ██╗  ██╗██╗   ██╗██████╗ 
██╔════╝██║     ██╔═══██╗╚██╗██╔╝╚██╗ ██╔╝██╔══██╗
█████╗  ██║     ██║   ██║ ╚███╔╝  ╚████╔╝ ██║  ██║
██╔══╝  ██║     ██║   ██║ ██╔██╗   ╚██╔╝  ██║  ██║
██║     ███████╗╚██████╔╝██╔╝ ██╗   ██║   ██████╔╝
╚═╝     ╚══════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚═════╝ 
                                                  
`
	fmt.Print(banner)
	fmt.Println("Workflow Engine Server")
	fmt.Println()
}
