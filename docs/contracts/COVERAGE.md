# Runtime coverage v1

Standalone coverage uses the current project directory and can generate or import
coverage evidence. The CLI selects matching installed engines that
advertise `coverage`. The engine request uses `operation: "coverage"` and
`coverage: {repository, artifact?, testBinaries?, testTargets?}`. Repository
and artifact paths are absolute. Select at most one of the optional inputs.
`artifact` imports evidence. `testBinaries` selects executable paths relative
to the temporary build directory. `testTargets` selects CMake executable
target names. Core sends target selections only if the package declares
`coverageTestTargets`. With no optional input each engine uses its native test lifecycle. C++ builds
and runs CTest. Go uses `go test`, Rust uses Cargo tests, and TS/JS uses supported
package test scripts. The ABI functions and request/response version remain unchanged.

The engine returns `schemaVersion`, `root`, `status` and any available evidence.
The project report adds `engine` and retains the JSON evidence:

- `format`, `version`, `artifact` and `sha256` identify the evidence. An
  import keeps the input path. Generated evidence uses the logical name
  `llvm-cov.json`, `go-coverprofile.out` or `ts-js-coverage.json` because intermediate exports are temporary.
- Generated reports include `runs[]` with a test name, profile count and
  status (`succeeded`, `failed`, `no-profile` or `no-tests`) and an optional
  diagnostic. CTest has one execution entry. Direct execution has one entry
  per binary, in request order.
- `sourceMatch: "unverified"` records that the imported/current-tree evidence has no verified source snapshot. An artifact digest identifies the artifact,
  and does not identify the source revision.
- `granularity` identifies aggregate LLVM regions, Go statement blocks or
  JavaScript source regions. No per-test identity is known.
- `files[]` contains repository-relative `path` and line, function and
  region counts as `{covered, count}`. Normalized `regions[]` describes start/end line and byte-column with an
  execution count. Optional `llvmSegments` retains the original LLVM tuples. Collectors retain their own file totals; region counts from different formats
  are not interchangeable.
- `notes[]` records missing provenance or paths that could not be mapped.

No usable profiles or no mapped files yields `status: "unavailable"`.
Partial test execution with collected evidence yields `status: "partial"`.
Invalid inputs and failed configure/build/export stages return an error. CLI
JSON, Markdown and HTML use the same project report. Impact reuses this operation on the after snapshot and verifies checkout identity
before associating evidence with affected symbols.

## Native collection and multi-engine projects

`--stack` optionally restricts collection; otherwise Core invokes all installed,
matching, coverage-capable engines. CMake test targets/binary paths are forwarded
only to engines declaring `coverageTestTargets`. The same invocation can select
C++ targets while Go, Rust and TS/JS run native defaults. A supplied selection
with no supporting selected engine is an error. An imported `--coverage-file`
requires `--stack` when several engines match.

A single-engine report keeps its original shape. A multi-engine report has
`engineReports[]`, each containing its engine ID, status, evidence and runs.
Top-level status is `partial` if any child is not collected. Top-level `runs[]`
qualifies execution names with the engine ID. There is no combined percentage.
Optional run `command[]`, `directory` and `configuration` retain execution inputs;
these fields do not establish per-test causality.

- **Go:** installed Go toolchain and offline-resolvable modules/workspace;
  `go test -mod=readonly -count=1` runs each discovered module. Project module
  import patterns select instrumentation. Import accepts set/count/atomic
  coverprofiles. `_test.go` files are not instrumented.
- **Rust:** Cargo/rustc, a frozen dependency set and LLVM coverage/profdata tools
  matching rustc's LLVM major. `SCRYER_RUST_LLVM_BIN` selects their directory.
  Cargo tests run in a temporary instrumented target directory. Doctests and
  per-test attribution are not collected. Import accepts LLVM exports.
- **TS/JS:** Node/npm and the project's installed test dependencies. Vitest needs
  its matching `@vitest/coverage-v8`; Jest and Vitest emit remapped Istanbul
  statements. Native `node --test` supports raw V8 only for untransformed JS.
  Unsupported runners, absent mappings and missing providers are explicit.
  Imports accept Istanbul JSON or raw V8 coverage JSON, with the same limits.

Tests execute with the user's environment. Temporary output directories isolate
collector files, not arbitrary side effects of project scripts. Native runtime
acceptance currently targets macOS; Windows packages have ABI/build checks,
not equivalent runtime coverage acceptance.
