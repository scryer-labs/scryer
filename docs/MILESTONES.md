# Product milestones

See [product direction](PRODUCT-DIRECTION.md) for scope and
[technical debt](TECH-DEBT.md) for maintenance. The items below are in delivery
order. Coverage is an input to impact analysis, and C++ ownership can proceed
independently once its source evidence is available.

## Delivered

- Scope-first CLI, engine-independent stack and LOC, Git repository identity,
  explicit workspace membership and multi-engine dispatch by language and fact.
- C ABI v1 and persistent package management. Java, C++, Rust, TS/JS and Go
  have separate engine repositories. All five publish stable macOS arm64 and
  Windows amd64 ZIPs. Java also publishes Linux amd64 and bundles Java 21.
- JSON and Markdown scans. Core aggregates CMake Git references, Windows named
  pipes and shared memory, and project-local Tauri communication. Java analyze
  retains its execution evidence and report formats.
- First toolchain-assisted `--resolve` pass for C++, Java dependencies, Rust,
  TS/JS (React and Svelte) and Go. See the [resolution contract](contracts/RESOLVE.md)
  for what each engine can resolve and the remaining gaps.
- Remote install, development-to-stable upgrade and static scans verified in
  Orrery **Official Engine Release Installation**
  (`20261114T172344415Z-3f64c660`) and in
  [Windows Core CI](https://github.com/scryer-labs/scryer/actions/runs/43620960684).
  Core CI already builds its own archives and a native Windows executable.

## M1 — Publish Scryer Core

Publish versioned Core packages for macOS arm64 and Windows amd64 using GitHub
Actions. Verify the extracted CLI in an isolated home can install the official
C++ package and scan architecture facts. Include checksums and describe the
contents, supported targets and installation steps. Retain the source-build
path and independent engine releases. Windows compiler-assisted project scans
have a separate toolchain acceptance boundary.

## M2 — C++ scan acceptance

Verify product and test targets across architecture, file membership and
includes, call graph, and IPC. Use Scriptorium for workspace communication
regressions. Cover project-local shared-memory matching with focused fixtures.

Compare static and `--resolve` outputs where the target toolchain and compile
database are available. Fix concrete false links and missing facts, and retain
unsupported conditions and unresolved references. Keep C++ source collection
and CMake build declarations modular. Validate the JSON before handing its
architecture and test evidence to Orrery for interactive rendering.

## M3 — Runtime coverage and change impact

Design a shared execution-evidence contract, then implement the first C++
collector against a repeatable test setup:

1. Bind test executions and coverage observations to a specific revision,
   build configuration and artifact. Keep test identity, observed file/line or
   symbol coverage, and unknown or unavailable evidence distinguishable.
2. Compare immutable before/after inputs, identify changed symbols, and traverse
   relevant incoming and outgoing call paths. Retain unresolved calls and
   distinguish potential static impact from observed execution.
3. Correlate the impact scope with measured coverage. Report which tests
   exercised affected code when the collector has per-test evidence. Preserve
   aggregated datasets in cases where attribution is unavailable.

Java's existing analyze path supplies prior experience with evidence freshness,
provenance and failure outcomes. Its Maven/Gradle and JaCoCo payload remains
Java-specific.

## M4 — Test effectiveness and C++ ownership

- **Test effectiveness.** Design bounded checks for whether tests detect
  behavior changes. Coverage establishes execution evidence. Effectiveness
  needs stronger evidence, such as detection of selected controlled changes.
  Keep reproducible findings and raw facts independently inspectable.
- **C++ ownership and lifetime.** Start with observable ownership, borrowing,
  lifetime and transfer facts. Establish evidence and diagnostics for uncertain
  aliasing or lifetimes. Decide separately whether to cover atomic operations,
  memory ordering and concurrency.

## Deferred

- Pinned engine install versions, remote catalog/discovery, and engine-supplied
  project YAML initialization.
- Java workspace dependencies, REST relationship matching and other IPC
  mechanisms driven by real projects. GraphQL, RPC and branch policy have
  lower priority.
- Broader language or framework adapters, shared rules for code smells and
  further toolchain-resolution work. See
  [product direction](PRODUCT-DIRECTION.md) and [resolution](contracts/RESOLVE.md).

### CMake test target pattern matching

Low priority. Let users select multiple CMake executable targets with a single
pattern in `--test-target`. Expand against the configured CMake target names,
deduplicate the matches in a deterministic order, and report which targets ran.
Keep exact names useful for selecting one target. Decide pattern syntax when
implementing the feature.
