# Scope backend cache repair

The actual concurrent cache probe detected DATA RACE and fatal concurrent map read/write in Garden.BackendFor. Thirty-two workspace scopes concurrently request their native adapters sixteen times each. The fixture uses supported real canonical/BM25 facade APIs, not a fake backend or SQL seeding; it is a unit proof, not complete App/ONNX acceptance.

Serialize the existing cache initialization, lookup and lazy adapter admission with a separate short mutex. Do not reuse the lifecycle mutex: Close holds it while draining ingestion, whose workers may call BackendFor. Caller-owned lifetime protection remains required. No authority, scope union, storage schema or injected-backend behavior changes.
