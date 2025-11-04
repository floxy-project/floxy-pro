# Simple Floxyctl Example

This example demonstrates how to use `floxyctl` to run a simple workflow with shell handlers.

## Prerequisites

- `floxyctl` binary built
- `jq` installed (for JSON processing)

## Files

- `workflow.yaml` - Workflow definition with handlers and steps
- `input.json` - Initial input data for the workflow

## Usage

Run the workflow:

```bash
floxyctl run -f workflow.yaml -i input.json
```

Or with custom parameters:

```bash
floxyctl run -f workflow.yaml -i input.json --workers 2 --completion-timeout 5m
```

## Workflow Description

The workflow consists of three steps:

1. **process** - Multiplies the value by 2 using `jq`
2. **validate** - Validates if the value is greater than 100
3. **final** - Finalizes the workflow and marks it as completed

Each handler receives JSON input via `$INPUT` environment variable and outputs JSON to stdout.

## Handler Details

Handlers use shell scripts with `jq` to process JSON:
- Input is passed via `$INPUT` environment variable
- Output must be valid JSON printed to stdout
- The output of each step becomes the input for the next step

