<!-- Parent: ../../AGENTS.md -->

# laputa/governance/scheduler — Scheduler Daemon & Wakeup Service

**Generated:** 2026-08-01  
**Purpose:** Background daemon for scheduled tasks (rhythm reports, cleanup)

---

## Purpose

The `scheduler/` package provides background scheduling:

- **Scheduler daemon** — run tasks on cron-like schedules
- **Rhythm scheduling** — daily/weekly/monthly report generation
- **Cleanup tasks** — purge expired sessions, audit log rotation
- **Graceful shutdown** — stop scheduler cleanly without orphaning goroutines
- **Task result tracking** — log success/failure of scheduled tasks

---

## Structure

```
scheduler/
├── daemon.go         # Scheduler daemon implementation
├── daemon_test.go
└── (supporting code)
```

---

## Key Concepts

### Scheduler Task

```go
type Task struct {
    Name     string
    Schedule string        // cron format or interval (e.g., "0 0 * * *" for daily)
    Handler  func(context.Context) error
}
```

### Daemon

Runs tasks on schedule:
- **Start(ctx)** — start scheduler, begin running tasks
- **Add(task)** — register a task
- **Stop(ctx)** — graceful shutdown, wait for in-flight tasks
- **Health()** — report daemon health (running, last task status)

### Built-in Tasks

**Daily Rhythm Report** — 00:00 UTC daily
**Weekly Rhythm Report** — Sunday 00:00 UTC
**Monthly Rhythm Report** — 1st of month 00:00 UTC
**Audit log rotation** — daily at midnight

---

## Testing

```bash
cd laputa
GOSUMDB=off go test -v ./governance/scheduler/...
```

**Behavioral tests:**

- Tasks run on correct schedule
- Graceful shutdown waits for in-flight tasks
- Failed tasks log error but don't crash scheduler
- Health reports accurate status
- Multiple tasks don't interfere

---

## Conventions

- All schedules are UTC
- Task timeout: 5 minutes (configurable)
- Failed tasks are retried next interval
- Results logged to governance audit trail

---

## MANUAL

Keep scheduler focused on execution timing. Task implementations go to rhythm or other packages.

Parent reference: ../AGENTS.md
