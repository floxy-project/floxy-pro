# Advanced Floxyctl Example - Parallel and Condition Steps

This example demonstrates advanced workflow features including:
- **Condition steps** for branching logic
- **Parallel steps** for concurrent execution
- Complex workflow orchestration with conditional and parallel execution paths

## Prerequisites

- `floxyctl` binary built
- `jq` installed (for JSON processing)
- Scripts must be executable: `chmod +x scripts/*.sh`

## Files

- `workflow.yaml` - Workflow definition with condition and parallel steps
- `input_heavy.json` - Input for heavy data processing path
- `input_light.json` - Input for light data processing path
- `scripts/` - Handler scripts for all workflow steps

## Setup

Make scripts executable:

```bash
chmod +x scripts/*.sh
```

## Usage

### Heavy Data Processing Path

```bash
floxyctl run -f workflow.yaml -i input_heavy.json
```

With debug mode:

```bash
floxyctl run -f workflow.yaml -i input_heavy.json --debug
```

### Light Data Processing Path

```bash
floxyctl run -f workflow.yaml -i input_light.json
```

### Custom Parameters

```bash
floxyctl run -f workflow.yaml -i input_heavy.json --workers 5 --completion-timeout 5m --debug
```

## Workflow Description

The workflow demonstrates a data processing pipeline with conditional and parallel execution:

### Step Flow

1. **validate** - Validates the incoming request
   - Checks for required `user_id`
   - Marks request as validated

2. **check_data_type** (Condition) - Determines processing path
   - **Condition**: `input.data_type == 'heavy'`
   - **If true**: Proceeds to parallel processing
   - **If false**: Executes `process_light` step, then continues

3. **parallel_processing** (Parallel) - Executes three tasks concurrently
   - **process_heavy**: Heavy data processing (simulated with sleep)
   - **notify_email**: Sends email notification
   - **notify_sms**: Sends SMS notification
   - All three tasks run in parallel and wait for completion at join

4. **update_db** - Updates database with results
   - Runs after parallel processing completes

5. **check_generate_report** (Condition) - Decides whether to generate report
   - **Condition**: `input.generate_report == true`
   - **If true**: Proceeds to generate report
   - **If false**: Executes `skip_report` (cleanup) step, then continues

6. **generate** - Generates final report
   - Creates report ID
   - Finalizes workflow

## Execution Paths

### Path 1: Heavy Data Processing (`input_heavy.json`)

```
validate → check_data_type (true) → parallel_processing
                                           ├─ process_heavy
                                           ├─ notify_email
                                           └─ notify_sms
         → update_db → check_generate_report (true) → generate
```

**Steps executed:**
1. validate
2. check_data_type (condition: true)
3. parallel_processing:
   - process_heavy
   - notify_email
   - notify_sms
4. update_db
5. check_generate_report (condition: true)
6. generate

### Path 2: Light Data Processing (`input_light.json`)

```
validate → check_data_type (false) → process_light → parallel_processing
                                                           ├─ process_heavy
                                                           ├─ notify_email
                                                           └─ notify_sms
         → update_db → check_generate_report (false) → skip_report → generate
```

**Steps executed:**
1. validate
2. check_data_type (condition: false)
3. process_light
4. parallel_processing:
   - process_heavy
   - notify_email
   - notify_sms
5. update_db
6. check_generate_report (condition: false)
7. skip_report
8. generate

## Condition Steps

Condition steps evaluate expressions against the input data using Go template syntax:

```yaml
- type: condition
  name: check_data_type
  expr: "{{ eq .data_type \"heavy\" }}"
  else:
    - name: process_light
      handler: process_data_light
```

- **expr**: Go template expression that evaluates to `true` or `false`
  - Use `{{ eq .field "value" }}` for equality comparison
  - Use `{{ gt .field 10 }}` for greater than
  - Use `{{ lt .field 10 }}` for less than
  - Access input fields with `.field` (not `input.field`)
- **else**: Optional branch executed when condition is false
- If condition is true, execution continues to next steps
- If condition is false, `else` branch executes first, then continues

**Available comparison functions:**
- `eq` - equals
- `ne` - not equals
- `gt` - greater than
- `lt` - less than
- `ge` - greater than or equal
- `le` - less than or equal

## Parallel Steps

Parallel steps execute multiple tasks concurrently:

```yaml
- type: parallel
  name: parallel_processing
  tasks:
    - name: process_heavy
      handler: process_data_heavy
    - name: notify_email
      handler: send_email
    - name: notify_sms
      handler: send_sms
```

- All tasks in `tasks` array execute concurrently
- A join step is automatically created after parallel execution
- Workflow waits for all tasks to complete before proceeding
- If any task fails, the workflow can be configured to handle failures

## Handler Details

All handlers:
- Receive JSON input via `$INPUT` environment variable
- Can access input fields as individual environment variables
- Must output valid JSON to stdout
- Support debug mode via `$FLOXY_DEBUG` environment variable
- Use `set -e` for error handling

## Environment Variables

Each handler receives:

- `INPUT`: Complete JSON input
- `FLOXY_INSTANCE_ID`: Workflow instance ID
- `FLOXY_STEP_NAME`: Current step name
- `FLOXY_IDEMPOTENCY_KEY`: Idempotency key
- `FLOXY_RETRY_COUNT`: Retry attempt number
- `FLOXY_DEBUG`: Debug mode flag
- Input field variables (uppercase): `USER_ID`, `DATA_TYPE`, `EMAIL`, etc.

## Debug Mode

Enable debug mode to see:

- Handler input/output for each step
- Condition evaluation results
- Parallel execution progress
- Step execution details

Example output:

```
[DEBUG] Handler 'validate' input: {"user_id":"USER-001","data_type":"heavy"}
[DEBUG] Handler 'validate' output: {"validated":true,"step":"validate"}
[DEBUG] Handler 'process_heavy' input: {...}
[DEBUG] Handler 'process_heavy' output: {...}
[DEBUG] Handler 'notify_email' input: {...}
[DEBUG] Handler 'notify_email' output: {...}
```

## Expected Results

### Heavy Path Output

When using `input_heavy.json`:
- All parallel steps execute concurrently
- Report is generated
- Final output includes all processing results

### Light Path Output

When using `input_light.json`:
- Light processing step executes first
- Parallel steps execute (note: process_heavy still runs even for light data)
- Report generation is skipped, cleanup runs instead
- Final output includes processing results

## Notes

- Parallel steps create an automatic join point
- Condition steps evaluate expressions using Go template syntax
- All steps in parallel block execute even if some are conditional
- The `process_heavy` step in parallel block always runs regardless of condition result
- Input data flows through the workflow, with each step potentially modifying it

## Troubleshooting

### Condition not evaluating correctly

- Check expression syntax (Go template format)
- Verify input data structure matches expression
- Use `--debug` to inspect handler inputs

### Parallel steps not completing

- Ensure all parallel tasks output valid JSON
- Check that all tasks complete successfully
- Review step execution logs

### Join step issues

- Verify all parallel tasks are defined correctly
- Check that tasks reference valid handlers
- Ensure handlers are registered before workflow execution

