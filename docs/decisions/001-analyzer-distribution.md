# Work decision 001: Independent analyzer distribution and Go integration

Date: 2026-10-25
Status: Agreed direction; extraction, installation and plugin packaging are not implemented.

## Context and current implementation

The main repository now has a Go CLI/application/reporting layer, a Go
`internal/contract.Analyzer` interface and a Java process adapter. Java analysis
remains in `analyzers/java/` as a Kotlin/JVM worker. The Go host owns snapshots,
request orchestration and common presentation; Java owns language/build/test facts.
The installed application currently bundles the worker. Only Java is composed.
See [architecture](../../ARCHITECTURE.md) and [current contracts](../contracts/README.md).

This split is useful but does not provide an independent repository, installable
analyzer discovery, a plugin registry or a public Go SDK. In particular, an
external Go repository cannot import the host's `internal/` contract packages.

## Agreed direction

- Extract Java analysis into an independently versioned repository. Its public
  integration/package is Go-wrapped; Kotlin/JVM analysis stays behind that wrapper.
  This is a distribution/ownership change, not a rewrite of Java analysis in Go.
- Provide Go-oriented analyzer integration for other stacks and an installation
  flow such as `scryer install java`. Installed analyzers must be usable through
  the ordinary Scryer commands without users pointing to a source checkout or Go
  source files. The example installs an analyzer, not a system-wide Java/JDK.
  Users run packaged programs; a local Go compiler should not be
  required merely to use an installed analyzer.
- Keep public CLI/report composition and repository snapshot lifecycle in Scryer.
  Analyzer-specific dependencies, build adapters and evidence collectors belong
  with their analyzer. Machine settings and runtime dependencies stay distinct
  from portable scan requests and report facts.
- Use a minimal externally consumable contract/SDK or protocol; do not expose all
  host internals or prescribe Java bytecode descriptors to other languages.
  Interface reuse, public Go module boundaries and report-version evolution must
  be decided before extraction. Existing transport/report versions remain separate.
- A Go wrapper does not imply Go's shared-library `plugin` API, in-process loading
  or a custom RPC framework. The current process/JSON boundary is a practical
  starting point; exact plugin transport and discovery are not yet selected.

## Installation and compatibility questions for M1

Decide these against the working Java package rather than inventing an unused
plugin framework:

1. Package identity/version, supported OS/architecture and release source. Java
   worker jars and optional analyzer Java 21 runtime must remain discoverable after
   relocation; the target project's JVM remains independently selectable.
2. User-level install/config/cache locations; explicit version selection and the
   default selection policy. Do not silently reinterpret prior report provenance
   when an analyzer is updated.
3. Download integrity verification, archive extraction constraints, interruption
   handling and atomic publication of a complete installation. Define uninstall,
   update and rollback expectations before exposing those commands.
4. Analyzer discovery and capability negotiation, including absent, incompatible
   and unsupported-operation outcomes. A Markdown analyzer need not pretend to
   implement Java coverage collection just to satisfy a universal interface.
5. Cancellation, progress/stderr versus machine-readable stdout, malformed-report
   handling and process/runtime lifecycle. Preserve current snapshot cleanup and
   explicit structured analysis failures.
6. Minimal cross-repository compatibility tests, package smoke checks and release
   ownership. Move Java-specific tests with the worker; retain host-side contract
   fixtures and end-to-end evidence that both independently versioned pieces work.

Exact flags, repository name, published package schema, SDK module path and
installation filenames are intentionally undecided. This document does not
create another repository or implement an installer.

## Consumer and responsibility boundaries

Scryer and installed analyzers remain usable independently of Orrery. Orrery may
bundle/install a compatible Scryer distribution, invoke it and retain/display its
facts. Agent concurrency, manual review, plan execution and workspace history stay
with Orrery or another consumer; analyzer installability must not make Scryer
responsible for orchestration of refactoring plans.

Cross-stack architecture and test analysis extend facts/capabilities, with explicit
provenance and unknowns. Human-readable diagrams use shared report data rather
than embedding UI-specific interpretation inside language collectors.

## Delivery tracking

[Milestones](../MILESTONES.md) owns priority/status. [Technical debt](../TECH-DEBT.md)
owns existing limitations. Keep [architecture](../../ARCHITECTURE.md) descriptive
of implemented code until extraction actually changes package ownership.
