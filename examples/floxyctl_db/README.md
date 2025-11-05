# Floxyctl Database Mode Example

This example demonstrates how to use `floxyctl` commands for managing workflows stored in PostgreSQL database.

## Prerequisites

- PostgreSQL database running
- `floxyctl` binary built
- Database connection parameters (host, port, user, password, database name)

## Workflow Overview

This example uses the `hello-world-v1` workflow definition with a simple handler that greets a name.

## Setup

### 1. Register Workflow Definition in Database

First, you need to register the workflow definition in the database. Run the setup program:

```bash
cd examples/floxyctl_db
go run setup.go
```

This will:
- Connect to PostgreSQL database
- Run migrations
- Register the `hello-world-v1` workflow definition
- Register the `hello` handler (for demonstration, you'll need a worker process running)

**Note:** The setup program uses default connection parameters. Modify them in `setup.go` if needed:
- Host: `localhost`
- Port: `5435`
- User: `floxy`
- Password: `password`
- Database: `floxy`

### 2. Start Worker Process (Required)

For workflows to execute, you need a worker process running. In a separate terminal:

```bash
cd examples/floxyctl_db
go run worker.go
```

This worker will:
- Connect to the database
- Process workflow steps
- Execute handlers

## Usage

### Start Workflow Instance

Start a new workflow instance with input:

```bash
floxyctl start -o hello-world-v1 \
  --host localhost \
  --port 5435 \
  --user floxy \
  --database floxy \
  -W
```

Or with input file:

```bash
floxyctl start -o hello-world-v1 \
  -i input.json \
  --host localhost \
  --port 5435 \
  --user floxy \
  --database floxy \
  -W
```

Or with input from stdin:

```bash
echo '{"name": "Floxy"}' | floxyctl start -o hello-world-v1 \
  --host localhost \
  --port 5435 \
  --user floxy \
  --database floxy \
  -W
```

The command will output:
```
Workflow instance started with ID: 123
```

### Cancel Workflow Instance

Cancel a running workflow instance with rollback (compensation):

```bash
floxyctl cancel -o 123 \
  --host localhost \
  --port 5435 \
  --user floxy \
  --database floxy \
  -W
```

With custom reason:

```bash
floxyctl cancel -o 123 \
  --host localhost \
  --port 5435 \
  --user floxy \
  --database floxy \
  -W \
  --reason "User requested cancellation"
```

### Abort Workflow Instance

Abort a running workflow instance without rollback:

```bash
floxyctl abort -o 123 \
  --host localhost \
  --port 5435 \
  --user floxy \
  --database floxy \
  -W
```

With custom reason:

```bash
floxyctl abort -o 123 \
  --host localhost \
  --port 5435 \
  --user floxy \
  --database floxy \
  -W \
  --reason "Critical error detected"
```

## Workflow Definition

The `hello-world-v1` workflow consists of:
- **Step**: `say-hello`
- **Handler**: `hello`
- **Max Retries**: 3

## Input Format

The workflow accepts JSON input:

```json
{
  "name": "Floxy"
}
```

If no name is provided, it defaults to "World".

## Complete Example Workflow

1. **Setup** (one time):
   ```bash
   go run setup.go
   ```

2. **Start worker** (in separate terminal):
   ```bash
   go run worker.go
   ```

3. **Start workflow instance**:
   ```bash
   floxyctl start -o hello-world-v1 -i input.json \
     --host localhost --port 5435 --user floxy --database floxy -W
   ```
   Note the instance ID from output (e.g., `123`)

4. **Cancel workflow** (if needed):
   ```bash
   floxyctl cancel -o 123 \
     --host localhost --port 5435 --user floxy --database floxy -W
   ```

5. **Or abort workflow** (if needed):
   ```bash
   floxyctl abort -o 123 \
     --host localhost --port 5435 --user floxy --database floxy -W
   ```

## Differences from YAML Mode

- **YAML mode** (`floxyctl run`): Uses in-memory store, runs workflow to completion synchronously
- **Database mode** (`floxyctl start/cancel/abort`): Uses PostgreSQL, requires separate worker process, returns immediately after starting

## Notes

- The `-W` flag prompts for password (masked input)
- Workflow instances are stored in PostgreSQL and persist across CLI sessions
- Multiple workers can process steps from the same database
- Cancel performs rollback (compensation), abort stops immediately without compensation

