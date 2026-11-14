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

## Scope and milestones

Use the [current documentation](docs/README.md) and [milestones](docs/MILESTONES.md).
[Decision 004](docs/decisions/004-multilanguage-scopes-cli-and-engine-config.md)
records the rationale for scope-first commands, engine configuration and result layers.

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

Run the Core plan first for Core changes. Run the affected engine's own plan
when changing its scanner or package, then the relevant workspace integration.
For official release installation and upgrade, use
`scryer-lab--all-release-installation` with an isolated Scryer home. For Java
worker or analyze changes, use `scryer-java-engine--daily-development` and
`scryer-lab--scryer-java-engine-integration`. The latter verifies Java snapshots,
execution and cancellation, and refreshes the default Java installation.

If Orrery/provider implementation changes, also run `orrery--daily-dev-pipeline`.
Provider operations use their resolved project target. Use
`modernization--scryer-orrery-integration` under the existing `test_workspace`
for changes to its active project scans. The retired CSV adapter remains
recorded under TD-006.

Keep engine tests independent of Core report resources and keep Kotlin/Gradle
in the Java engine repository. Avoid an implicit sibling-repository fallback
in Core. CI handles hosted artifact upload and release publication.
Global installation/config storage defaults to ~/.config/scryer on macOS as well,
matching Orrery/nvim; do not use os.UserConfigDir() or Library/Application Support.
SCRYER_HOME remains an explicit override. Project engine YAML is a separate layer.
