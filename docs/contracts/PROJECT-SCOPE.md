# Project file selection

Core owns project discovery and file selection. Engines own fact semantics.
The first implementation supports Core stack/LOC and the C++ Engine's C/C++
and CMake collectors. This policy selects source observations; it does not
classify code as production, tests, generated or third-party.

## Project configuration

Optional `.scryer/project.yaml`, relative to the project root:

```yaml
schemaVersion: 1
scan:
  include:
    - CMakeLists.txt
    - cmake/**
    - src/**
    - include/**
    - tests/**
  exclude:
    - src/generated/**
    - tests/fixtures/**
```

Patterns use `/` and are relative to the project root. `**` matches zero or
more complete path segments; `*`, `?` and character classes operate within
one segment. For example, `**/*.cpp` also matches a root-level `.cpp` file.
Use `vendor/**` for a directory subtree, not `vendor/`. Absolute paths,
parent traversal, malformed patterns and unknown configuration keys fail.
The configuration size limit is 1 MiB.

Omitting `include` selects `**/*`; an explicit `include: []` selects nothing.
Exclusion wins over inclusion. Omitted `exclude` is empty. Regular files only:
symlink entries are not scanned. In Git projects the candidate inventory is
tracked files plus non-ignored untracked files (`git ls-files`). Deleted tracked
files are skipped; ignored untracked files cannot be restored by an include
pattern. Without Git, Core walks the directory, excluding Git metadata.
Counts describe regular candidate files, before language/binary/size checks.
They are not LOC file counts or completeness scores.

No configuration preserves the existing default behavior, including existing
Engine traversal rules. With configuration, Core hands its exact selected list
to scope-aware Engines. CMake and C/C++ consume that same list independently;
include desired build files explicitly when narrowing `include`. Excluding a
header or CMake file can leave references or component membership unresolved.
Literal references from selected files are retained even when their destination
is outside the selection. Exclusion does not remove compiler include paths,
change CMake targets, sandbox configure scripts or stop libclang from reading
headers as context. Compiler observations originate in selected source files.

Workspace scans delegate to each project's configuration. There is no workspace
scope override or automatic classification based on directory names.

## Consumer JSON additions

Configured project reports add an optional top-level `scope`:

```json
{
  "scope": {
    "schemaVersion": 1,
    "config": ".scryer/project.yaml",
    "configSha256": "sha256-of-config-bytes",
    "base": "git-tracked-and-nonignored",
    "include": ["CMakeLists.txt", "src/**"],
    "exclude": ["src/generated/**"],
    "considered": 100,
    "included": 40,
    "excluded": 60,
    "selectionSha256": "sha256-of-manifest-bytes"
  }
}
```

`base` is `git-tracked-and-nonignored` or `filesystem`. The selection digest
includes the absolute root and path list; it is a dispatch receipt, not a source
content hash or a portable project identity. Do not use it as a cache key for
code content. No `scope` means the previous default behavior, not an error.
Existing `engineFacts[].data`, component IDs, file paths, symbol IDs and relation
fields retain their shapes. Workspace child reports retain `scope` normally.
Markdown and terminal output describe the selected scope.

Consumers such as Orrery should display scope/counts as evidence boundaries,
preserve the metadata, and avoid presenting an empty selected graph as a full
repository scan. They need no new C++ architecture/files/call-graph adapter.

## Core → Engine JSON protocol

A scope-aware package declares `capabilities.scanFileSelection: true` in its
manifest. This is an additive JSON protocol feature; C ABI v1 and `execute`
symbols do not change. Core adds `scan.fileSelection`:

```json
{"path":"/temporary/selection.json","sha256":"digest","fileCount":2}
```

That UTF-8 JSON file contains:

```json
{"schemaVersion":1,"root":"/absolute/project","files":["CMakeLists.txt","src/app.cpp"]}
```

Core writes sorted, unique project-relative paths and keeps the temporary file
alive for the request. Empty selection is `files: []`, never a fallback to a
full walk. Engines validate SHA-256, version, root, count and relative paths.
The current C++ reader limits manifests to 64 MiB, anchors source reads to the
project root and rejects traversal, duplicates and Git metadata. Core removes
the temporary manifest after execution, including failures and cancellation.
Only paths are externalized; no report data or source history is persisted.

The Engine scan report returns `selection: {"sha256":"digest","fileCount":2}`.
Core checks both fields before accepting facts. This receipt is internal to
Core/Engine exchange; consumer-facing project reports expose `scope` instead.
An older Engine lacking the capability returns `unsupported` for configured
facts without execution. A missing/mismatched receipt is `failed`. Core never
silently sends a configured scope to an incompatible Engine. Rust, TS/JS, Go
and Java do not yet implement this selection protocol. Their unconfigured
behavior is unchanged. Legacy Java scan flags reject project selection; use
the facts path and a scope-aware Engine.

## Impact and runtime coverage

Impact pins the **current original project's configuration** across before and
after; each snapshot gets its own inventory. Historical copies of project.yaml
are not interpreted independently. This prevents a config-only change from
looking like symbol deletion. Requests add `impact.beforeSelection` and
`impact.afterSelection` with the same reference shape; Engine snapshot results
acknowledge each as `before.selection` and `after.selection`.

The final analyze report adds `scope: {"policy":"current-project-config",
"before":{...},"after":{...}}` with the same ScanScope shape per snapshot.
Changes and graph traversal use selected Engine projections. Excluded files
can still affect selected code indirectly; the filtered graph is a bounded
view, not proof of absence of impact.

`scan.include/exclude` does **not** filter coverage instrumentation, build or
test execution. Coverage remains the full native collection, and impact matches
that evidence against selected symbols. Test-target selection continues to
control which tests run. Standalone `--operations coverage` deliberately ignores
scan selection. Static/dynamic evidence and partial/failure semantics stay as
specified in [impact](IMPACT.md) and [coverage](COVERAGE.md).

## Follow-up

User-defined source groups, third-party/generated labels and test subgroups are
not implemented. Future labels must retain their configuration provenance and
remain separate from deterministic Engine test-framework evidence. CMake
variable evaluation/build-model completeness and ownership/lifetime analysis
are separate work.
