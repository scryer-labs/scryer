# Runtime coverage v1

Standalone project coverage reads an explicitly provided artifact against
the current project directory. The CLI chooses an installed engine that
advertises `coverage`. The engine request uses `operation: "coverage"` and
`coverage: {repository, artifact}`, with absolute paths. The ABI functions
and request/response version remain unchanged.

The engine returns `schemaVersion`, `root` and `coverage`. The project
report adds `engine` and `status` and preserves the JSON evidence:

- `format`, `version`, `artifact` and `sha256` identify the input.
- `sourceMatch: "unverified"` records that the imported C++ artifact cannot
  establish a source snapshot. An artifact digest identifies the artifact,
  and does not identify the source revision.
- `granularity: "aggregate"` describes the observed profile. No per-test
  identity is known.
- `files[]` contains repository-relative `path` and line, function and
  region counts as `{covered, count}`. An optional `llvmSegments` contains
  original LLVM segment tuples. Consumers should use LLVM counts for the
  file totals.
- `notes[]` records missing provenance or paths that could not be mapped.

A missing input or no mapped files yields `status: "unavailable"`. Invalid
inputs and failed collectors return an error. CLI JSON, Markdown and HTML
all use the same project report. Future impact analysis can call the same
coverage use case, but must verify evidence against its selected after
snapshot before using it to label impacted code.
