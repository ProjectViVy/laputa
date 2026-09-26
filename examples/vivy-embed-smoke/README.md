# Vivy-style embedded Go consumer smoke

This is an **independent Go module**, not a Garden package or Vivy integration. It imports the public `github.com/dashimaki/garden/agentapi` API and uses Laputa's public `persona` package only for operator fixture initialization. Local `replace` directives point to sibling source modules; this proves **local importability**, not remote module publication or Vivy product integration.

From this directory:

```sh
CGO_ENABLED=0 GOSUMDB=off GOTOOLCHAIN=local go run .
```

Requires Go 1.26.4, module dependencies, and the sibling `garden`, `laputa`, and `mentle` directories. If Garden's Go embed prerequisite reports `console/dist: no matching files found`, build Garden's Console first using its documented build workflow; generated Console assets are not part of this example.

The executable creates and deletes its own temporary absolute paths, with a deliberately absent local model (`RequireLocalModel=false`). It checks in-process open/bind, frozen-only bootstrap/fast recall (WORLD excluded), explicit WORLD read, unavailable Mentle card search while offline, rejection of a conflicting binding and nonterminal capture, terminal capture receipt/replay/status, a closed handle, and durable status after reopening. It makes no HTTP calls or model downloads. A `PASS` line and zero exit status mean all checks passed; any failure prints `FAIL` and exits nonzero. This is a smoke, not transport parity or remote-release evidence.
