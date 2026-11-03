# Scryer layered architecture

Scryer Core is a Go module; Java facts come from the independent scryer-java-engine repository. Public `project scan` and `project analyze` are shared use cases owned by Go. The initial stack is Java; introducing C++ does not require adopting Java AST/bytecode structures.

| Layer / location | Responsibility |
|---|---|
| `cmd/scryer/main.go` | Dispatch to the application entry point only |
| `internal/cmd` | Cobra composition, public commands/flags, signal cancellation and presentation choices |
| `internal/cmd/project` | Project scope composition in project.go; scan/analyze arguments and presentation in separate operation files |
| `internal/cmd/workspace` | Workspace scope composition in workspace.go; temporary CSV input in import.go |
| `internal/cmd/engine` | Scope composition in engine.go; persistent local-package list/install/uninstall operations in separate files |
| `internal/cmd/terminal` | Shared terminal color policy across scopes |
| `internal/application` | Stack/snapshot lifecycle coordination and use-case orchestration |
| `internal/engine` | Package manifest/identity probing, user storage, inventory locks, owned copying and deletion |
| `internal/contract` | Analyzer interface, versioned requests/responses, report document requirements and uncertainty semantics |
| `internal/analyzer/external` | Engine executable discovery, process protocol, request identity/version validation; no JVM or rendering |
| `internal/repository` | Git ref resolution, shared clone, isolated standalone before/after checkouts and cleanup |
| `internal/process` | Subprocess cancellation/process-group lifecycle |
| `internal/report` | Terminal, JSON, Markdown, Mermaid and offline HTML rendering, atomic output replacement |
| `internal/report/assets` | Shared HTML/CSS/JS/logo assets embedded into the Go binary |
| External `scryer-java-engine` repository | Go wrapper, Kotlin/JVM facts, Maven/Gradle models and test/coverage evidence |

The flow is `main → Cobra command → application → Analyzer interface → external process adapter → independent Go engine wrapper → Kotlin worker → validated report → Go renderer`. No technology-specific rendering or output-file flags cross the analyzer boundary. Presentation does not recompute evidence. JSON output preserves additive fields and nulls.

Go resolves refs to immutable SHAs and creates isolated snapshots. Kotlin consumes those directories, verifies their SHAs, identifies changed Java symbols from Git differences, builds attributed source graphs, runs target tests and matches evidence. Java owns Maven/Gradle and target-JDK details. Go owns the outer lifecycle/cancellation, while Java build/test runners retain command-specific timeout, log and snapshot-integrity handling. Retained artifacts live in the cache, not in temporary checkouts.

`--build-tool` selects declarations/models and default test lifecycle. `--test-command` replaces only the target test command. `--skip-tests` skips build-model commands as well as tests. Unsupported Java features and failed model collection stay explicit partial-analysis notes/boundaries. The process protocol is described in [docs/contracts/README.md](contracts/README.md).

## Java analyzer packages

- `scan.inspect`: local declarations/layout/source signals; no build execution.
- `scan.resolve`: build-native dependency/model adapters and target-JDK process handling.
- `scan.remote`: Maven Central enrichment, deduplicated lookups and scoped selected-version matching.
- `analyze`: Java method differences, symbol attribution, reverse caller traversal and conservative forward impact candidates.
- `analyze.model`: module roots, external classpaths and reactor source attribution. Maven uses temporary external-only effective POMs; no reactor install is required.
- `analyze.execute`: target lifecycle planning, timeout/cancellation, logs and after-snapshot integrity.
- `analyze.evidence`: fresh JUnit/JaCoCo artifacts, hash/class-ID provenance and independent datasets.
- `analyze.match`: affected production symbols versus execution observations; UNKNOWN and partial hits remain distinct.
- `analyze.report`: Java projection into the compatible versioned report document; no presentation.
- `worker`: one request/response over stdin/stdout. Progress/errors are separate from report facts.

The Java packages above live only in the engine repository. Kotlin produces JSON facts and has no CLI rendering, Markdown/HTML assets or dependency on Core reporting resources. Old Kotlin presentation fixtures/tests were removed; Core owns presentation regression checks.

## Contract and evidence rules

Protocol version and report schema version are independent. Symbol IDs are opaque to Go; signatures/paths/roles/impact and graph endpoints are common facts. Java descriptors and class-ID coverage provenance are retained without prescribing them to C++. Future analyzers may implement the Go interface directly or use another process adapter. Introduce a registry/capability extension when an actual additional analyzer exists; only Java is composed today.

No existence of a test class, command success, static test route or method hit proves every affected execution path. Missing execution records remain UNKNOWN. Matching datasets stay independent. Method execution percentages exclude UNKNOWN and are not safety scores. Reflection resolves only bounded constants/class literals/local arrays/simple static helpers/exact constructors; mutation, escapes and dynamic/framework behavior remain boundaries.

## Code and dependency rules

Keep `main` thin and dependencies directed toward the contract/application boundaries. CLI must not parse Java symbols or coverage data; analyzers must not render terminal/Markdown/HTML. Prefer named operations, readable collection transformations and explicit side-effect/error handling. Presentation labels are never identity/matching keys. Keep one root `go.mod` for the current application; do not scatter modules through the existing tree. The independent Java repository now communicates through the JSON process contract; it imports no Core internal Go packages. Local package installation is persistent; release/source installation and capability/template negotiation remain future work.

Core `scripts/build` produces `build/install/scryer/bin/scryer` with Go only.
The engine's `scripts/build` produces its Go wrapper and worker jars under
`build/install/scryer-java-engine/{bin,lib/java}`. Its optional runtime package adds
`runtime/`. Core never selects a JDK/classpath; engine discovery uses
`SCRYER_ENGINE_<ID>` (uppercase, hyphens replaced by underscores), then persisted
installed inventory, then `scryer-<id>-engine` on PATH. Explicit invalid overrides fail without fallback.

On macOS, Core starts one process group containing the wrapper. The JVM inherits
that group, as do ordinary child build/test processes, so cancellation reaches the
whole group. The wrapper must not create a separate JVM process group. Windows
whole-tree cancellation remains outside current platform acceptance.

## Workspace facts

`workspace import` is a Go-only temporary external-CSV use case. `internal/workspace` owns CSV adapters
and cross-repository facts; the command owns flags and terminal presentation.
No JVM analyzer is needed. Dependency arrows run from consumer to dependency.
Declared edges are not runtime calls or inferred release-phase constraints.

Workspace JSON and HTML render the same schema-versioned, sorted document.
The former `refactor-plan generate` command, phase scheduling and platform-plan
YAML exporter have been removed. Execution-plan generation belongs to consumers
such as Orrery. The workspace CSV adapter is temporary compatibility code under
[TD-006](TECH-DEBT.md#td-006-retire-csv-independencies-out-workspace-adaptation),
not the future native workspace scanner. Legacy entries are removed; consumers must use the scope-first commands. Renderers present the normalized facts
without computing a schedule.

## Documentation and future work

This file describes the implementation, not the future plugin system. The
[documentation index](README.md) separates current contracts from plans;
[milestones](MILESTONES.md) owns delivery order and
[technical debt](TECH-DEBT.md) records existing limitations. Release/source
engine installation, non-Java analysis and architecture/test-strategy
capabilities must not be described as present until their slices are implemented.

## Installed package and fact-contract boundaries

The default global home is ~/.config/scryer, matching Orrery on macOS rather than
using the platform-native Library/Application Support directory. SCRYER_HOME
remains an optional override.

engine.json and .scryer-install.json describe local tool installation, not source
facts. internal/engine owns that data; internal/contract owns request/response and
report data. Install copies complete packages and validates describe without
requiring the Java runtime to start. A staged package is renamed into inventory
only after successful verification; installed packages are not overwritten.
Inventory mutation is serialized across processes. Uninstall checks ownership
before deletion; it never treats a source checkout as an installed engine.

The current report v1 was preserved from Java extraction. Its envelope is generic,
but fields/semantics still reflect Java dependencies, symbol descriptors and
JaCoCo evidence. Do not describe it as a mature universal multilanguage schema.
New languages should drive common-fact and language-extension refinements; local
installation must not silently redesign the scan/analyze data contract.

## Report views by engine

Core selects an engine view by document stack in internal/report/engine_view.go.
The Java view in java.go owns the Java scan presentation and dependency projection,
and supplies Java-specific analyze labels, source fences and runtime fields.
Shared report code still owns JSON, formatting primitives, analyze layout and
offline HTML. A scan with no specialized view renders the structured facts.
Unlabelled legacy documents retain the Java view.

This is a Core-local presentation boundary, not a plugin loader or a change to
the existing JSON schema. New engines still need deliberate fact-contract design.
Project engine YAML initialization and reading remain planned.
