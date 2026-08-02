#!/usr/bin/env python3
"""Laputa launcher — guarantee a shared Laputa instance for this session.

Designed for the multi-session sharing model:
  - One Laputa instance runs as the canonical HTTP layer (port 7373)
  - Each session (or script) needs to USE that instance, not start its own

Strategy:
  1. Probe http://127.0.0.1:7373/healthz with a short timeout
  2. If 200 OK -> already running, return immediately
  3. If connection refused or non-200 -> run the supervisor script
     (which is idempotent: spawns only if port is free)
  4. Re-probe; if still not up, fail loud with diagnostic hints

Usage:
  python laputa-launcher.py             # probe + ensure + print URL
  python laputa-launcher.py --check     # probe only, exit 0/1
  python laputa-launcher.py --restart   # kill existing, then ensure fresh

Exit codes:
  0 = laputa is up at 127.0.0.1:7373
  1 = laputa failed to come up; see stderr
  2 = --check and not running
"""
from __future__ import annotations
import argparse
import json
import os
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

DEFAULT_URL = "http://127.0.0.1:7373"
SUPERVISOR = Path(r"C:\Users\Administrator\Desktop\projects\laputa\scripts\laputa-supervisor.cmd")
HEALTH_TIMEOUT = 2.0
STARTUP_WAIT = 6  # max seconds to wait for spawn


def probe(base_url: str) -> dict | None:
    """Return parsed JSON if /healthz is up, else None."""
    try:
        with urllib.request.urlopen(f"{base_url}/healthz", timeout=HEALTH_TIMEOUT) as r:
            return json.loads(r.read().decode())
    except (urllib.error.URLError, urllib.error.HTTPError, socket.error, json.JSONDecodeError):
        return None


def wait_up(base_url: str, deadline: float) -> dict | None:
    while time.time() < deadline:
        info = probe(base_url)
        if info and info.get("ok"):
            return info
        time.sleep(0.4)
    return None


def kill_existing(port: int) -> None:
    """Kill any process listening on the given port (Windows)."""
    out = subprocess.run(["netstat", "-ano"], capture_output=True, text=True).stdout
    for line in out.splitlines():
        if f":{port} " not in line or "LISTENING" not in line:
            continue
        parts = line.split()
        if len(parts) >= 5 and parts[-1].isdigit():
            subprocess.run(["taskkill", "//F", "//PID", parts[-1]],
                           capture_output=True, text=True)


def run_supervisor() -> None:
    if not SUPERVISOR.exists():
        print(f"FATAL: supervisor script missing at {SUPERVISOR}", file=sys.stderr)
        sys.exit(1)
    # One-shot mode: spawn if not running, then exit.
    # Mute stdout/stderr so the .bat wrapper that called us doesn't
    # accidentally parse our diagnostic prints as commands (cmd.exe
    # greedily consumes anything piped into stdin if it isn't a tty).
    # We only care about the exit code.
    try:
        with open(os.devnull, "rb") as devnull:
            result = subprocess.run(
                ["cmd", "//c", str(SUPERVISOR)],
                stdin=devnull,
                stdout=devnull,
                stderr=devnull,
                timeout=30,
            )
        if result.returncode not in (0, 2):
            print(
                f"WARNING: supervisor returned exit code {result.returncode}",
                file=sys.stderr,
            )
    except subprocess.TimeoutExpired:
        print("FATAL: supervisor timed out after 30s", file=sys.stderr)
        sys.exit(1)
    except OSError as exc:
        print(f"FATAL: failed to invoke supervisor: {exc}", file=sys.stderr)
        sys.exit(1)


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--url", default=DEFAULT_URL)
    p.add_argument("--check", action="store_true",
                   help="Probe only, do not start anything")
    p.add_argument("--restart", action="store_true",
                   help="Kill existing laputa then ensure fresh")
    p.add_argument("--quiet", action="store_true",
                   help="Only print on error")
    args = p.parse_args()

    if args.restart:
        port = int(args.url.rsplit(":", 1)[-1].rstrip("/"))
        kill_existing(port)
        time.sleep(1)

    info = probe(args.url)
    if info and info.get("ok"):
        if not args.quiet:
            print(f"laputa OK at {args.url} (uptime={info.get('uptime', '?')})")
        return 0

    if args.check:
        print(f"laputa NOT running at {args.url}", file=sys.stderr)
        return 2

    if not args.quiet:
        print(f"laputa not at {args.url}; invoking supervisor...")
    run_supervisor()

    deadline = time.time() + STARTUP_WAIT
    info = wait_up(args.url, deadline)
    if not info:
        print(
            f"FATAL: laputa still not responding at {args.url} after {STARTUP_WAIT}s.\n"
            f"  Check: tail ~/.laputa/supervisor.log\n"
            f"  Manual: {SUPERVISOR}",
            file=sys.stderr,
        )
        return 1

    if not args.quiet:
        print(f"laputa spawned OK at {args.url} (uptime={info.get('uptime', '?')})")
    return 0


if __name__ == "__main__":
    sys.exit(main())