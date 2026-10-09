# Analyzer contract v1

Go owns the public commands, repository/snapshot lifecycle and report rendering. An analyzer supplies facts through `internal/contract.Analyzer`; no analyzer receives color, verbosity or output-file parameters. Only Java is registered initially. A C++ analyzer may implement this interface in Go directly or use a separate process adapter.

## Process transport

The Java adapter starts one JVM per operation, sends one UTF-8 JSON request to stdin and closes it, then expects exactly one JSON response on stdout. Progress/diagnostics use stderr. JSON framing is EOF, not JSON Lines. Requests are limited to 1 MiB by the JVM worker. Paths are absolute filesystem paths, never file URIs. Exit status 0 means a response was delivered; the response can still contain a structured analyzer error. Target-test failure is a valid analysis result, not a transport error.

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

The abbreviated `report` above is not a valid complete report. An error response has `report: null` and `error: {"code": "JAVA_ANALYZER_ERROR", "message": "..."}`. Go rejects mismatched version/request/operation/stack, malformed reports, duplicate symbol IDs, graph edges referencing missing nodes and extra stdout documents. Analyze report refs must match the requested SHAs. Go cancellation terminates the JVM process group and child build processes on POSIX, then removes isolated checkouts.

## Report fields

The report schema is versioned separately from the process protocol. Version 1 preserves the existing public JSON facts and uncertainty semantics. Required field shapes are checked in `internal/contract/document.go`; additional fields remain intact in JSON/HTML exports. Renderers consume documents in Go, while the analyzer's Java report projection supplies the compatible data structure. `map[string]any` in the renderer is a view over a validated document, not permission to invent evidence.

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
