# Floxyctl - Command Line Tool for Running Workflows

`floxyctl` is a CLI tool for DevOps engineers to run workflows from YAML files. It can execute workflows autonomously using in-memory storage, making it perfect for testing and development.

## Installation

Build the floxyctl binary:

```bash
go build -o floxyctl ./cmd/floxyctl
```

## Usage

### Basic Command

```bash
floxyctl run -f workflow.yaml -i input.json
```

### Command Flags

- `-f, --file` (required): YAML file with workflow configuration
- `-i, --input` (optional): JSON file with initial input data
- `-w, --workers` (default: 3): Number of worker pool workers
- `--worker-interval` (default: 100ms): Worker pool polling interval (e.g., 100ms, 1s, 500ms)
- `--completion-timeout` (default: 10m): Timeout for workflow completion (e.g., 10m, 30m, 1h)
- `--status-check-interval` (default: 500ms): Interval for checking workflow status (e.g., 500ms, 1s)
- `-D, --debug` (default: false): Enable debug mode (prints handler input/output to stderr)

### Examples

Run workflow with input file:

```bash
floxyctl run -f workflow.yaml -i input.json
```

Run workflow with input from stdin:

```bash
echo '{"key": "value"}' | floxyctl run -f workflow.yaml
```

Run workflow with empty input:

```bash
floxyctl run -f workflow.yaml
```

Run workflow with custom parameters:

```bash
floxyctl run -f workflow.yaml -i input.json --workers 5 --worker-interval 200ms --completion-timeout 1h
```

Run workflow with debug mode:

```bash
floxyctl run -f workflow.yaml -i input.json --debug
```

## Workflow YAML Format

### Handler Definition

Handlers can be defined as:
1. **Inline scripts** (multi-line YAML strings)
2. **External script files** (paths to executable shell scripts)

```yaml
handlers:
  - name: my_handler
    exec: |
      echo "$INPUT" | jq '.result = "success"'
  
  - name: file_handler
    exec: ./scripts/my_script.sh
```

### Workflow Definition

```yaml
flows:
  - name: my_workflow
    steps:
      - name: step1
        handler: my_handler
        on_failure: compensation_handler
        max_retries: 3
```

For more details on YAML format, see the existing `yaml_parser.go` implementation.

## Environment Variables

Floxyctl automatically sets the following environment variables for each handler execution:

### Input Data

- `INPUT`: Complete JSON input data as a string (always available)

### Workflow Context Variables

- `FLOXY_INSTANCE_ID`: Workflow instance ID (int64)
- `FLOXY_STEP_NAME`: Current step name (string)
- `FLOXY_IDEMPOTENCY_KEY`: Step idempotency key (string, UUID format)
- `FLOXY_RETRY_COUNT`: Current retry attempt number (int)
- `FLOXY_DEBUG`: Debug mode flag (boolean as string: "true" or "false")

### Input Field Variables

All fields from the input JSON are also available as individual environment variables with uppercase names. For example, if input contains:

```json
{
  "customer_id": "CUST-001",
  "amount": 100.50
}
```

The following environment variables will be available:

- `CUSTOMER_ID=CUST-001`
- `AMOUNT=100.50`

Note: Non-string values are converted to strings using `fmt.Sprintf("%v", value)`.

### System Environment

All system environment variables are also passed through to the handler scripts.

## Handler Requirements

### Input

Handlers receive JSON input via:
- `$INPUT` environment variable (complete JSON string)
- Individual environment variables for each input field

### Output

Handlers **must** output valid JSON to stdout. The output becomes the input for the next step.

**Valid output examples:**

```bash
echo '{"status": "success", "result": 123}'
```

```bash
echo "$INPUT" | jq '.processed = true'
```

**Invalid output (will cause error):**

```bash
echo "some text"
# Error: script output is not valid JSON
```

### Error Handling

Handlers should:
- Use `set -e` in scripts to exit on any command failure
- Return exit code 0 on success
- Return non-zero exit code on failure
- Output error messages to stderr (for debugging)

**Example error handling:**

```bash
#!/bin/bash
set -e

if ! command -v jq >/dev/null 2>&1; then
  echo "Error: jq is required" >&2
  exit 1
fi

echo "$INPUT" | jq '.result = "success"'
```

## Error Handling and Rollback

When a handler fails:

1. **Error Detection**: If a script exits with non-zero code or produces invalid JSON, the step fails
2. **Retry Logic**: If `max_retries` is set and not exceeded, the step will be retried
3. **Compensation**: If `on_failure` handler is defined, it will be executed as compensation
4. **Rollback**: The engine automatically rolls back completed steps in reverse order, executing their compensation handlers
5. **Workflow Status**: The workflow status is set to `failed`

### Compensation Handler Behavior

Compensation handlers:
- Receive the same input as the failed step
- Are executed automatically by the engine
- Should perform cleanup or rollback operations
- Must output valid JSON (same as regular handlers)

## Debug Mode

When `--debug` flag is enabled:

1. **Handler Input**: Each handler's input JSON is printed to stderr before execution
2. **Handler Output**: Each handler's output JSON is printed to stderr after execution
3. **Handler Stderr**: If a handler fails, its stderr output is printed to stderr
4. **Script Debug**: Handlers can check `$FLOXY_DEBUG` environment variable to enable debug output within scripts

**Example debug output:**

```
[DEBUG] Handler 'create_order' input: {"customer_id":"CUST-001","amount":100.50}
[DEBUG] Handler 'create_order' output: {"order_id":"ORD-12345","order_created":true}
```

## Workflow Status

The workflow can end with the following statuses:

- `completed`: All steps executed successfully
- `failed`: One or more steps failed and rollback completed
- `cancelled`: Workflow was cancelled
- `aborted`: Workflow was aborted

## Examples

See the following example directories:

- `examples/floxyctl_simple/`: Simple workflow with inline handlers
- `examples/floxyctl_file_handlers/`: Workflow using external script files
- `examples/floxyctl_rollback/`: Workflow demonstrating rollback with compensation handlers

## Troubleshooting

### Handler produces no output

**Error**: `script produced no output`

**Solution**: Ensure your script outputs valid JSON to stdout. Use `echo` or `jq` to output JSON.

### Handler output is not valid JSON

**Error**: `script output is not valid JSON`

**Solution**: Ensure all output is valid JSON. If using shell commands, pipe through `jq` or output JSON directly.

### Script execution failed

**Error**: `script execution failed with exit code N`

**Solution**: 
- Check that all commands in the script exist
- Verify script file permissions (`chmod +x script.sh`)
- Review stderr output for detailed error messages
- Use `--debug` flag to see detailed execution information

### Workflow completes but shouldn't

If a workflow completes with `completed` status when it should fail:

1. Check that scripts use `set -e` or handle errors explicitly
2. Verify that failed commands return non-zero exit codes
3. Ensure error messages go to stderr, not stdout
4. Use `--debug` to inspect handler execution

