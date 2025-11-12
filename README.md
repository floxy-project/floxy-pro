# floxy-pro

[![Go Reference](https://pkg.go.dev/badge/github.com/rom8726/floxy.svg)](https://pkg.go.dev/github.com/rom8726/floxy)
[![Go Report Card](https://goreportcard.com/badge/github.com/rom8726/floxy)](https://goreportcard.com/report/github.com/rom8726/floxy)
[![Coverage Status](https://coveralls.io/repos/github/rom8726/floxy/badge.svg?branch=main)](https://coveralls.io/github/rom8726/floxy?branch=main)

[![boosty-cozy](https://gideonwhite1029.github.io/badges/cozy-boosty_vector.svg)](https://boosty.to/dev-tools-hacker)

A Go library for creating and executing workflows with a custom DSL. Implements the Saga pattern with orchestrator approach, providing transaction management and compensation capabilities.

**This is the Pro version of the Floxy library**, which includes advanced features for production use:

- **Partitioned Tables**: Database schema redesigned with partitioned tables using PostgreSQL `pg_partman` extension for efficient management of large data volumes
- **floxyctl**: CLI tool for running workflows with in-memory store or managing workflow instances (start/cancel/abort) stored in PostgreSQL
- **floxyd**: Ready-to-use runtime daemon for continuous workflow processing with support for bash and HTTP handlers

floxy means "flow" + "flux" + "tiny".

> Disclaimer: This app has not been tested in production yet. By using it, you are acting at your own risk.

<img src="docs/floxy_logo.png" width="500">

## Table of Contents

- [Features](#features)
- [Pro Version Features](#pro-version-features)
  - [Partitioned Tables with pg_partman](#partitioned-tables-with-pg_partman)
  - [floxyctl - CLI Tool](#floxyctl---cli-tool)
  - [floxyd - Runtime Daemon](#floxyd---runtime-daemon)
- [Ecosystem](#ecosystem)
- [Why Floxy?](#why-floxy)
  - [1. Lightweight, Not Heavyweight](#1-lightweight-not-heavyweight)
  - [2. Pragmatic by Design](#2-pragmatic-by-design)
  - [3. Embedded](#3-embedded)
- [Quick Start](#quick-start)
- [Examples](#examples)
- [Integration Tests](#integration-tests)
- [Database Migrations](#database-migrations)
- [Dead Letter Queue](#dead-letter-queue-dlq)
- [Known Issues](#known-issues)
  - [Condition Steps in Forked Branches](#condition-steps-in-forked-branches)
- [Installation](#installation)
- [Dependencies](#dependencies)

## Features

- **Workflow DSL**: Declarative workflow definition using Builder pattern
- **Saga Pattern**: Orchestrator-based saga implementation with compensation
- **Workflows Versioning**: Safe changing flows using versions
- **Transaction Management**: Built-in transaction support with rollback capabilities
- **Parallel Execution**: Fork/Join patterns for concurrent workflow steps with dynamic wait-for detection
- **Error Handling**: Automatic retry mechanisms and failure compensation
- **SavePoints**: Rollback to specific points in workflow execution
- **Conditional branching** with Condition steps. Smart rollback for parallel flows with condition steps
- **Human-in-the-loop**: Interactive workflow steps that pause execution for human decisions
- **Cancel\Abort**: Possibility to cancel workflow with rollback to the root step and immediate abort workflow
- **Dead Letter Queue (DLQ)**: Two modes for error handling - Classic Saga with rollback/compensation or DLQ Mode with paused workflow and manual recovery
- **Distributed Mode**: Microservices can register only their handlers; steps without local handlers are returned to queue for other services to process
- **Priority Aging**: Prevents queue starvation by gradually increasing step priority as waiting time increases
- **PostgreSQL Storage**: Persistent workflow state and event logging
- **Migrations**: Embedded database migrations with `go:embed`

PlantUML diagrams of compensations flow: [DIAGRAMS](docs/SAGA_COMPENSATION_DIAGRAMS.md)

Engine specification: [ENGINE](docs/ENGINE_SPEC.md)

## Pro Version Features

### Partitioned Tables with pg_partman

The Pro version uses partitioned tables managed by PostgreSQL's `pg_partman` extension for efficient data management at scale. All high-volume tables (`workflow_instances`, `workflow_steps`, `workflow_events`, `workflow_dlq`) are partitioned by `created_at` with daily partitions.

**Key benefits:**
- **Automatic partition management**: `pg_partman` automatically creates new partitions (30 days ahead) and removes old ones (90 days retention)
- **Improved query performance**: Queries can leverage partition pruning for faster execution
- **Easier maintenance**: Old data can be dropped by dropping partitions instead of deleting rows
- **Scalability**: Handles millions of workflow instances and steps efficiently

**Partition configuration:**
- Partition interval: 1 day
- Premake: 30 partitions ahead
- Retention: 90 days (automatic cleanup)
- Partition key: `created_at` timestamp

The partitioned schema is defined in `migrations_pro/001_initial.up.sql` and requires the `pg_partman` extension to be installed in PostgreSQL.

### floxyctl - CLI Tool

`floxyctl` is a command-line tool for running and managing workflows. It supports two modes of operation:

#### 1. Run Mode (In-Memory Store)

Execute workflows directly from YAML files using an in-memory store. Perfect for testing, development, and one-off workflow executions.

**Commands:**
- `floxyctl run -f workflow.yaml [-i input.json]` - Run workflow from YAML file

**Features:**
- Runs workflow to completion synchronously
- Uses in-memory store (no database required)
- Supports bash script handlers (inline or file-based)
- Configurable worker pool and timeouts
- Debug mode for inspecting handler input/output

**Example:**
```bash
# Run workflow with input file
floxyctl run -f workflow.yaml -i input.json

# Run with input from stdin
echo '{"key": "value"}' | floxyctl run -f workflow.yaml

# Run with custom worker settings
floxyctl run -f workflow.yaml -w 5 --worker-interval 50ms --completion-timeout 5m
```

#### 2. Database Mode (PostgreSQL)

Manage workflow instances stored in PostgreSQL database. Requires database connection parameters.

**Commands:**
- `floxyctl start -o workflow-id [--host HOST --port PORT --user USER --database DB]` - Start new workflow instance
- `floxyctl cancel -o instance-id [--host HOST --port PORT --user USER --database DB]` - Cancel workflow with rollback
- `floxyctl abort -o instance-id [--host HOST --port PORT --user USER --database DB]` - Abort workflow without rollback

**Features:**
- Start workflow instances from registered workflow definitions
- Cancel workflows with automatic rollback to root step
- Abort workflows immediately without rollback
- Password can be provided via `-W` flag (prompt) or `PG_PASSWORD` environment variable
- Automatically runs database migrations on connection

**Example:**
```bash
# Start workflow instance
floxyctl start -o my-workflow-v1 \
  --host localhost --port 5432 --user floxy --database floxy \
  -i input.json -W

# Cancel workflow (with rollback)
floxyctl cancel -o 123 \
  --host localhost --port 5432 --user floxy --database floxy \
  --reason "User requested cancellation" -W

# Abort workflow (without rollback)
floxyctl abort -o 123 \
  --host localhost --port 5432 --user floxy --database floxy \
  --reason "Critical error" -W
```

**Handler Support:**
- Bash script handlers (inline scripts or file paths)
- HTTP endpoint handlers (automatically detected by `http://` or `https://` prefix)
- TLS configuration for secure HTTP handlers
- Debug mode for troubleshooting

### floxyd - Runtime Daemon

`floxyd` is a daemon service that continuously processes workflows stored in PostgreSQL. It's designed to run as a long-running service with multiple workers.

**Key Features:**
- **Continuous Processing**: Long-running workers poll database for pending steps
- **YAML Configuration**: Loads handlers and workflow definitions from YAML file
- **Multiple Handler Types**: Supports both bash scripts and HTTP endpoints
- **TLS Support**: Configurable TLS for secure HTTP handlers (global and per-handler)
- **Worker Pool**: Configurable number of workers and polling intervals
- **Statistics**: Real-time workflow statistics printed every 10 seconds
- **Tech Server**: Built-in HTTP server with Prometheus metrics and health check endpoints
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals cleanly

**Configuration:**

Environment variables:
- `FLOXY_DB_HOST` - Database host (required)
- `FLOXY_DB_PORT` - Database port (required)
- `FLOXY_DB_USER` - Database user (required)
- `FLOXY_DB_PASSWORD` - Database password (optional)
- `FLOXY_DB_NAME` - Database name (required)
- `FLOXY_WORKERS` - Number of workers (default: 3)
- `FLOXY_WORKER_INTERVAL` - Worker polling interval (default: "100ms")

**YAML Configuration:**

```yaml
tls:
  skip_verify: false
  cert_file: /path/to/cert.pem
  key_file: /path/to/key.pem
  ca_file: /path/to/ca.pem

handlers:
  - name: bash_handler
    exec: |
      echo "$INPUT" | jq '.value * 2'
  
  - name: script_handler
    exec: ./scripts/process.sh
  
  - name: http_handler
    exec: https://api.example.com/process
    tls:
      skip_verify: true

flows:
  - name: my_workflow
    steps:
      - name: step1
        handler: bash_handler
      - name: step2
        handler: http_handler
```

**Usage:**
```bash
# Set environment variables
export FLOXY_DB_HOST=localhost
export FLOXY_DB_PORT=5432
export FLOXY_DB_USER=floxy
export FLOXY_DB_PASSWORD=password
export FLOXY_DB_NAME=floxy
export FLOXY_WORKERS=5

# Run floxyd
./floxyd handlers.yaml
```

**Tech Server Endpoints:**
- `http://localhost:8081/metrics` - Prometheus metrics
- `http://localhost:8081/health` - Health check (checks database connection)

**Workflow Registration:**
On startup, `floxyd`:
1. Parses YAML file for workflow definitions (`flows` section)
2. Registers each workflow in the database (version 1 by default)
3. Logs each registered workflow with its ID and version

**Handler Types:**

**Bash Handlers:**
- Inline scripts or file paths
- Receive JSON input via `$INPUT` environment variable
- Input fields available as uppercase environment variables
- Workflow context via `FLOXY_*` environment variables
- Must output valid JSON to stdout

**HTTP Handlers:**
- Automatically detected by `http://` or `https://` prefix
- POST requests with JSON body containing `metadata` and `data` fields
- Custom headers: `X-Floxy-Instance-ID`, `X-Floxy-Step-Name`, `X-Floxy-Idempotency-Key`, `X-Floxy-Retry-Count`
- TLS configuration support (client certificates, CA certificates, skip verify)

**Statistics Output:**
Every 10 seconds, `floxyd` prints:
- Total, completed, failed, running, pending, and active workflow counts
- Active workflow instances with details (ID, workflow name, status, current step, progress, runtime)

**Differences from floxyctl:**

| Feature | floxyctl | floxyd |
|---------|----------|--------|
| **Mode** | CLI tool | Daemon service |
| **Storage** | In-memory (run mode) or PostgreSQL | PostgreSQL only |
| **Execution** | Runs workflow to completion | Continuous processing |
| **Workers** | Temporary pool | Long-running workers |
| **Use Case** | Testing, one-off runs, management | Production workflow processing |

## Ecosystem

 - **Web UI** for visualizing and managing workflows: [Web UI](https://github.com/rom8726/floxy-ui)
 - **Web Manager**: advanced Web UI with SSO, RBAC, etc.: [Floxy Manager](https://github.com/floxy-project/floxy-manager)
 - **GoLand Plugin**: [plugin](https://github.com/rom8726/floxy-goland-plugin)
 - **VS Code Extension**: [extension](https://github.com/rom8726/floxy-vsc-plugin)

## Why Floxy?

**Floxy** is a lightweight, embeddable workflow engine for Go developers.
It was born from the idea that *not every system needs a full-blown workflow platform like Cadence or Temporal.*

### 1. Lightweight, Not Heavyweight

Most workflow engines require you to deploy multiple services, brokers, and databases just to run a single flow.
Floxy is different — it’s a **Go library**.
You import it, initialize an engine, and define your workflow directly in Go code.

No clusters. No queues.

```go
wf, _ := floxy.NewBuilder("order", 1).
    Step("reserve_stock", "stock.Reserve").
    Then("charge_payment", "payment.Charge").
    OnFailure("refund", "payment.Refund").
    Build()
```

That’s it — a complete Saga with compensation.

### 2. Pragmatic by Design

Floxy doesn’t try to solve every problem in distributed systems.
It focuses on **clear, deterministic workflow execution** with the tools Go developers already use:

* PostgreSQL as durable storage
* Go’s standard `net/http` for API
* Structured retries, compensation, and rollback

You don’t need to learn Cadence’s terminology.
Everything is plain Go — just like your codebase.

### 3. Embedded

You can embed Floxy inside any Go service.

> Floxy is for developers who love Go, simplicity, and control.
No orchestration clusters. No external DSLs.
Just workflows — defined in Go, executed anywhere.

## Quick Start

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    
    "github.com/jackc/pgx/v5/pgxpool"
	floxy "github.com/rom8726/floxy-pro"
)

func main() {
    ctx := context.Background()
    
    // Connect to PostgreSQL
    pool, err := pgxpool.New(ctx, "postgres://floxy:password@localhost:5432/floxy?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()
    
    // Run database migrations
    if err := floxy.RunMigrations(ctx, pool); err != nil {
        log.Fatal(err)
    }
    
    // Create engine
    engine := floxy.NewEngine(pool)
	defer engine.Shutdown()
    
    // Register step handlers
    engine.RegisterHandler(&PaymentHandler{})
    engine.RegisterHandler(&InventoryHandler{})
    engine.RegisterHandler(&ShippingHandler{})
    engine.RegisterHandler(&CompensationHandler{})
    
    // Define workflow using Builder DSL
    workflow, err := floxy.NewBuilder("order-processing", 1, floxy.WithDLQEnabled(true)).
        Step("process-payment", "payment", floxy.WithStepMaxRetries(3)).
        OnFailure("refund-payment", "compensation").
        SavePoint("payment-checkpoint").
        Then("reserve-inventory", "inventory", floxy.WithStepMaxRetries(2)).
        OnFailure("release-inventory", "compensation").
        Then("ship-order", "shipping").
        OnFailure("cancel-shipment", "compensation").
        Build()
    if err != nil {
        log.Fatal(err)
    }
    
    // Register and start workflow
    if err := engine.RegisterWorkflow(ctx, workflow); err != nil {
        log.Fatal(err)
    }
    
    order := map[string]any{
        "user_id": "user123",
        "amount":  100.0,
        "items":   []string{"item1", "item2"},
    }
    input, _ := json.Marshal(order)
    
    instanceID, err := engine.Start(ctx, "order-processing-v1", input)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Workflow started: %d", instanceID)
    
    // Process workflow steps
    for {
        empty, err := engine.ExecuteNext(ctx, "worker1")
        if err != nil {
            log.Printf("ExecuteNext error: %v", err)
        }
        if empty {
            break
        }
    }
    
    // Check final status
    status, err := engine.GetStatus(ctx, instanceID)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Workflow status: %s", status)
}

// Step handlers
type PaymentHandler struct{}
func (h *PaymentHandler) Name() string { return "payment" }
func (h *PaymentHandler) Execute(ctx context.Context, stepCtx floxy.StepContext, input json.RawMessage) (json.RawMessage, error) {
    // Process payment logic
    return json.Marshal(map[string]any{"status": "paid"})
}

type InventoryHandler struct{}
func (h *InventoryHandler) Name() string { return "inventory" }
func (h *InventoryHandler) Execute(ctx context.Context, stepCtx floxy.StepContext, input json.RawMessage) (json.RawMessage, error) {
    // Reserve inventory logic
    return json.Marshal(map[string]any{"status": "reserved"})
}

type ShippingHandler struct{}
func (h *ShippingHandler) Name() string { return "shipping" }
func (h *ShippingHandler) Execute(ctx context.Context, stepCtx floxy.StepContext, input json.RawMessage) (json.RawMessage, error) {
    // Ship order logic
    return json.Marshal(map[string]any{"status": "shipped"})
}

type CompensationHandler struct{}
func (h *CompensationHandler) Name() string { return "compensation" }
func (h *CompensationHandler) Execute(ctx context.Context, stepCtx floxy.StepContext, input json.RawMessage) (json.RawMessage, error) {
    // Compensation logic
    return json.Marshal(map[string]any{"status": "compensated"})
}
```

## Examples

See the `examples/` directory for complete workflow examples:

- **Hello World**: Basic single-step workflow
- **E-commerce**: Order processing with compensation flows
- **Data Pipeline**: Parallel data processing with Fork/Join patterns
- **Microservices**: Complex service orchestration with multiple branches
- **SavePoint Demo**: Demonstrates SavePoint functionality and rollback
- **Rollback Demo**: Shows full rollback mechanism with OnFailure handlers
- **Human-in-the-loop**: Interactive workflows with human decision points

Run examples:
```bash
# Start PostgreSQL (required for examples)
make dev-up

# Run all examples
cd examples/hello_world && go run main.go
cd examples/ecommerce && go run main.go
cd examples/data_pipeline && go run main.go
cd examples/microservices && go run main.go
cd examples/savepoint_demo && go run main.go
cd examples/rollback_demo && go run main.go
cd examples/human_in_the_loop__approved && go run main.go
cd examples/human_in_the_loop__rejected && go run main.go
```

## Integration Tests

The library includes comprehensive integration tests using testcontainers:

```bash
# Run all tests
go test ./...

# Run only integration tests
go test -v -run TestIntegration

# Run specific integration test
go test -v -run TestIntegration_DataPipeline
```

Integration tests cover:
- **Data Pipeline**: Parallel data processing with multiple sources
- **E-commerce**: Order processing with success/failure scenarios
- **Microservices**: Complex orchestration with multiple service calls
- **SavePoint Demo**: SavePoint functionality with conditional failures
- **Rollback Demo**: Full rollback mechanism testing
- **Human-in-the-loop**: Make decisions (confirm/reject)

## Database Migrations

The library includes embedded database migrations using `go:embed`. Migrations are automatically applied when using `floxy.RunMigrations()`:

```go
// Run migrations
if err := floxy.RunMigrations(ctx, pool); err != nil {
    log.Fatal(err)
}
```

### Pro Version Migrations

The Pro version uses partitioned tables managed by `pg_partman`. The migrations are located in `migrations_pro/`:

- `001_initial.up.sql`: Initial schema with partitioned tables (`workflow_instances`, `workflow_steps`, `workflow_events`, `workflow_dlq`) using `pg_partman`
- `002_add_savepoint_and_rollback.up.sql`: SavePoint and rollback support
- `003_add_compensation_retry_count.up.sql`: Compensation step status and compensation_retry_count added
- `004_add_compensation_to_views.up.sql`: Active workflows view updated
- `005_add_idempotency_key_to_steps.up.sql`: Idempotency Key added to step table
- `006_add_human_in_the_loop_step.up.sql`: Human-in-the-loop step support and decision tracking
- `007_add_workflow_cancel_requests_table.up.sql`: Cancel requests table
- `008_add_dead_letter_queue.up.sql`: Dead Letter Queue for failed steps
- `009_add_dlq_and_paused_statuses.up.sql`: DLQ and paused statuses support
- `010_add_cleanup_function.up.sql`: Cleanup function for partitioned tables

**Note**: The Pro version requires the `pg_partman` extension to be installed in PostgreSQL. The extension is automatically created in the `partman` schema during migration.

## Dead Letter Queue (DLQ)

### Overview

Floxy supports two different error handling modes:

1. **Classic Saga Mode** (default): When a step fails, the engine performs rollback to the last SavePoint and executes compensation handlers
2. **DLQ Mode**: Rollback is disabled, the workflow pauses in `dlq` state, and failed steps are stored in DLQ for manual investigation

### DLQ Mode Behavior

When DLQ is enabled for a workflow:

- **No Rollback**: Compensation handlers are **not** executed on failure
- **Workflow Paused**: Instance status → `dlq` (not terminal, can be resumed)
- **Active Steps Frozen**: All `running` steps are set to `paused` status
- **Queue Cleared**: Instance queue is cleared to prevent further progress
- **Manual Recovery**: After fixing issues, use `RequeueFromDLQ` to resume

### Enabling DLQ Mode

Enable DLQ for a workflow during definition:

```go
workflow, err := floxy.NewBuilder("payment-processing", 1, floxy.WithDLQEnabled(true)).
    Step("validate-payment", "payment-validator", floxy.WithStepMaxRetries(2)).
    Then("process-payment", "payment-processor", floxy.WithStepMaxRetries(3)).
    Then("notify-user", "notification-service", floxy.WithStepMaxRetries(1)).
    Build()
```

### Fork/Join with DLQ

When using Fork/Join with DLQ enabled:

- **Parallel Branches**: Other branches continue to completion before workflow pauses
- **Join Step**: Created as `paused` when all dependencies are met but the instance is in `dlq` state
- **Requeue Behavior**: After requeuing the failed step, join transitions from `paused` → `pending` for automatic continuation

### RequeueFromDLQ

The `RequeueFromDLQ` method restores a failed step from DLQ and resumes workflow execution:

```go
// Requeue with original input
err := engine.RequeueFromDLQ(ctx, dlqID, nil)

// Requeue with modified input
newInput := json.RawMessage(`{"status": "fixed", "data": "corrected"}`)
err := engine.RequeueFromDLQ(ctx, dlqID, &newInput)
```

**What happens during requeue:**
1. Instance status: `dlq` → `running`
2. Failed step: `paused` → `pending` (retry counters reset)
3. Input updated if `newInput` provided
4. Step enqueued for execution
5. Join steps: `paused` → `pending`
6. DLQ record deleted

### Use Cases for DLQ Mode

- **Manual Data Review**: Steps that require human inspection before retry
- **External Service Outages**: When downstream services are temporarily unavailable
- **Data Quality Issues**: Malformed data requiring manual correction
- **Complex Debugging**: When failures need detailed investigation
- **Business Approval Workflows**: Where failures should pause for review rather than auto-rollback

### Example: Payment Processing with DLQ

```go
// Define workflow with DLQ enabled
workflow, err := floxy.NewBuilder("payment-processing", 1, floxy.WithDLQEnabled(true)).
    Step("validate-payment", "payment-validator", floxy.WithStepMaxRetries(2)).
    Then("process-payment", "payment-processor", floxy.WithStepMaxRetries(3)).
    Then("notify-user", "notification-service", floxy.WithStepMaxRetries(1)).
    Build()

// Register and start workflow
err = engine.RegisterWorkflow(ctx, workflow)
instanceID, err := engine.Start(ctx, "payment-processing-v1", input)

// Process workflow - if step fails, workflow goes to dlq state
for {
    empty, err := engine.ExecuteNext(ctx, "worker1")
    if empty || err != nil {
        break
    }
}

// Later: investigate failure in DLQ, fix issue, then requeue
newInput := json.RawMessage(`{"payment_id": "corrected-id", "amount": 100.0}`)
err = engine.RequeueFromDLQ(ctx, dlqID, &newInput)

// Workflow resumes from where it paused
```

## Known Issues

### Condition Steps in Forked Branches

When using `Condition` steps within `Fork` branches, the `JoinStep` step may not wait for all dynamically created steps (like `else` branches) to complete before considering the workflow finished. This can lead to premature workflow completion.

**Example of problematic case:**
```go
Fork("parallel_branch", func(branch1 *floxy.Builder) {
    branch1.Step("branch1_step1", "handler").
        Condition("branch1_condition", "{{ gt .count 5 }}", func(elseBranch *floxy.Builder) {
            elseBranch.Step("branch1_else", "handler") // This step might not be waited for
        }).
        Then("branch1_next", "handler")
}, func(branch2 *floxy.Builder) {
    branch2.Step("branch2_step1", "handler").
        Condition("branch2_condition", "{{ lt .count 3 }}", func(elseBranch *floxy.Builder) {
            elseBranch.Step("branch2_else", "handler") // This step might not be waited for
        }).
        Then("branch2_next", "handler")
}).
JoinStep("join", []string{"branch1_step1", "branch2_step1"}, floxy.JoinStrategyAll)
```

**SOLVED:** Avoid using `JoinStep` with `Condition`, use `Join` instead that dynamically creates waitFor list (virtual steps conception used).

See `examples/condition/main.go` for a demonstration of this issue.

## Installation

```bash
go get github.com/rom8726/floxy
```

## Dependencies

- PostgreSQL database (with `pg_partman` extension for Pro version)
- Go 1.24+

**Pro Version Requirements:**
- PostgreSQL with `pg_partman` extension installed
- The extension is automatically created during migration, but PostgreSQL must have the extension available
