# Scryer contributor instructions

Follow the package responsibilities and readability guidelines in `docs/ARCHITECTURE.md`. Prefer clear branches, named intermediate results and named arguments; retain lambdas and collection pipelines when they help readability.

Commit subjects use `<type>: <summary>`. Allowed types are:

- `feat`: new functionality
- `refactor`: structural/readability changes without intended behavior changes
- `tests`: test additions or changes (use `tests`, not `test`)
- `docs`: documentation changes
- `chore`: build, CI and repository maintenance
- `fix`: corrections to existing behavior
- `revert`: reverting an earlier change

Choose the type for the primary purpose of the commit. Keep summaries concise and specific. Do not rewrite existing history unless the user requests it.

## Refactoring design reference

Read [decision 004](docs/decisions/004-multilanguage-scopes-cli-and-engine-config.md)
for the agreed multilanguage domain, scope-first commands, minimal engine YAML
(no pre-stability compatibility work), three JSON result layers and modular Core
reporting before the next refactoring slice. It separates agreed behavior from
remaining open schema/flag details; it is not a claim of implemented capability.

## Development acceptance

Agreed 2026-10-31; configured 2026-11-01: use the Orrery **DevOps / Daily Development**
plan for acceptance. Workspace root is
`/Users/edwardnoaland/Desktop/experiment/scryer-lab`; plan ID is
`scryer-lab--daily-development`. Read its manifest for the current revision before
starting a run through Orrery's API. As of the 2026-11-02 workspace relocation,
this definition targets the Scryer project (not the workspace); its commands run at
that project root without a `cd scryer` prefix. The start request still uses the
scryer-lab workspace root. Resolve/read the current definition rather than restoring
old workspace-scoped commands. Checks follow `.github/workflows/ci.yml`, with
local macOS toolchain paths replacing hosted setup actions. Packaging checks remain
local; GitHub artifact upload is a hosted workflow concern.

Inspect the recorded results so Edward can review the same run in Orrery. Do not
substitute separately executed local test/build commands or modify the plan's
commands/scope without agreement. Read-only inspection and documentation checks
remain appropriate. Report exactly which checks ran and any missing acceptance
evidence; documentation checks are not code-pipeline acceptance.

Routine acceptance also includes the workspace integration plan
`modernization--scryer-orrery-integration`, rooted at
`/Users/edwardnoaland/Desktop/ThoughtWorksProjects/modernization/test_workspace`.
Run Scryer's plan first, then this integration plan. It runs seven project scans in
parallel through Scryer v1.4 and retains JSON, without a verification script or Mod.
If Orrery/provider implementation changes, also run `orrery--daily-dev-pipeline` in
scryer-lab; otherwise do not add Orrery's full pipeline to every Scryer change.
Provider operations use their project target, not an arbitrary directory override.

The current plan assumes the combined Go CLI/Kotlin Java-engine repository layout.
When CLI and language engines are split into independent repositories, update the
plan targets, toolchains, build/test/package steps and cross-repository compatibility
checks together with the corresponding CI workflows. Do not treat today's plan as
permanent acceptance coverage for the extracted architecture.


For the initial independent mock-engine connection slice, the user explicitly
selects three acceptance plans: Scryer daily, scryer-java-engine--daily-development
(project), then scryer-lab--scryer-java-engine-integration (workspace), all rooted in
scryer-lab. No persisted engine installation or Java extraction is part of this
slice; do not add the old scanning integration or Orrery pipeline to it.
