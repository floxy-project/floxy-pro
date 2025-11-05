# Floxyd - Workflow Server Worker

Floxyd is a daemon service that runs workflow workers with PostgreSQL storage. It loads handlers from a YAML configuration file and processes workflow steps continuously.

## Overview

Floxyd is designed to run as a long-running service that:
- Connects to PostgreSQL database for workflow storage
- Loads handlers and workflow definitions from YAML configuration
- Registers workflow definitions in the database
- Runs multiple workers to process workflow steps
- Supports both bash script handlers and HTTP endpoint handlers
- Provides TLS configuration for secure HTTP handlers

## Installation

Build floxyd from source:

```bash
go build ./cmd/floxyd
```

## Configuration

### Environment Variables

Floxyd requires the following environment variables:

**Database Connection:**
- `FLOXY_DB_HOST` - Database host (required)
- `FLOXY_DB_PORT` - Database port (required)
- `FLOXY_DB_USER` - Database user (required)
- `FLOXY_DB_PASSWORD` - Database password (optional, can be empty)
- `FLOXY_DB_NAME` - Database name (required)

**Worker Configuration:**
- `FLOXY_WORKERS` - Number of workers (default: 3)
- `FLOXY_WORKER_INTERVAL` - Worker polling interval (default: "100ms")

### YAML Configuration File

Floxyd accepts a YAML file as a command-line argument containing handler definitions and workflow definitions. The file should contain `handlers` section (required), `flows` section (optional, for workflow definitions), and optionally `tls` for global TLS settings.

**YAML Structure:**

```yaml
tls:
  skip_verify: false
  cert_file: /path/to/cert.pem
  key_file: /path/to/key.pem
  ca_file: /path/to/ca.pem

handlers:
  - name: handler_name
    exec: handler_execution_string
    tls:
      skip_verify: true
      cert_file: /path/to/client-cert.pem
      key_file: /path/to/client-key.pem
      ca_file: /path/to/ca.pem

flows:
  - name: workflow_name
    steps:
      - name: step1
        handler: handler_name
      - name: step2
        handler: another_handler
```

## Handler Types

### Bash Script Handlers

Bash handlers can be defined in two ways:

**1. Inline Script:**
```yaml
handlers:
  - name: process_data
    exec: |
      echo "$INPUT" | jq '.value * 2'
```

**2. Script File:**
```yaml
handlers:
  - name: process_data
    exec: ./scripts/process.sh
```

Bash handlers:
- Receive JSON input via `$INPUT` environment variable
- Can access input fields as individual environment variables (uppercase)
- Receive workflow context via `FLOXY_*` environment variables
- Must output valid JSON to stdout
- Use `set -e` automatically for inline scripts

**Environment Variables Available to Bash Handlers:**
- `INPUT` - Complete JSON input
- `FLOXY_INSTANCE_ID` - Workflow instance ID
- `FLOXY_STEP_NAME` - Current step name
- `FLOXY_IDEMPOTENCY_KEY` - Idempotency key
- `FLOXY_RETRY_COUNT` - Retry attempt number
- Input field variables (uppercase): `USER_ID`, `DATA_TYPE`, etc.

### HTTP Endpoint Handlers

HTTP handlers are detected automatically when the `exec` field starts with `http://` or `https://`:

```yaml
handlers:
  - name: api_handler
    exec: https://api.example.com/process
    tls:
      skip_verify: true
```

HTTP handlers:
- Send POST requests to the specified URL
- Request body contains:
  ```json
  {
    "metadata": {
      // All variables from StepContext.CloneData()
    },
    "data": {
      // JSON data from previous step
    }
  }
  ```
- Request headers include:
  - `Content-Type: application/json`
  - `X-Floxy-Instance-ID`
  - `X-Floxy-Step-Name`
  - `X-Floxy-Idempotency-Key`
  - `X-Floxy-Retry-Count`
- Must return valid JSON response
- Response body is used as step output

## TLS Configuration

### Global TLS Settings

Global TLS settings apply to all HTTP handlers unless overridden:

```yaml
tls:
  skip_verify: false
  cert_file: /path/to/cert.pem
  key_file: /path/to/key.pem
  ca_file: /path/to/ca.pem
```

### Handler-Specific TLS Settings

Handler-specific TLS settings override global settings:

```yaml
handlers:
  - name: secure_handler
    exec: https://secure-api.example.com/process
    tls:
      skip_verify: false
      cert_file: /path/to/client-cert.pem
      key_file: /path/to/client-key.pem
      ca_file: /path/to/ca.pem
```

**TLS Configuration Options:**
- `skip_verify` - Skip certificate verification (insecure mode)
- `cert_file` - Client certificate file path
- `key_file` - Client private key file path
- `ca_file` - CA certificate file path

## Usage

### Basic Usage

```bash
export FLOXY_DB_HOST=localhost
export FLOXY_DB_PORT=5432
export FLOXY_DB_USER=floxy
export FLOXY_DB_PASSWORD=password
export FLOXY_DB_NAME=floxy
export FLOXY_WORKERS=5
export FLOXY_WORKER_INTERVAL=100ms

./floxyd handlers.yaml
```

### Complete Example

**1. Create handlers.yaml:**

```yaml
tls:
  skip_verify: false

handlers:
  - name: bash_process
    exec: |
      echo "$INPUT" | jq '.value * 2'
  
  - name: script_process
    exec: ./scripts/process.sh
  
  - name: http_handler
    exec: https://api.example.com/process
    tls:
      skip_verify: true
  
  - name: secure_http_handler
    exec: https://secure-api.example.com/process
    tls:
      cert_file: /path/to/client-cert.pem
      key_file: /path/to/client-key.pem
      ca_file: /path/to/ca.pem

flows:
  - name: my_workflow
    steps:
      - name: step1
        handler: bash_process
      - name: step2
        handler: http_handler
```

**2. Run floxyd:**

```bash
./floxyd handlers.yaml
```

**3. Start workflow instances:**

Use `floxyctl start` to create workflow instances that will be processed by floxyd workers:

```bash
floxyctl start -o workflow-definition-id \
  --host localhost --port 5432 --user floxy --database floxy
```

## Workflow Registration

On startup, floxyd:
1. Parses YAML file for workflow definitions (`flows` section)
2. Registers each workflow definition in the database
3. Uses version 1 by default for all workflows
4. Logs each registered workflow with its ID and version

Workflow definitions are stored in the `workflows.workflow_definitions` table. If a workflow with the same name and version already exists, it will be updated.

## Workflow Processing

Floxyd workers continuously poll the database for pending workflow steps and execute them:

1. Worker dequeues a step from the queue
2. Finds the registered handler by name
3. Executes the handler (bash script or HTTP request)
4. Updates step status based on result
5. Enqueues next steps if successful

## Graceful Shutdown

Floxyd handles shutdown signals gracefully:
- `SIGINT` (Ctrl+C)
- `SIGTERM`

On shutdown:
- Stops all workers
- Waits for current step executions to complete
- Closes database connections
- Exits cleanly

## Logging

Floxyd logs:
- Handler registration messages
- Worker start/stop events
- Worker errors during step processing
- Shutdown messages

## Differences from floxyctl

| Feature | floxyctl | floxyd |
|---------|----------|--------|
| **Mode** | CLI tool | Daemon service |
| **Storage** | In-memory (YAML mode) or PostgreSQL | PostgreSQL only |
| **Execution** | Runs workflow to completion | Continuous processing |
| **Workers** | Temporary pool | Long-running workers |
| **Handlers** | Bash scripts only | Bash scripts + HTTP endpoints |

## Troubleshooting

### Handler not found

Ensure the handler name in your workflow definition matches the handler name in the YAML file.

### HTTP handler errors

- Check TLS configuration (certificates, skip_verify)
- Verify endpoint URL is accessible
- Ensure endpoint returns valid JSON
- Check request format (metadata + data fields)

### Database connection errors

- Verify environment variables are set correctly
- Check database is running and accessible
- Verify database schema exists (migrations run automatically)

### Worker not processing steps

- Check worker count (`FLOXY_WORKERS`)
- Verify workflow instances are created in database
- Check database connection is active
- Review logs for errors

## Example HTTP Endpoint Handler Implementation

Your HTTP endpoint should:

1. Accept POST requests with JSON body:
```json
{
  "metadata": {
    "instance_id": 123,
    "step_name": "process",
    ...
  },
  "data": {
    "input": "from previous step"
  }
}
```

2. Return JSON response:
```json
{
  "result": "processed data"
}
```

3. Handle errors by returning appropriate HTTP status codes (4xx/5xx)

## See Also

- [Floxyctl Documentation](../docs/floxyctl.md) - CLI tool for workflow management
- [Engine Specification](../docs/ENGINE_SPEC.md) - Core engine documentation

