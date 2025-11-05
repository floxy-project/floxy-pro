package floxyctl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/rom8726/floxy-pro"
	"github.com/rom8726/floxy-pro/internal/handlers"
)

type Config struct {
	PoolWorkers         int
	WorkerInterval      time.Duration
	CompletionTimeout   time.Duration
	StatusCheckInterval time.Duration
	Debug               bool
}

func RunWorkflow(ctx context.Context, yamlFile, inputFile string, config Config) error {
	yamlData, err := os.ReadFile(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to read YAML file: %w", err)
	}

	var yamlRoot floxy.YamlRoot
	if err := yaml.Unmarshal(yamlData, &yamlRoot); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	defs, handlersExec, err := floxy.ParseWorkflowYAML(yamlData, 1)
	if err != nil {
		return fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	if len(defs) == 0 {
		return fmt.Errorf("no workflows defined in YAML file")
	}

	var workflowDef *floxy.WorkflowDefinition
	for _, def := range defs {
		workflowDef = def

		break
	}
	if workflowDef == nil {
		return fmt.Errorf("no workflows defined in YAML file")
	}

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
	}

	store := floxy.NewMemoryStore()
	txManager := floxy.NewMemoryTxManager()

	engine := floxy.NewEngine(nil,
		floxy.WithEngineStore(store),
		floxy.WithEngineTxManager(txManager),
	)
	defer engine.Shutdown()

	handlerMap := make(map[string]floxy.YamlHandler)
	for _, h := range yamlRoot.Handlers {
		handlerMap[h.Name] = h
	}

	for handlerName, exec := range handlersExec {
		var handlerDef floxy.YamlHandler
		var ok bool
		if handlerDef, ok = handlerMap[handlerName]; !ok {
			handlerDef = floxy.YamlHandler{
				Name: handlerName,
				Exec: exec,
			}
		}

		handler, err := handlers.CreateHandler(
			handlerName,
			exec,
			yamlRoot.TLS,
			handlerDef.TLS,
			config.Debug,
		)
		if err != nil {
			return fmt.Errorf("failed to create handler %q: %w", handlerName, err)
		}

		engine.RegisterHandler(handler)
	}

	if err := engine.RegisterWorkflow(ctx, workflowDef); err != nil {
		return fmt.Errorf("failed to register workflow: %w", err)
	}

	instanceID, err := engine.Start(ctx, workflowDef.ID, input)
	if err != nil {
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	fmt.Printf("Workflow started with instance ID: %d\n", instanceID)

	workerPool := floxy.NewWorkerPool(engine, config.PoolWorkers, config.WorkerInterval)
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	workerPool.Start(workerCtx)

	_, err = waitForCompletion(ctx, engine, instanceID, config.CompletionTimeout, config.StatusCheckInterval)
	if err != nil {
		return fmt.Errorf("error waiting for workflow completion: %w", err)
	}

	instance, err := store.GetInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	fmt.Printf("Workflow completed with status: %s\n", instance.Status)

	if instance.Status == floxy.StatusCompleted {
		if len(instance.Output) > 0 {
			fmt.Printf("Output: %s\n", string(instance.Output))
		}
	} else if instance.Status == floxy.StatusFailed {
		if instance.Error != nil {
			fmt.Printf("Error: %s\n", *instance.Error)
		}
		return fmt.Errorf("workflow failed")
	}

	steps, err := engine.GetSteps(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get steps: %w", err)
	}

	fmt.Printf("\nSteps executed:\n")
	for _, step := range steps {
		statusIcon := "✓"
		if step.Status == floxy.StepStatusFailed {
			statusIcon = "✗"
		} else if step.Status != floxy.StepStatusCompleted {
			statusIcon = "○"
		}

		fmt.Printf("  %s %s (%s)", statusIcon, step.StepName, step.Status)
		if step.Error != nil {
			fmt.Printf(" - Error: %s", *step.Error)
		}
		fmt.Println()
	}

	return nil
}

func waitForCompletion(
	ctx context.Context,
	engine *floxy.Engine,
	instanceID int64,
	timeout, checkInterval time.Duration,
) (floxy.WorkflowStatus, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			status, err := engine.GetStatus(ctx, instanceID)
			if err != nil {
				return "", err
			}

			if status == floxy.StatusCompleted ||
				status == floxy.StatusFailed ||
				status == floxy.StatusCancelled ||
				status == floxy.StatusAborted {
				return status, nil
			}

			if time.Now().After(deadline) {
				return status, fmt.Errorf("timeout waiting for workflow completion")
			}
		}
	}
}
