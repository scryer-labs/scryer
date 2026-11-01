# Scryer contributor instructions

Follow the package responsibilities and readability guidelines in `ARCHITECTURE.md`. Prefer clear branches, named intermediate results and named arguments; retain lambdas and collection pipelines when they help readability.

Commit subjects use `<type>: <summary>`. Allowed types are:

- `feat`: new functionality
- `refactor`: structural/readability changes without intended behavior changes
- `tests`: test additions or changes (use `tests`, not `test`)
- `docs`: documentation changes
- `chore`: build, CI and repository maintenance
- `fix`: corrections to existing behavior
- `revert`: reverting an earlier change

Choose the type for the primary purpose of the commit. Keep summaries concise and specific. Do not rewrite existing history unless the user requests it.

## Development acceptance

Agreed 2026-10-31; configured 2026-11-01: use the Orrery **DevOps / Daily Development**
plan for acceptance. Workspace root is
`/Users/edwardnoaland/Desktop/experiment/scryer-lab`; plan ID is
`scryer-lab--daily-development`. Read its manifest for the current revision before
starting a run through Orrery's API. Checks follow `.github/workflows/ci.yml`, with
local macOS toolchain paths replacing hosted setup actions. Packaging checks remain
local; GitHub artifact upload is a hosted workflow concern.

Inspect the recorded results so Edward can review the same run in Orrery. Do not
substitute separately executed local test/build commands or modify the plan's
commands/scope without agreement. Read-only inspection and documentation checks
remain appropriate. Report exactly which checks ran and any missing acceptance
evidence; documentation checks are not code-pipeline acceptance.

The current plan assumes the combined Go CLI/Kotlin Java-engine repository layout.
When CLI and language engines are split into independent repositories, update the
plan targets, toolchains, build/test/package steps and cross-repository compatibility
checks together with the corresponding CI workflows. Do not treat today's plan as
permanent acceptance coverage for the extracted architecture.
