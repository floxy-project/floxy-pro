# Rollback Example with On Failure Handlers

This example demonstrates how to use `on_failure` handlers (compensation handlers) for workflow rollback in floxyctl.

## Prerequisites

- `floxyctl` binary built
- `jq` installed (for JSON processing)
- Scripts must be executable: `chmod +x scripts/*.sh`

## Files

- `workflow.yaml` - Workflow definition with compensation handlers
- `input.json` - Normal input that should succeed
- `input_fail.json` - Input that triggers failure in charge_payment step
- `scripts/create_order.sh` - Creates an order
- `scripts/charge_payment.sh` - Charges payment (can fail if `fail_charge` is true)
- `scripts/send_notification.sh` - Sends notification
- `scripts/refund_payment.sh` - Compensation handler for charge_payment
- `scripts/cancel_order.sh` - Compensation handler for create_order

## Setup

Make scripts executable:

```bash
chmod +x scripts/*.sh
```

## Usage

### Successful workflow execution:

```bash
floxyctl run -f workflow.yaml -i input.json
```

### Failed workflow execution (triggers rollback):

```bash
floxyctl run -f workflow.yaml -i input_fail.json
```

With debug mode:

```bash
floxyctl run -f workflow.yaml -i input_fail.json --debug
```

## Workflow Description

The workflow consists of three main steps:

1. **create** - Creates an order
   - Compensation: `cancel_order` (runs if create fails)

2. **charge** - Charges payment
   - Compensation: `refund_payment` (runs if charge fails)
   - Can be forced to fail by setting `fail_charge: true` in input

3. **notify** - Sends notification
   - No compensation handler

## Rollback Behavior

When a step fails:

1. The engine automatically triggers the `on_failure` handler (compensation)
2. Compensation handlers execute in reverse order of completed steps
3. If `charge` fails after `create` succeeded:
   - `refund_payment` is executed (compensation for charge)
   - `cancel_order` is executed (compensation for create)
4. The workflow status is set to `failed`

## Example Scenarios

### Scenario 1: Successful execution
- All steps complete successfully
- No compensation handlers are executed
- Workflow status: `completed`

### Scenario 2: Payment charge fails
- `create` succeeds
- `charge` fails (due to `fail_charge: true`)
- `refund_payment` is executed (compensation for charge)
- `cancel_order` is executed (compensation for create)
- Workflow status: `failed`

## Compensation Handler Details

Compensation handlers:
- Receive the same input as the failed step
- Must output valid JSON
- Are executed automatically by the engine
- Can access step context via environment variables:
  - `FLOXY_INSTANCE_ID`
  - `FLOXY_STEP_NAME`
  - `FLOXY_IDEMPOTENCY_KEY`
  - `FLOXY_RETRY_COUNT`
