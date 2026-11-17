# Runtime coverage v1

Standalone coverage uses the current project directory and can generate or import
coverage evidence. The CLI chooses an installed engine that
advertises `coverage`. The engine request uses `operation: "coverage"` and
`coverage: {repository, artifact?, testBinaries?, testTargets?}`. Repository
and artifact paths are absolute. Select at most one of the optional inputs.
`artifact` imports evidence. `testBinaries` selects executable paths relative
to the temporary build directory. `testTargets` selects CMake executable
target names. Core sends target selections only if the package declares
`coverageTestTargets`. With no optional input the engine builds and runs
CTest. The ABI functions and request/response version remain unchanged.

The engine returns `schemaVersion`, `root`, `status` and any available evidence.
The project report adds `engine` and retains the JSON evidence:

- `format`, `version`, `artifact` and `sha256` identify the evidence. An
  import keeps the input path. Generated evidence uses the logical name
  `llvm-cov.json` because the intermediate export is temporary.
- Generated reports include `runs[]` with a test name, profile count and
  status (`succeeded`, `failed`, `no-profile` or `no-tests`) and an optional
  diagnostic. CTest has one execution entry. Direct execution has one entry
  per binary, in request order.
- `sourceMatch: "unverified"` records that the C++ evidence has no verified source snapshot. An artifact digest identifies the artifact,
  and does not identify the source revision.
- `granularity: "aggregate"` describes the observed profile. No per-test
  identity is known.
- `files[]` contains repository-relative `path` and line, function and
  region counts as `{covered, count}`. An optional `llvmSegments` contains
  original LLVM segment tuples. Consumers should use LLVM counts for the
  file totals.
- `notes[]` records missing provenance or paths that could not be mapped.

No usable profiles or no mapped files yields `status: "unavailable"`.
Partial test execution with collected evidence yields `status: "partial"`.
Invalid inputs and failed configure/build/export stages return an error. CLI
JSON, Markdown and HTML use the same project report. Future impact analysis
can reuse this operation and must verify evidence against its selected after
snapshot before labelling impacted code.
