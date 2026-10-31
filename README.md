# Scryer

Scryer collects repository facts and compares code changes with available test
execution evidence. A native Go/Cobra CLI owns repository snapshots, commands and
shared terminal/JSON/Markdown/HTML reports. A Kotlin/JVM worker currently supplies
Java analysis. Workspace dependency processing is Go-only.

Facts, partial results and unknowns remain explicit. Command success, a test file,
a method hit or a dependency relationship is not proof of refactoring safety.

## Build and run

Prerequisites: Go 1.24 or later, Java 21 and Git. The Gradle Wrapper builds the
analyzer. Run these commands from the repository root:

```sh
./scripts/build
./build/install/scryer/bin/scryer --help
./build/install/scryer/bin/scryer scan ../legacy-maven-fixture --static
./build/install/scryer/bin/scryer analyze ../repository --before HEAD~1 --after HEAD
```

`go build ./cmd/scryer` builds only the Go executable. For development, point
`SCRYER_JAVA_CLASSPATH` at analyzer jars, or use the combined installation above.
`SCRYER_ANALYZER_JAVA_HOME` selects the analyzer's Java 21 runtime;
`SCRYER_JAVA_HOME` independently selects the target project's JVM. `--stack java`
is optional and is currently the only registered stack. `./gradlew installDist`
alone installs the internal worker, not the complete public CLI.

Default Java scans evaluate target build configuration and may download metadata;
`--static` skips build-model execution. Analyze runs target tests unless
`--skip-tests` is supplied. See the [command reference](docs/CLI.md) for modes,
side effects, working-tree comparisons, custom commands and evidence limits.

## Verify and package

```sh
go test -race ./...
go vet ./...
./gradlew test
node --test scripts/test-report-graph.cjs
./scripts/package
./scripts/package --with-runtime
```

The default package contains the native CLI and Java analyzer jars and needs
external Java 21. `--with-runtime` adds a host-specific jlink Java 21 runtime and
legal notices. Keep `bin/`, `lib/` and `runtime/` together; this is a directory
distribution, not one monolithic executable. The target build/test JVM remains
separate. Archives are written under `build/distributions/`.

The [CI workflow](.github/workflows/ci.yml) checks Go, JVM and browser report code,
builds both distributions, smoke-checks extracted packages and uploads archives
and checksums as workflow artifacts. It currently builds on Ubuntu; no GitHub
Release publication is configured. See [runtime/package details](docs/CLI.md#bundled-java-runtime).

## Documentation and direction

- [Documentation index](docs/README.md): document ownership and navigation.
- [Milestones](docs/MILESTONES.md): completed work and ordered future work.
- [Architecture](ARCHITECTURE.md): current package responsibilities and coding rules.
- [Analyzer contracts](docs/contracts/README.md): current interface and process/report schemas.
- [Analyzer extraction decision](docs/decisions/001-analyzer-distribution.md): planned
  independent Java repository, Go wrapper and analyzer installation.
- [Technical debt](docs/TECH-DEBT.md): existing limitations, separate from new features.
- [Verification history](VERIFICATION.md): earlier observations, not current test totals.

The next direction is explicit project/workspace commands with handwritten
`.scryer/` configuration and JSON contracts, followed by independent Java engine
installation/extraction and an Orrery provider update. These changes are **not
implemented**; current usage remains in [CLI](docs/CLI.md). See
[decision 002](docs/decisions/002-scan-scopes-and-fact-boundary.md) and
[milestones](docs/MILESTONES.md). Scryer supplies facts through language engines;
consumers own judgments, plan execution and long-lived history. macOS is the
primary development/acceptance platform; Windows migration is deferred.

Contributions follow [AGENTS.md](AGENTS.md), including `<type>: <summary>` commits.
