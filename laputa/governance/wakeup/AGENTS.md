<!-- Parent: ../../AGENTS.md -->

# laputa/governance/wakeup — Wakeup Event Provider

**Generated:** 2026-08-01  
**Purpose:** Detect activity and trigger governance updates or recalls

---

## Purpose

The `wakeup/` package provides event notifications for governance state changes:

- **Activity detection** — watch for changes in watched sections
- **Event emission** — publish events when changes detected
- **Subscriber support** — allow other services to listen for wakeups
- **Backpressure handling** — queue events, drop old events if queue full
- **Graceful shutdown** — close event channels cleanly

---

## Structure

```
wakeup/
├── provider.go       # Wakeup provider service
└── provider_test.go
```

---

## Key Concepts

### Wakeup Event

```go
type WakeupEvent struct {
    Section   string
    Kind      string        // "updated", "created", "deleted"
    Timestamp time.Time
    Data      map[string]any
}
```

### Provider

```go
type Provider struct {
    Events chan WakeupEvent
}

func (p *Provider) Watch(ctx context.Context, section string) error
func (p *Provider) Stop(ctx context.Context) error
```

### Usage

**Subscribe:**
```go
provider := wakeup.New()
go provider.Watch(ctx, "01-identity")

for event := range provider.Events {
    log.Printf("Section %s updated: %v", event.Section, event.Data)
}
```

---

## Testing

```bash
cd laputa
GOSUMDB=off go test -v ./governance/wakeup/...
```

**Behavioral tests:**

- Events are emitted on section change
- Multiple subscribers don't interfere
- Graceful shutdown closes channels
- Backpressure doesn't block watchers

---

## Conventions

- Event channel has buffer size 100
- Old events are dropped if buffer full
- All events are timestamped
- No ordering guarantees across sections

---

## MANUAL

Keep wakeup focused on event notification. Event handling logic goes to consumers.

Parent reference: ../AGENTS.md
