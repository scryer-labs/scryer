# Analyzer contract v1

Go owns public commands, snapshots and rendering. The independent Java engine
imports no Core internal packages: the interoperable boundary is JSON, not Go
struct linkage. `internal/contract.Analyzer` is Core's internal use-case interface.
Only Java is composed today; generic installation/capability/config negotiation
remains separate work.

## Process transport

Core resolves `SCRYER_ENGINE_JAVA` or `scryer-java-engine` on PATH and invokes
`execute`. The engine's Go wrapper starts its JVM worker and forwards stdin,
stdout and stderr without rendering or rewriting JSON. Each invocation accepts
one request and returns one response. Diagnostics use stderr; framing is EOF,
not JSON Lines. Requests are limited to 1 MiB by the JVM worker. Paths are absolute
filesystem paths. Exit 0 means delivery, not necessarily successful analysis:
a structured engine error or failed target tests can be part of a valid response.

The separate `describe` operation returns exactly one identity with protocolVersion
1, id/name/version/source and mock false for the real Java engine. Core applies a
10-second describe deadline. Non-persistent management probes still return mock
receipts with persisted false; identity mock false describes the engine itself.

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

The abbreviated `report` above is not a valid complete report. An error response has `report: null` and `error: {"code": "JAVA_ANALYZER_ERROR", "message": "..."}`. Go rejects mismatched version/request/operation/stack, malformed reports, duplicate symbol IDs, graph edges referencing missing nodes and extra stdout documents. Analyze report refs must match the requested SHAs. Go cancellation terminates the shared wrapper/JVM process group and ordinary child build processes on POSIX, then removes isolated checkouts.

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
