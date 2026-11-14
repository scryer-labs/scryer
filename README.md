# Scryer

Scryer collects source facts for repositories and workspaces. The Go CLI
dispatches scans to installed C++, Rust, TS/JS, Go and Java engines and produces
JSON and Markdown. Java also supports change analysis with test execution evidence.
Orrery can consume the JSON for interactive architecture.

## Build and run

Core builds with Go 1.24 or later. Git is used for repository facts and analysis.
Use the source build while the versioned Core release is pending:

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
and uploads archives/checksums. The [Windows workflow](.github/workflows/windows.yml) checks the released engines
through the native CLI. Official engine repositories publish platform-specific
GitHub Releases. Install them with `scryer engine install cpp rust ts-js go java`.
Local acceptance uses Orrery plans as specified in [AGENTS.md](AGENTS.md).

## Documentation

- [Command reference](docs/CLI.md)
- [Architecture and contracts](docs/README.md)
- [Milestones](docs/MILESTONES.md)
- [Contributor instructions and Orrery acceptance](AGENTS.md)
