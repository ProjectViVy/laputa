<!-- Parent: ../AGENTS.md -->

# laputa/scripts — Utility Scripts and Launchers

**Generated:** 2026-08-01  
**Purpose:** Setup, launch, and supervision utilities for Laputa governance

---

## Purpose

The `scripts/` directory contains shell and batch scripts for local development, deployment, and operational tasks:

- **Launcher scripts** — start Laputa services and supervisors
- **Setup utilities** — initialize governance sections and dependencies
- **Development helpers** — build, test, and diagnostic tools

---

## Structure

```
scripts/
├── laputa-launcher.bat              # Windows batch launcher
├── laputa-launcher.py               # Cross-platform Python launcher
├── laputa-supervisor.cmd            # Windows supervisor for long-running processes
├── one-shot-mempalace-up.ps1        # PowerShell one-shot initialization
└── (additional utilities as needed)
```

---

## Scripts

### laputa-launcher.bat

Windows batch script for launching Laputa services.

**Usage:**

```batch
laputa-launcher.bat [service-name]
```

Supported services:
- `governance` — start governance service
- `rhythm` — start rhythm reporting engine
- `all` — start all services

### laputa-launcher.py

Cross-platform Python launcher for Laputa services. Works on Windows, macOS, and Linux.

**Usage:**

```bash
python laputa-launcher.py --service governance --config ~/.laputa/config.yaml
python laputa-launcher.py --service rhythm --kind daily
python laputa-launcher.py --all
```

**Options:**

- `--service` — service to launch (governance, rhythm, etc.)
- `--kind` — for rhythm: daily, weekly, or monthly
- `--config` — path to config file
- `--all` — launch all services
- `--background` — run as background process

### laputa-supervisor.cmd

Windows command script for supervising long-running processes with restart policy.

**Usage:**

```cmd
laputa-supervisor.cmd start
laputa-supervisor.cmd stop
laputa-supervisor.cmd restart
laputa-supervisor.cmd status
```

Features:
- Auto-restart on failure
- Process health checks
- Log rotation
- Graceful shutdown

### one-shot-mempalace-up.ps1

PowerShell script for quick initialization of a complete MemPalace environment (Laputa + Mentle).

**Usage:**

```powershell
.\one-shot-mempalace-up.ps1 -Path ~/.my-palace -ConfigPath ~/.laputa/config.yaml
```

Sets up:
1. Laputa governance sections
2. Mentle palace storage
3. Links and integrations
4. Permissions and authority

---

## Development Use

### Building from Scripts

Scripts may invoke:

```bash
cd laputa && go build ./...
cd ../mentle && go build ./...
```

### Testing with Scripts

```bash
./run-tests.sh          # If present
python laputa-launcher.py --test
```

### Local Development

For rapid iteration during development:

```bash
python laputa-launcher.py --service governance --config dev.yaml
```

---

## Conventions

- **Batch files** (.bat, .cmd) — Windows-only entry points
- **PowerShell** (.ps1) — Windows preferred, cross-platform with caveats
- **Python** (.py) — cross-platform, Python 3.8+
- **Bash** (.sh) — Unix/Linux/macOS (if present)
- **Error handling** — exit codes reflect status; logging to stdout/stderr
- **Configuration** — environment variables or YAML (not hardcoded)
- **No interactive prompts** — use flags and defaults instead

---

## MANUAL

When adding scripts:

1. Create script file in `scripts/` directory
2. Add usage documentation here
3. Ensure cross-platform compatibility or note platform-specific behavior
4. Test on target platforms before committing
5. Update this file with new script descriptions

When removing scripts:

1. Document deprecation reason
2. Provide migration path
3. Keep deprecated scripts in `scripts/archive/` for 1 month

Parent reference: ../AGENTS.md
