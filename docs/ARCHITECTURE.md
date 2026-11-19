# Scryer layered architecture

Core is a Go CLI. Each language engine ships a C ABI library and manifest.
Core selects engines from observed languages and declared capabilities.
Java's library launches its Kotlin/JVM worker. All engines return JSON.
The [contracts](contracts/README.md) describe the request and report formats.

| Location | Responsibility |
|---|---|
| `cmd/scryer`, `internal/cmd` | Entry point, Cobra scopes, flags, cancellation and output choices |
| `internal/application` | Project and workspace orchestration, engine dispatch and snapshot lifecycle |
| `internal/projectfacts` | Repository identity, file inventory, stack and line counts |
| `internal/workspace`, `internal/framework/tauri` | Cross-repository and framework-specific relationship aggregation |
| `internal/engine`, `internal/engineabi` | Package inventory, stable releases, ABI loading, buffer ownership and cancellation |
| `internal/analyzer/external`, `internal/contract` | Engine client and versioned request, response and report validation |
| `internal/repository`, `internal/process` | Git refs, isolated checkouts and subprocess lifecycle |
| `internal/report` | Terminal, JSON and Markdown output; analyze has offline HTML coverage and impact views. |

A workspace scan uses explicitly configured members and collects each project's
facts before resolving inter-project edges. The project and workspace may each
expose communication relationships. Source evidence, candidates, directions
and unsupported mechanisms remain available in the document.

Core manages ABI calls and JSON validation. Engine packages declare their
identity and capabilities in `engine.json`. Remote releases are downloaded
and checked for expected size and SHA-256. Installation validates a staged
package before replacing an owned installation. Inventory changes are locked
across processes. The default home on macOS is `~/.config/scryer`, and
`SCRYER_HOME` overrides the default.

## Project analyze operations

CLI validates operation selection and the shared comparison flags.
For modern C++, Go, Rust and TS/JS impact, each engine reuses scan collection and supplies per-snapshot source identities, token
fingerprints and call candidates. Core compares declarations and traverses
incoming and outgoing graph edges in `internal/application/impact.go`.
The language-specific projection is owned by the engine. `--resolve` or a test target/binary path enables engine-resolved scan evidence for both
snapshots and coverage on the after checkout. Dynamic failure stops analysis. The repository layer verifies the
checkout stayed clean at that commit before Core associates normalized
coverage regions with affected symbols. Git
snapshot defaults belong to the repository layer. The application exposes
coverage as a reusable use case and selects an installed engine with the
coverage capability. Engines own native test execution and source normalization (LLVM, Go profiles,
and JavaScript source coverage). `analysis_targets.go` selects engines and routes
CMake selectors only to capable packages. `analysis_merge.go` qualifies graph IDs
and preserves independent engine reports and coverage datasets.
Core renders the result independently of the engine.

## Java analyze

Core resolves refs to immutable SHAs and creates isolated checkouts. The Java
worker validates the inputs, derives changed symbols and static impact, and
can run the target test lifecycle. It owns Maven/Gradle, target-JDK choices,
artifact freshness, and JUnit/JaCoCo interpretation. Core owns presentation
and snapshot cleanup. Retained logs and artifacts are kept outside the
temporary checkouts. POSIX cancellation terminates the JVM process group.
Windows installation and static scanning are verified independently of
target builds, test execution and process-tree cancellation.

## Code and dependency rules

Keep `main` thin and dependencies directed toward application and contract
boundaries. CLI packages compose commands and display results. Keep language
extraction in the engine and share only the protocol and fact contracts.
Renderers use the validated facts and preserve source identity and uncertainty.
Shared presentation belongs in Core. Keep stack-specific presentation in
its own module, such as the Java view and C++ Markdown projection.

Keep one root `go.mod` for Core. Prefer named operations, readable collection
transformations, and explicit side-effect and error handling. See
[contributor instructions](../AGENTS.md) for acceptance and commits.
