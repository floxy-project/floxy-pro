# File Handlers Example

This example demonstrates how to use `floxyctl` with external script files as handlers.

## Prerequisites

- `floxyctl` binary built
- `jq` installed (for JSON processing)
- Scripts must be executable: `chmod +x scripts/*.sh`

## Files

- `workflow.yaml` - Workflow definition with handlers pointing to script files
- `input.json` - Initial input data for the workflow
- `scripts/transform.sh` - Handler script that transforms data
- `scripts/filter.sh` - Handler script that filters data
- `scripts/aggregate.sh` - Handler script that aggregates data

## Setup

Make scripts executable:

```bash
chmod +x scripts/*.sh
```

## Usage

Run the workflow:

```bash
floxyctl run -f workflow.yaml -i input.json
```

Run with debug mode to see handler input/output:

```bash
floxyctl run -f workflow.yaml -i input.json --debug
```

Or with custom parameters:

```bash
floxyctl run -f workflow.yaml -i input.json --debug --workers 2 --completion-timeout 5m
```

## Workflow Description

The workflow consists of three steps:

1. **transform** - Multiplies the value by 3 and marks as transformed
2. **filter** - Filters data based on value (if > 50)
3. **aggregate** - Aggregates and completes the workflow

## Debug Mode

When `--debug` flag is enabled:
- Each handler prints its input to stderr before processing
- Each handler prints its output to stderr after processing
- Debug output is prefixed with `[DEBUG]` and handler name

## Handler Details

Handlers are external shell scripts:
- Scripts receive JSON input via `$INPUT` environment variable
- Scripts must output valid JSON to stdout
- Scripts can check `$FLOXY_DEBUG` environment variable for debug mode
- Scripts have access to workflow context variables:
  - `FLOXY_INSTANCE_ID`
  - `FLOXY_STEP_NAME`
  - `FLOXY_IDEMPOTENCY_KEY`
  - `FLOXY_RETRY_COUNT`

