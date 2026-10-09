# Verification

Fresh review identified inherited palace access and legacy-receipt replay conflicts. Adversarial inherited-path MCP regression was RED, then GREEN; all 8 MCP tests pass. Mentle full suite: 536 pass, 0 skip, exit 0.

Capture lookup regression was RED (capability absent), then GREEN with actual ingest SQLite. It verifies original receipt/sequence, no visibility for alternate session/agent, and authentication rejection. Garden agentapi/ingest focused suite and full Garden suite are green; full Garden: 480 pass, 0 skip, exit 0.

Paired VIVY tests cover observer receipt replay across a real factory/Journal/Garden close/reopen and unchanged direct changed-payload rejection. Full DIVA/new candidate verification is recorded in the DIVA handoff.
