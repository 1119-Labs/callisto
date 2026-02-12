# PR: feat: add optional dead-letter queue retry workers with configurable min-age delay

## Commit Message

```
feat: add DLQ retry workers with configurable min-age delay
```

## Description

Adds optional dead-letter queue (DLQ) retry workers that can be toggled on/off via config. When enabled, these workers consume failed messages from the `new-block-dlq` and `old-block-dlq` queues and force-retry them.

A configurable **minimum age** (`dlq_min_age`, default `5m`) ensures messages sit in the DLQ for at least that duration before being retried, preventing immediate retry loops on transient failures.

## Changes

### New Config Options (`lib/parser/config/config.go`)
| Field | YAML Key | Default | Description |
|-------|----------|---------|-------------|
| `ParseDLQ` | `parse_dead_letter_queue` | `false` | Enable/disable DLQ workers |
| `DLQWorkers` | `dlq_workers` | `1` | Number of workers per DLQ queue |
| `DLQMinAge` | `dlq_min_age` | `5m` | Minimum time a message must age in DLQ before retry |

### Queue Layer (`lib/queue/rabbitmq.go`)
- Added `minAge` field to `RabbitMQHeightQueue` struct
- Added `connectDLQueue()` — connects to an existing DLQ (prefetch=1, no queue creation)
- Added `ConnectNewBlockDLQ()` / `ConnectOldBlockDLQ()` public constructors
- Added `waitForMinAge()` — reads the `x-failed-at` header and sleeps until the message has aged past `dlq_min_age`
- `Consume()` now calls `waitForMinAge()` before processing when `minAge > 0`

### Worker (`lib/parser/block_worker.go`)
- Added `StartDLQ()` method — same as `Start()` but uses `Process()` (force-retry) instead of `ProcessIfNotExists()` (skip-if-exists)

### Pipeline Wiring (`lib/cmd/start/cmd.go`)
- Added DLQ pipeline section gated behind `cfg.ParseDLQ`
- Creates workers for both `new-block-dlq` and `old-block-dlq`
- Starts DLQ workers as goroutines alongside existing pipelines
- All DLQ connections tracked for graceful shutdown

### Config (`callisto-config/config.yaml`)
- Added `parse_dead_letter_queue`, `dlq_workers`, `dlq_min_age` under `parsing:`

## Usage

```yaml
parsing:
  parse_dead_letter_queue: true   # enable DLQ retry
  dlq_workers: 1                  # workers per DLQ queue
  dlq_min_age: 5m                 # wait at least 5 min before retrying
```

Set `parse_dead_letter_queue: false` and restart to disable.
