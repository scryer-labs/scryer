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

Java scan/analyze uses the independently built
[scryer-java-engine](https://github.com/scryer-labs/scryer-java-engine).
Build/extract its complete local package, then install it once:

```sh
./build/install/scryer/bin/scryer engine install java --from /absolute/path/to/scryer-java-engine/build/install/scryer-java-engine
./build/install/scryer/bin/scryer project scan ../legacy-maven-fixture --static
./build/install/scryer/bin/scryer project analyze ../repository --before HEAD~1 --after HEAD
```

Core starts the Go wrapper, which starts its JVM worker. The engine owns Java 21,
worker jars and runtime selection; Core owns snapshots, protocol validation and
reports. No Kotlin sources or Gradle files remain in this repository. Missing
engines produce an explicit error; default engine-free stack/LOC is still pending.
Local-package engine installation, listing and uninstall are persistent. Core automatically discovers installed engines. PATH and SCRYER_ENGINE_JAVA remain development overrides; release downloads and source builds are not implemented.

Default Java scans evaluate target build configuration and may download metadata;
`--static` skips build-model execution. Analyze runs target tests unless
`--skip-tests` is supplied. See the [command reference](docs/CLI.md) for modes,
side effects, working-tree comparisons, custom commands and evidence limits.

## Verify and package

```sh
go test -race ./...
go vet ./...
node --test scripts/test-report-graph.cjs
./scripts/package
```

Core packages contain only the Go CLI, embedded reporting assets, README and
license; no jars or Java runtime. Archives are written under `build/distributions/`.
The engine owns its slim and Java-21-runtime packages. Both distributions can be
relocated independently; installed packages are copied in full and discovered automatically.

The [CI workflow](.github/workflows/ci.yml) checks Go and browser report code,
builds Core without Java, verifies extracted packages and standalone `go install`,
and uploads archives/checksums. Engine CI owns JVM tests and packaging. Neither
workflow publishes GitHub Releases yet. Local acceptance uses Orrery plans as
specified in [AGENTS.md](AGENTS.md).

## Documentation and direction

- [Documentation index](docs/README.md): document ownership and navigation.
- [Milestones](docs/MILESTONES.md): completed work and ordered future work.
- [Architecture](docs/ARCHITECTURE.md): current package responsibilities and coding rules.
- [Analyzer contracts](docs/contracts/README.md): current interface and process/report schemas.
- [Analyzer extraction decision](docs/decisions/001-analyzer-distribution.md): independent Java repository, Go wrapper and analyzer installation.
- [Technical debt](docs/TECH-DEBT.md): existing limitations, separate from new features.
- [Verification history](docs/archive/VERIFICATION.md): earlier observations, not current test totals.

Current commands use project/workspace scopes; see [CLI](docs/CLI.md). The first
reorganization preserves Java defaults and removes legacy command entries.
`workspace import` remains a Mod-fed CSV adapter, not
native scanning. Release/source installation, `.scryer/` project configuration and default
engine-free stack/LOC scanning are planned next, not implemented. See
[decision 004](docs/decisions/004-multilanguage-scopes-cli-and-engine-config.md) and
[milestones](docs/MILESTONES.md). Consumers own business/design judgments, plan
execution and long-lived history. macOS remains the primary acceptance platform.

Contributions follow [AGENTS.md](AGENTS.md), including `<type>: <summary>` commits.
