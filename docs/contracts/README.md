# Analyzer contract v1

Core dispatches project facts through installed engine capabilities.
`internal/contract.Analyzer` is Core's internal use-case interface.
The engine boundary uses versioned JSON carried by the C ABI.
The Java scan/analyze examples below describe the Java report contract.

## Engine transport

Official packages declare their ABI library, identity and capabilities in
`engine.json`. Core loads the [C ABI v1 contract](ENGINE-ABI.md) and invokes
`execute_v1` with a JSON request. Java forwards its request to the JVM worker
over stdin/stdout and keeps diagnostics on stderr. Core also retains a legacy
executable adapter for development overrides. A valid response may describe a
structured engine error or a failed target test. Paths in requests are absolute.

Request envelope:

```json
{
  "protocolVersion": 1,
  "requestId": "opaque-unique-request-id",
  "stack": "java",
  "operation": "scan",
  "scan": {
    "repository": "/repository",
    "static": true,
    "remoteList": false,
    "buildTool": "maven"
  }
}
```

Analyze uses `operation: "analyze"` and an `analyze` object instead:

```json
{
  "repository": "/original/repository",
  "before": "immutable-before-commit-SHA",
  "after": "immutable-after-commit-SHA",
  "beforeRoot": "/temporary/before",
  "afterRoot": "/temporary/after",
  "skipTests": false,
  "testCommand": "./mvnw -B clean verify",
  "buildTool": "maven"
}
```

The Go layer resolves refs and owns the snapshot directories. The analyzer validates snapshot SHAs before analysis. Java retains the Java-specific Git-diff-to-symbol calculation, Maven/Gradle command planning, classpaths, coverage collection and test integrity checks. It never removes caller-owned snapshots. Build/test logs and evidence are retained outside those snapshots in the configured Scryer cache.

Response envelope:

```json
{
  "protocolVersion": 1,
  "requestId": "same-request-id",
  "stack": "java",
  "operation": "scan",
  "report": {"schemaVersion": 1},
  "error": null
}
```

The abbreviated `report` above is not a valid complete report. An error response has `report: null` and `error: {"code": "JAVA_ANALYZER_ERROR", "message": "..."}`. Go rejects mismatched version/request/operation/stack, malformed reports, duplicate symbol IDs, graph edges referencing missing nodes and extra stdout documents. Analyze report refs must match the requested SHAs. Cancellation terminates the JVM process group and ordinary child build processes on POSIX, then Core removes isolated checkouts.

## Report fields

The report schema is versioned separately from the process protocol. Version 1 preserves the existing public JSON facts and uncertainty semantics. Core Go DTOs for builds/modules/dependencies/symbols/relations/changes/execution/matching are defined in `internal/contract/reports.go`; in-process analyzers can use `NewDocument` without implementing the Java process protocol. Required field shapes are checked in `internal/contract/document.go`; additional fields remain intact in JSON/HTML exports. Renderers consume documents in Go, while the analyzer's Java report projection supplies the compatible data structure. `map[string]any` in the renderer is a view over a validated document, not permission to invent evidence.

Scan core:

- `schemaVersion`, `root`, `builds`, `modules`, `dependencies`
- `language`, `testing`, `verification`, `resolution`
- Optional/additive facts: `sources`, `signals`, `frameworks`, `parentPoms`, `compileTooling`, `repositories`, `customBuildFiles`, `selectedBuildTool`, `remoteVersions`, `notes`
- Dependency observations retain `module`, original `configuration`, declaration/provenance and selected versions. Remote comparison retains its original scope, current provenance and failed/unknown lookup status.

Analyze core:

- `schemaVersion`, `generatedAt`, `repository`, `beforeSha`, `afterSha`
- `files`, `changes`, `before`, `after`, `execution`, `evidence`, `matching`, `notes`
- Snapshot: `nodes`, `edges`, `boundaries`, `totalBoundaryCount`, `unresolvedChanges`, `notes`
- Symbol: opaque `id`, human-readable `signature`, `path`, `line`, `role`, `impact`, optional `module`. Java-specific `owner`, `name`, `descriptor` are retained but not required for other stacks.
- Relation: `caller`, `callee`, `kind`. Both endpoints refer to snapshot node IDs.
- Source roles: `PRODUCTION`, `TEST`, `UNKNOWN`. Impact classes: `CHANGED`, `CALLER`, `POTENTIAL_INDIRECT`.
- Execution: `status`, command, immutable `afterSha`, exit/duration/log and notes. Status is `SUCCEEDED`, `FAILED`, `SKIPPED`, `UNAVAILABLE`, `TIMED_OUT` or `SNAPSHOT_CHANGED`.
- Evidence: independent `tests`, `coverage`, `artifacts`, optional `testRecords`/manifest and notes.
- Matching: independent `datasets`, `removed`, `unresolvedChanges`, notes. Each dataset retains per-symbol status/reason/counters/routes and nullable method metrics.

`UNKNOWN` is not zero coverage. `PARTIALLY_EXECUTED` retains partial instruction/branch hits. Static test routes are candidates, not per-test attribution. Datasets are not silently merged. Percentages describe assessed method execution, not refactoring safety. Unsupported analysis remains a boundary with a reason.

Schemas are deliberately additive within v1; incompatible changes require a new report/protocol version and explicit adapter support. Stack capabilities and a fully typed cross-stack IR can evolve from concrete C++ requirements; this first boundary does not prescribe JVM descriptors to C++ or prematurely implement an architecture-guard command.

Working-tree analysis: Go may provide `comparisonRepository` (an isolated Git repository containing both immutable commit objects) independently of the original `repository` path. Java computes the diff there, while retaining the original path for report identity and target environment discovery. Both supplied checkout HEADs still match `before`/`after`. Go adds `afterSource` with `kind`, `ref`, `baseHead` and `snapshotSha` to reports for `--after .`. Consumers must label this as a private working-tree snapshot, not an original committed ref.

Remote scan enrichment may include additive `toolUpdates` (`checkedAt`, `tools`, `notes`). Each row supplies `tool`, `channel`, nullable `current`/`latest`, `currentSource`, `status`, nullable `metadataUrl` and `note`. Java feature-level and unknown patch observations remain separate channels. Tool updates are repository facts plus remote release metadata; they do not establish compatibility or upgrade safety.

## Current generality limits

The transport envelope and Core orchestration are language-independent boundaries.
Report v1 retains the existing Java-led shapes during extraction, including Maven/
Gradle scopes, Java descriptors and JaCoCo provenance. Shape validation is not a
JSON Schema-generated universal language model: document.go checks supported
required shapes and graph/snapshot invariants, while additive fields are retained.
A future engine may require deliberate common-field/extension evolution. Keep
transport version, report schema version and local package schema independent.

## Project configuration versus invocation requests

The request examples above are runtime messages assembled by Core, not files to
copy into .scryer/. Planned engine-provided templates supply persistent project
settings (for example build-tool selection/default test command). Core combines
those with the operation and CLI selection, resolves paths/refs/SHAs, generates
request identity and owns temporary snapshot directories. Engine-specific YAML
schemas and initialization are separate from the current local-package installer.

The response envelope is generic; report is the stack-specific/common-fact payload.
The Java report shown here preserves current behavior rather than prescribing the
schema or scan priorities for C++, Rust, Go or TypeScript. Those engines must follow
the source/architecture/symbol/relation fact goals in decision 004 instead of copying
the Java dependency/version-oriented scan implementation.

## Installed capabilities and selected scan facts

A package can declare `capabilities` in its manifest with `languages`,
`operations`, `scanFacts` and optional `resolveFacts` arrays. `engine list --json` exposes the effective
capabilities and their source. Older Java packages expose the legacy scan and
analyze operations without opt-in filtered facts.

A scan request can include `scan.facts`, a list of fact IDs. A filtered report
has `schemaVersion`, `root` and a `facts` object keyed by the requested IDs.
Core uses the installed capabilities to split a project request across engines.
Core keeps results from other facts and engines when one fact is unsupported.
Java supports filtered `dependencies` and returns `UNSUPPORTED_FACT` or
`UNSUPPORTED_OPERATION` for requests outside its capabilities.


## Project scan consumer contracts

- [Go project facts](https://github.com/scryer-labs/scryer-go-engine/blob/main/docs/FACTS.md)
- [Rust call graph](rust-call-graph.md)
- [TS/JS call graph and Svelte facts](ts-js-call-graph.md)
- [React architecture](react-architecture.md)
- [Test classification](test-classification.md)
- [Tauri IPC](tauri-ipc.md)
- [Project and workspace Windows IPC](WORKSPACE-IPC.md)

These contracts describe engine-specific data in collected project outcomes and
Core's communication projection. Consumers can use scan-local IDs to join facts
within the same report. The optional `resolutionReason` describes an observed
item, and `limitations[]` describes coverage gaps for the enclosing fact.

See also [toolchain-assisted scan facts](RESOLVE.md).

See [runtime coverage](COVERAGE.md) for the standalone coverage operation.

See [static impact projection](IMPACT.md) for the C++ source-evidence boundary
and Core comparison and traversal.
