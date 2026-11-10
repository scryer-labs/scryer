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

After the 2026-11-02 Java extraction, acceptance is ordered:

1. Scryer: `scryer-lab--daily-development` (Go/report tests, standalone build/install,
   Core-only packages; no Java or Gradle setup).
2. Engine: `scryer-java-engine--daily-development` (Go wrapper, Kotlin facts/JSON,
   JVM worker and slim/bundled-Java-21 package checks).
3. Workspace: `scryer-lab--scryer-java-engine-integration` (persistent local-package
   lifecycle in an isolated home, source independence, analyze/test execution,
   snapshots/cancellation, then installing the current bundled engine for normal use).
4. Modernization: `modernization--scryer-orrery-integration`, rooted at
   `/Users/edwardnoaland/Desktop/ThoughtWorksProjects/modernization/test_workspace`.
   Its seven project plans run in parallel through the formal scryer@v1.5 provider.
   Keep seven-project scans here, not duplicated in scryer-lab.

The first three definitions live under scryer-lab. Integration installs a known
complete artifact, then normal scans discover it without executable overrides.
Its final step refreshes the owned default Java installation from the current
bundled package so the modernization provider can consume it without a restart. Never add Core
checkout/report-resource dependencies to engine tests. No Kotlin/Gradle files
belong in this repository.

If Orrery/provider implementation changes, also run `orrery--daily-dev-pipeline`.
Provider operations use their resolved project target, not an arbitrary directory
override. The existing `modernization--scryer-orrery-integration` in test_workspace
is the provider acceptance plan; provider processes discover the persisted
installation automatically. Do not add an implicit sibling-repository fallback to Core.

Keep each repository's plan aligned with its CI workflow when build ownership
changes. Packaging checks are local acceptance; uploading artifacts is CI-only.
The installed Java package declares a C ABI v1 library. The workspace cancellation check verifies Core owns the JVM child and that cancelling the ABI request terminates its process group.

Global installation/config storage defaults to ~/.config/scryer on macOS as well,
matching Orrery/nvim; do not use os.UserConfigDir() or Library/Application Support.
SCRYER_HOME remains an explicit override. Project engine YAML is a separate layer.
