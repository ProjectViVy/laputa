<!-- Parent: ../AGENTS.md -->

# Retired JSON Authority Store

`.laputa/sections/` contains the previous JSON section fixtures and descriptors. It is not part of the current Laputa contract.

The current authority model is defined by [`../../docs/architecture/0012-laputa-markdown-clean-break.md`](../../docs/architecture/0012-laputa-markdown-clean-break.md): one profile-level Markdown authority directory containing `IDENTITY.MD`, `RELATIONSHIP.MD`, `REDLINE.MD`, `USER.MD`, `DREAM.MD`, `DARK.MD`, and `WORLD.MD`, plus separate `ACTMEM.MD` semantics.

Do not add files, schemas, migrations, or tests to this directory. The clean-break implementation must stop reading it without translating, importing, or falling back to its data.
