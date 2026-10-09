# Scryer layered architecture

Scryer is one Go module with a Kotlin JVM analyzer. Public `scan` and `analyze` are shared use cases owned by Go. The initial stack is Java; introducing C++ does not require adopting Java AST/bytecode structures.

| Layer / location | Responsibility |
|---|---|
| `cmd/scryer/main.go` | Dispatch to the application entry point only |
| `internal/cmd` | Cobra composition, public commands/flags, signal cancellation and presentation choices |
| `internal/cmd/scan`, `internal/cmd/analyze` | Command-specific argument validation and invocation of application use cases |
| `internal/application` | Stack selection and repository/snapshot lifecycle coordination |
| `internal/contract` | Analyzer interface, versioned requests/responses, report document requirements and uncertainty semantics |
| `internal/analyzer/java` | Java 21 worker discovery, process protocol, request identity/version validation; no rendering |
| `internal/repository` | Git ref resolution, shared clone, isolated standalone before/after checkouts and cleanup |
| `internal/process` | Subprocess cancellation/process-group lifecycle |
| `internal/report` | Terminal, JSON, Markdown, Mermaid and offline HTML rendering, atomic output replacement |
| `internal/report/assets` | Shared HTML/CSS/JS/logo assets embedded into the Go binary |
| `analyzers/java/src/main/kotlin` | Java/JVM facts, symbol/impact analysis, Maven/Gradle models and test/coverage evidence |

The flow is `main → Cobra command → application → Analyzer interface → Java process adapter → Kotlin worker → validated report → Go renderer`. No technology-specific rendering or output-file flags cross the analyzer boundary. Presentation does not recompute evidence. JSON output preserves additive fields and nulls.

Go resolves refs to immutable SHAs and creates isolated snapshots. Kotlin consumes those directories, verifies their SHAs, identifies changed Java symbols from Git differences, builds attributed source graphs, runs target tests and matches evidence. Java owns Maven/Gradle and target-JDK details. Go owns the outer lifecycle/cancellation, while Java build/test runners retain command-specific timeout, log and snapshot-integrity handling. Retained artifacts live in the cache, not in temporary checkouts.

`--build-tool` selects declarations/models and default test lifecycle. `--test-command` replaces only the target test command. `--skip-tests` skips build-model commands as well as tests. Unsupported Java features and failed model collection stay explicit partial-analysis notes/boundaries. The process protocol is described in [docs/contracts/README.md](docs/contracts/README.md).

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

The old Kotlin command/rendering code is retained **only under test sources** as regression fixtures. It is not part of the installed JVM analyzer or the public command path. Public rendering assets are owned by Go; those tests receive the same assets as test resources.

## Contract and evidence rules

Protocol version and report schema version are independent. Symbol IDs are opaque to Go; signatures/paths/roles/impact and graph endpoints are common facts. Java descriptors and class-ID coverage provenance are retained without prescribing them to C++. Future analyzers may implement the Go interface directly or use another process adapter. Introduce a registry/capability extension when an actual additional analyzer exists; only Java is composed today.

No existence of a test class, command success, static test route or method hit proves every affected execution path. Missing execution records remain UNKNOWN. Matching datasets stay independent. Method execution percentages exclude UNKNOWN and are not safety scores. Reflection resolves only bounded constants/class literals/local arrays/simple static helpers/exact constructors; mutation, escapes and dynamic/framework behavior remain boundaries.

## Code and dependency rules

Keep `main` thin and dependencies directed toward the contract/application boundaries. CLI must not parse Java symbols or coverage data; analyzers must not render terminal/Markdown/HTML. Prefer named operations, readable collection transformations and explicit side-effect/error handling. Presentation labels are never identity/matching keys. Keep one root `go.mod`; package boundaries are sufficient until independently versioned components are needed.

Build tooling compiles the JVM analyzer and Go executable into `build/install/scryer`: `bin/scryer` is Go, `lib/java/*.jar` is the worker runtime. The native package contains both components and requires an external Java 21 runtime; no JRE is bundled. CI tests Go, JVM and browser graph behavior, then checks extracted packages.
