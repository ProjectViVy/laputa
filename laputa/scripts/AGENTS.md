<!-- Parent: ../AGENTS.md -->

# laputa/scripts — Legacy Utility Review Area

Scripts in this directory were written for the retired Governance runtime. Treat each as a delete-or-rehome candidate before use.

- Do not initialize or discover `.laputa/sections`, numbered descriptors or JSON Persona state.
- Do not start a standalone legacy Governance/Web/Rhythm service as a production fallback.
- A retained utility must use configured profile paths, current Persona/ACTMEM services and non-interactive error handling.
- No script may migrate legacy JSON into Markdown, inject WORLD/ACTMEM into context, print secrets or modify Mentle authority directly.
- Update script usage only after the corresponding runtime contract and tests exist.

Parent reference: `../AGENTS.md`
