# Product milestones

See [product direction](PRODUCT-DIRECTION.md) for scope and
[technical debt](TECH-DEBT.md) for maintenance. Milestone IDs identify work rather than a fixed delivery
order. Coverage is an input to impact analysis. The deferred M4/M5 extensions
do not block consumers from using the existing facts, coverage and impact APIs.

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

Project-level include/exclude selection is implemented for Core and C++ with
shared file inventory, scan scope metadata and pinned impact scope. See
[project scope](contracts/PROJECT-SCOPE.md). User-defined source/test groups and
other Engine adoption remain follow-up work; scope selection does not establish
third-party, generated or test classifications.

Compare static and `--resolve` outputs where the target toolchain and compile
database are available. Fix concrete false links and missing facts, and retain
unsupported conditions and unresolved references. Keep C++ source collection
and CMake build declarations modular. Validate the JSON before handing its
architecture and test evidence to Orrery for interactive rendering.

### Current C++ delivery order

The source-development slice after v0.3.0 now includes bounded CMake variables
and source lists, evidence-based header provider candidates, macro-body source
scopes, macro/typed-callback candidate links, and POSIX/native-loader observations.
See the Engine [fact extensions](https://github.com/scryer-labs/scryer-cpp-engine/blob/main/docs/FACTS.md)
and [communication contract](contracts/WORKSPACE-IPC.md). Candidates do not become
confirmed ownership or runtime dispatch. These additions ship in C++ Engine v0.4.0; use that version or newer
with current Core for remote install/upgrade.

Next priorities:

1. Stronger header/provider binding and unsupported CMake constructs, driven by
   concrete fixtures; preserve ambiguous ownership and conditional declarations.
2. Large-repository acceptance: explicit scope and measured time/memory/JSON size,
   with regressions for source parser recovery and indirect callbacks.
3. Completeness diagnostics and stronger value propagation: unresolved compiler
   inputs, typedefs/macros, callback mutation, indirect/global resource names,
   POSIX stream endpoints and loader replacement remain explicit boundaries.
4. Additional deterministic test-framework registration fixtures. GoogleTest and
   Catch2 registration evidence are supported; a matching macro name alone is
   never a test identity. User-defined groups remain separate follow-up metadata.

Keep CMake and C/C++ collectors modular. Function-body control/data flow and
execution-order modeling wait until ownership/lifetime work needs them. Function
implementation similarity for external test consumers is not a current requirement.
Test effectiveness and stronger per-test coverage attribution remain deferred.
Continue macOS acceptance first; Windows dynamic acceptance and test-target pattern
matching remain later work. Legitimate compact reports can exceed consumer limits;
Orrery owns artifact storage/loading capacity independently of scan selection.

## M3 — Runtime coverage and change impact

Status: baseline delivered; stronger execution attribution remains follow-up work.

Current macOS coverage can build instrumented CMake targets and run tests.
The first C++ impact slice compares immutable function projections and traverses
candidate calls in Core. Impact now reuses coverage on the after snapshot and verifies that the checkout
stays at the requested commit before matching source regions to affected symbols.
Go, Rust and TS/JS now expose the same separate `coverage` and `impact`
operations, with native collection, scan reuse and multi-engine aggregation.
macOS acceptance covers Go tests, Cargo tests, Node tests, React TSX/Vitest,
Svelte/Vitest, Jest and a mixed C++/Go/Rust/JS fixture.
Engine/stage partial completion retains usable graphs and independent evidence;
coverage gaps remain unknown, and TS/JS semantic diagnostics preserve partial
compiler observations without static retry.
Workspace analyze orchestrates configured project analyses with independent
baselines, complete child reports and explicit partial failures.
Rust dynamic graphs remain Cargo-validated source candidates; per-test
attribution and complete runtime dispatch remain open. The baseline uses these
evidence principles; stronger per-test attribution is follow-up work:

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

## M4 — Test effectiveness

Status: deferred. Resume priority relative to M5 is not decided. Existing scan,
coverage and impact facts can support consumers without completing this milestone.

Design bounded, reproducible checks for whether tests detect behavior changes.
Define rule-file inputs, Engine interpretation and the evidence contract when
this milestone resumes. Coverage establishes observed execution; effectiveness
needs stronger evidence, such as detection of selected controlled changes.
Keep findings, diagnostics and raw facts independently inspectable. Do not
present coverage percentages as assertion effectiveness or safety scores.

## M5 — C++ ownership and lifetime

Status: deferred. Resume priority relative to M4 is not decided. Function-internal
control/data-flow collection is deferred to this work rather than required by
current architecture/call-graph scanning.

Start with observable ownership, borrowing, lifetime and transfer facts in the
C++ Engine. Establish evidence and diagnostics for uncertain aliasing or
lifetimes. Decide separately whether to cover atomic operations, memory ordering
and concurrency. Keep unresolved observations explicit and reusable by consumers.

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
