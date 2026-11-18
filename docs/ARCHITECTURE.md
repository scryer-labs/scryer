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
| `internal/report` | Terminal, JSON and Markdown output. Java analyze also owns an offline HTML view. |

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
For C++ impact, the engine supplies per-snapshot source identities, token
fingerprints and call candidates. Core compares declarations and traverses
incoming and outgoing graph edges in `internal/application/impact.go`.
The language-specific projection is owned by the engine. Git
snapshot defaults belong to the repository layer. The application exposes
coverage as a reusable use case and selects an installed engine with the
coverage capability. The C++ engine imports LLVM evidence and returns JSON.
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
