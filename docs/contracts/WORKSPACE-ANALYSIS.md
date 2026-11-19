# Workspace analysis v1

`workspace analyze` reads the same `.scryer/workspace.yaml` as workspace scan
and invokes project analysis. This is Core orchestration; no engine ABI changes.
Membership is an ordered list of workspace-relative paths. Existing scan
validation for duplicates, traversal and symlink escapes is reused.

The result has `schemaVersion: 1`, canonical `root`, `operations[]`, `resolve`,
`status`, `counts`, `projects[]` and `notes[]`. Status is `collected` when every
member succeeds, `partial` when some succeed, and `failed` when none succeed.
Counts always include `collected`, `failed` and `unsupported`.

Each project has:

- `path`: configured member path, normalized relative to the workspace.
- `status`: collected, failed or unsupported.
- `error`: diagnostic for failed/unsupported results.
- `engine`: report dispatch identity when available (possibly `multiple`).
- `reportOperation`: `analyze` for impact envelopes or `coverage` for standalone
  coverage envelopes; this selects the existing project renderer.
- `report`: complete validated project JSON, including unknown engine fields.
  Failed dynamic analysis does not attach a static replacement. A valid legacy
  execution-failure or partial coverage report can be retained with failed status.

No matching installed capability yields unsupported. Invalid repos/baselines,
failed runtime execution, missing evidence or malformed responses yield failed.
Other members continue. Top-level configuration errors and cancellation stop
execution without an authoritative aggregate. Incomplete results are emitted
before CLI exit 1 so consumers can retain successful project evidence.

Impact baselines are chosen independently by each project. The workspace has
no before/after flags and creates no combined revision identity. Each child
retains its own before/after SHAs and optional working-tree provenance. Coverage
alone uses the current project trees without requiring comparison commits.

Static impact is the default. `--resolve` enables project dynamic analysis;
`impact,coverage` requires it. Coverage alone is already runtime collection and
rejects `--resolve`. Test selection currently uses project-native defaults;
per-member Engine YAML initialization/parameterization remains a separate concern.

JSON, Markdown and HTML use the same aggregate. Markdown retains full JSON as
well as rendered project sections; HTML embeds independently rendered project
reports in sandboxed frames and includes a complete JSON download. Symbol IDs
and coverage percentages stay scoped to their project report. Library/IPC
relationships are not evidence of inter-project change impact propagation.
