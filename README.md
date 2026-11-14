# Scryer

Scryer collects repository facts and compares code changes with available test
execution evidence. A native Go/Cobra CLI owns repository snapshots, commands and
shared terminal/JSON/Markdown/HTML reports. A Kotlin/JVM worker currently supplies
Java analysis. Workspace dependency processing is Go-only.

Facts, partial results and unknowns remain explicit. Command success, a test file,
a method hit or a dependency relationship is not proof of refactoring safety.

## Build and run

Core prerequisites: Go 1.24 or later and Git. Core builds and installs without
Java, Kotlin, Gradle or an engine checkout:

```sh
./scripts/build
./build/install/scryer/bin/scryer --help
# Alternatively, install the Go command from this checkout:
go install ./cmd/scryer
```

Install the official engines you need for your projects:

```sh
scryer engine install cpp rust ts-js go java
scryer engine upgrade cpp rust ts-js go java
scryer project scan . --facts architecture,files,call-graph --json
```

Core discovers installed engines and loads their ABI libraries. The Java engine
runs its JVM worker with a bundled Java 21 runtime. Use `--from` to install a
local package during engine development. See the [command reference](docs/CLI.md)
for scan modes, resolve prerequisites and analysis options.

## Verify and package

```sh
go test -race ./...
go vet ./...
node --test scripts/test-report-graph.cjs
./scripts/package
```

Core packages contain only the Go CLI, embedded reporting assets, README and
license; no jars or Java runtime. Archives are written under `build/distributions/`.
The Java engine owns its slim and Java-21-runtime packages. Installed engine
packages are copied in full and discovered automatically.

The [CI workflow](.github/workflows/ci.yml) checks Go and browser report code,
builds Core without Java, verifies extracted packages and standalone `go install`,
and uploads archives/checksums. The Windows workflow checks the released engines
through the native CLI. Official engine repositories publish platform-specific
GitHub Releases. Install them with `scryer engine install cpp rust ts-js go java`.
Local acceptance uses Orrery plans as specified in [AGENTS.md](AGENTS.md).

## Documentation and direction

- [Documentation index](docs/README.md): document ownership and navigation.
- [Milestones](docs/MILESTONES.md): completed work and ordered future work.
- [Architecture](docs/ARCHITECTURE.md): current package responsibilities and coding rules.
- [Analyzer contracts](docs/contracts/README.md): current interface and process/report schemas.
- [Analyzer extraction decision](docs/decisions/001-analyzer-distribution.md): independent Java repository, Go wrapper and analyzer installation.
- [Technical debt](docs/TECH-DEBT.md): existing limitations, separate from new features.
- [Verification history](docs/archive/VERIFICATION.md): earlier observations, not current test totals.

Current commands use project/workspace scopes; see [CLI](docs/CLI.md). Project scan defaults to Core stack and LOC facts.
`workspace scan` collects facts from configured members and matches supported cross-project relationships.
Release/source installation and `.scryer/` project configuration remain planned.
Use `scryer project scan --facts stack,loc` for explicit fact selection. See
[decision 004](docs/decisions/004-multilanguage-scopes-cli-and-engine-config.md) and
[milestones](docs/MILESTONES.md). Consumers own business/design judgments, plan
execution and long-lived history. macOS remains the primary acceptance platform.

Contributions follow [AGENTS.md](AGENTS.md), including `<type>: <summary>` commits.

Use the global `--compact` flag with `--json` to emit a single-line JSON
report. This applies to project and workspace scans and project analysis.
