# Scryer Core working conventions

This file owns collaboration and acceptance rules. Product contracts, supported
facts and known limitations remain in the references below; this is not a roadmap.

## Responsibilities and references

Read [architecture](ARCHITECTURE.md), the [documentation index](README.md),
[contracts](contracts/README.md), [ABI](contracts/ENGINE-ABI.md) and
[project scope](contracts/PROJECT-SCOPE.md) as affected. Product scope and delivery
order remain in [milestones](MILESTONES.md), [product direction](PRODUCT-DIRECTION.md)
and [decision 004](decisions/004-multilanguage-scopes-cli-and-engine-config.md).

Core owns the CLI, project/workspace orchestration, engine discovery and package
lifecycle, C ABI loading/cancellation/buffer ownership, generic contract validation,
source selection, comparison/traversal, aggregation and reports. Language extraction
belongs in engines. Keep engine-specific report modules separate from generic
orchestration; retain complete engine JSON and its evidence limits. Keep `main`
and command composition thin. Avoid implicit sibling-repository fallbacks.

Keep Kotlin/Gradle in the Java engine and engine tests independent of Core report
resources. CI owns hosted artifact upload/publication. Installation/config storage
defaults to `~/.config/scryer` on macOS too; do not substitute `os.UserConfigDir()`
or Library/Application Support. `SCRYER_HOME` is an explicit override, distinct
from project engine YAML.

## Coding and commit conventions

Use clear names/branches, named intermediate results and readable boundaries.
Prefer named arguments where useful; keep lambdas/pipelines when they improve
readability. Keep code comments in English. Add meaningful regression tests for
new behavior rather than tests that merely repeat the implementation.
Go tests use `Test_should_xxx_when_xxx` where applicable. Commit subjects use
`<type>: <summary>` with `feat`, `fix`, `refactor`, `tests` (not `test`), `docs`,
`chore` or `revert`. Choose the primary purpose, keep summaries concise, preserve
unrelated work and do not rewrite history without explicit user instruction.

## Workflow triggers

Classify the candidate by its actual effect, not its filename or directory.
The full analysis → coding → independent review workflow applies when changes
affect repository build, test, execution or runtime behavior, including:

- implementation, tests or test data;
- executable scripts, dependencies, build or CI configuration;
- scan/provider plans, executable configuration or machine-consumed contracts;
- executable or generation logic located inside documentation directories.

Mixed behavior-changing and prose changes use the full workflow. Coding and the
final independent reviewer run the existing complete acceptance entries for the
final candidate. The two independent source/scan tracks below remain required
for that path. Do not treat every configuration file as lightweight.

Ordinary prose, `AGENTS.md`, subagent role descriptions/prompts and working
agreements, and other content without repository executable-behavior changes use
the [lightweight path](#lightweight-path). Role TOML prompt wording alone is not a
product implementation change. A `.md`, `.toml`, `.yaml` or `.json` suffix does
not decide the path. Inspect ambiguous effects before classifying; if executable
behavior is discovered, promote the candidate to the full workflow. Report the
classification and checks actually performed.

## Sequential subagent workflow

For candidates that trigger the full workflow above, use the following stages.
Lightweight changes do not require these subagents, scans or complete plans.

The primary agent owns user discussion, requirement boundaries, coordination,
questions, ownership assignments and the final report. It coordinates the roles
below in order. These instructions are an orchestration convention, not an
automatically enforced pipeline. Do not parallelize dependent phases.

Start each role with a short handoff. Use a fresh agent without the full chat
history when the client supports it. Give it the current requirement and
constraints, its role and ownership, the candidate identity, the relevant source
and contract paths, the acceptance entries, and the findings or failures to
resolve. Link to retained evidence and include exact run IDs. Quote only the
relevant log lines. Let the agent read the diff, code and full records as needed.
Resume the same agent after an interruption when possible. If it needs replacing,
pass the last frozen candidate and pending checks rather than replaying the chat.
Reviewers still form their own source inventory and verify the acceptance evidence.

1. **Analysis** reads the current handoff, source, Git state and applicable
   contracts. Stay read-only: do not modify implementation or commit. Hand off the
   problem, scope, constraints, affected repositories/boundaries, acceptance
   criteria, unresolved questions and an executable coding plan.
2. **Coding** implements that handoff within the authorized scope, adds meaningful
   regression tests and runs this repository's complete acceptance entry below.
   Deliver the diff, candidate identity, exact plan/run evidence and limitations.
   Do not commit before independent review, and do not push.
3. **Code review** is independent of coding. Build source-review observations and
   fresh Scryer facts in two independent tracks for the same candidate, then
   reconcile their factual inventories as below. Directly inspect implementation
   and tests against requirements, contracts, responsibilities and regression
   risks in every comparison branch. Send actionable findings to coding rather
   than silently rewriting implementation. After fixes, review again and refresh
   both affected tracks. Once review passes, rerun the complete acceptance entry
   against the final candidate. Only with user commit authorization and passing
   final review/acceptance, make small commits and report hashes, key changes,
   evidence and remaining limits. Explicit user authorization is separately
   required for push.

Shared workspaces need one writer per code area. Cross-repository work needs
explicit file/repository ownership and coordinated final integration acceptance.
Do not revert unrelated edits, rewrite history, change plans, restart user-owned
services or expand permissions implicitly. Do not overlap scans or complete plans
that clean/rebuild the same build/package directories or use the same installation
home. Read prerequisites and effects before scheduling them.

The root `AGENTS.md` is the persistent entry. Role definitions are in
`.codex/agents/`; shared settings are in `.codex/config.toml`. If custom roles are
not discoverable in the current client, the primary reads the relevant role file
and passes its instructions to an available subagent. If no subagents are
available, report the limitation; do not claim an independent review happened.
Do not pin a model, credentials, personal paths, permissions or runtime history in
shared role/settings files. Track these files; do not ignore all of `.codex/`.

## Candidate identity and evidence

At each handoff retain repository/branch/HEAD, staged and unstaged binary diffs,
relevant untracked file contents and hashes, scope/configuration and dependency
identities. HEAD alone does not identify a dirty candidate, and hashes alone
cannot restore its content. Bind review and acceptance to that candidate; any
later edit invalidates affected conclusions and requires refreshed evidence.
After an authorized commit, retain the resulting SHA. Keep raw facts, patches,
logs and run metadata in a task-specific local evidence directory outside the
reviewed repositories, not in shared role/configuration files or committed history.
A workspace-local `.orrery/reviews/` directory is one possible location.

## Reviewer source and scan tracks

This section applies only to full-workflow candidates. Lightweight changes do
not require a reviewer or Scryer scan.

Use two independent evidence tracks bound to the same candidate. Neither an
AI/source inventory nor scanner output is presumed correct. Self-scanning is
supporting evidence, not a correctness certificate for the scanner or its code.

1. **Source track**: Read the actual diff, relevant source, contracts and tests.
   Record a code-derived factual inventory, review findings, checked scope and
   supporting source references. Include definitions, component/file ownership,
   imports/references, calls, relationships, source positions and classifications.
   When the tracks run sequentially, record this inventory and findings before
   viewing this candidate's scan output. The scanner must never supply the sole
   inspection checklist or determine the initial source-review scope.
2. **Scan track**: Obtain the current registered project's facts through Orrery's
   normal project Scan API and effective managed workflow, never by executing the
   Scryer CLI directly. Discover relevant installed engine capabilities from
   recorded provider/engine metadata or package manifests; automatically detected
   implementation languages, not the repository's engine name, determine the
   applicable facts. Request the complete relevant supported fact set, including
   dependencies, architecture, files, call graphs and communication as applicable.
   Do not freeze review to a small fact whitelist or assume every engine supports
   every requested fact. Retain the published raw facts and provenance below.

Safe parallel tracks are optional; independence is required. If sequential,
record source observations before viewing this candidate's scan output. Do not
mix outputs before the independent inventories are recorded. Scans must not
conflict with acceptance tasks cleaning/rebuilding the same build/package paths
or installation homes. Do not install tools or restart user services implicitly.

### Registration and effective scanning configuration

Resolve the current workspace root from workspace registration or explicit
configuration. Match the reviewed repository's actual path to its registered
project and discover that project's `projectId`; do not copy Orrery's own project
UUID or guess from a repository name. Keep machine roots and UUIDs out of these
shared instructions. Use the configured Orrery API origin, not a fixed host.

Inspect the effective project configuration through
`GET /api/scanning/local?root=<url-encoded-workspace-root>&kind=project&projectId=<registered-project-id>`.
Preserve the returned local/effective definitions, configuration source and
revision before any change. Precedence is project override, workspace project
default, then user default; a selected definition replaces the inherited one.
Check actual YAML/provider bindings, not only effective control labels.

If a correction is necessary, report it to the primary and keep it within the
authorized project scope. Actual workflow changes affect execution and take the
full validation path. Use `PUT /api/scanning/local` with the same `root`,
`kind=project` and `projectId` query, and body `yaml`, `fact`, `currentRevisionId`.
For an existing local override, use its captured local revision; when creating
one with no local override, use null, not the inherited effective revision.
Retain before/after snapshots, reread the effective selection and verify the
saved result. On a revision conflict, reload and reconcile; never force a stale
snapshot over concurrent changes. Do not edit scanning files directly or change
workspace defaults, other projects or global configuration as a shortcut.

The workflow must request full relevant supported facts, declare a `mode`
parameter whose normal default may remain `static`, and bind that parameter to
the selected provider scan operation (`mode: {parameter: mode}`). Merely declaring
it is insufficient. Preserve `--compact` in the resolved provider. The workflow's
`fact` identifies the exact retained JSON output reference, not a hand-built
report filename. See Orrery's
[managed scanning contract](https://github.com/scryer-labs/orrery/blob/main/docs/SCANNING-PLANS.md)
and [scan publication API](https://github.com/scryer-labs/orrery/blob/main/docs/WORKSPACE-PROJECT-API.md).

### Required scan entry point and publication evidence

For full-workflow review, request this registered project through the normal API:

```http
POST /api/workspace/projects/{registeredProjectId}/scan
Content-Type: application/json

{"root":"<registered workspace root>","mode":"full","remoteList":false}
```

The reviewer explicitly requests `full`; ordinary UI scans may still default to
static. Do not invoke Scryer CLI commands directly as another scan path, manually
merge/construct a replacement facts document, or run only a Settings/plan-run
workflow and treat its success as published facts. The normal scan endpoint owns
execution, publication and history. No CLI fallback is allowed.

Require successful publication for this exact attempt, not just a successful
command or plan. Retain its returned scan ID, successful manifest, published
facts and original provider JSON (`source/facts.json` where retained), logs and
`plan-run.json`. Use that provenance to retain the configuration source/revision,
frozen definition and parameters, provider operation/version/behavior, plan-run
identity, executable/engine/package sources and versions or hashes, scope,
requested facts and actual per-engine outcomes. Preserve original raw JSON;
an architecture projection alone or a reconstructed aggregate is insufficient.
Do not select an older `latestSuccessful` or unrelated latest log silently when
the current attempt failed or another scan intervened.

Inspect actual frozen `mode` bindings, remote-list value and resolved provider
command/behavior to establish that `full` selected `--resolve` while preserving
`--compact`. The API response saying `mode=full` alone does not prove this. A full
request is not proof of complete semantic resolution: retain unsupported,
uncollected, partial, fallback, unresolved and parser/compiler diagnostics, and
distinguish source candidates, compiler evidence and runtime observations.

If service, registration, configuration, execution or publication fails, preserve
the concrete failure, mark scan verification incomplete and continue direct
source/test review. Do not restart user services, expand permissions, bypass the
API with CLI or another plan invocation, fabricate facts or silently reuse old
scans. After implementation fixes, refresh the corresponding successful published
scan and affected source inventory for the same final candidate. Keep local scan
configuration, UUIDs, machine paths and run artifacts in local evidence outside
source repositories; do not commit them as portable role defaults.

### Factual reconciliation

Compare factual inventories, not correctness verdicts. Verify discrepancies
against actual source, tests and applicable contracts, correcting whichever
observation is wrong. Keep the distinction between source candidates, compiler
bindings and runtime observations throughout reconciliation.

| Comparison | Required action |
| --- | --- |
| Both tracks agree | Record agreement within the checked scope, then continue normal source/requirements and test review. Matching inventories can share omissions. |
| Source-only fact | Verify the source observation and requested contract. Report a confirmed scan omission only when real code evidence establishes that the fact should have been collected. Otherwise correct the mistaken AI judgment, record a coverage limit, or leave it unconfirmed. |
| Scan-only fact | Verify it in code and tests. If correct, add it to the source-review scope and review the related paths; if disproved, report a confirmed scan false positive. Insufficient evidence remains unconfirmed. |
| Ownership, target, direction, classification or position differs | Trace code and contract evidence, correct the wrong side and retain the evidence explaining the discrepancy. Neither track wins by default. |
| Evidence is insufficient | Retain an unconfirmed item and its reason; do not force agreement or label an unknown as a confirmed scanner error. |

Inspect actual code and tests in every branch, including relevant paths missing
from or unresolved by the scan. Iterate the expanded review scope and
reconciliation until differences are explained or explicitly unknown. Never
invent an edge or turn a scanner defect into an application defect. On scan
failure, continue source review and state that scan-to-source verification is
incomplete; failure cannot establish agreement.

Retain the checked scope, source references, matched observations and discrepancy
log with fact identifiers where available. Keep five report buckets separate:

- **Code findings**: independent requirement, contract, responsibility and
  regression findings from direct source/test review.
- **Confirmed scanner omissions/false positives**: source-backed differences
  under the requested contract, not merely missing or unresolved facts.
- **Corrected AI judgments**: mistaken source observations corrected using real
  code and contract evidence.
- **Unconfirmed items and coverage limits**: unsupported, uncollected, partial,
  fallback, unresolved, failed scans and other insufficient evidence, with reasons.
- **Final candidate acceptance**: exact candidate identity, final review scope and
  validation/plan outcomes, independently of scanner accuracy.

If no discrepancy is found, say **“在已核对范围内一致”** and list that scope; do not
claim whole-repository correctness. Agreement can hide shared omissions.
Confidence comes from verified scope and repeatable regression evidence, not
repeated matching verdicts. Confirmed scanner or projection defects whose fixes
are authorized and in scope require a minimal reproducible example and meaningful
regression coverage. Report out-of-scope or cross-repository follow-ups without
silently expanding implementation ownership.

After fixes, refresh both affected source and scan tracks against the final
candidate and reconcile again. Preserve raw facts and provenance for handoff.
Impact analysis is not an additional reviewer duty yet. This does not waive
impact regression steps already included in an established complete acceptance
plan.

## Running acceptance plans

Use the existing Orrery plan-run API, not separately executed local test/build
commands as a substitute. Resolve the workspace root from current registration
or explicit configuration; do not assume every checkout's parent is its workspace.
Find this repository's registered project and effective plan target at runtime.

Read `.orrery/plans/definitions/<plan-id>/manifest.json`, its current revision and
frozen `revisions/<revision-id>/plan.yaml`. Verify active state, target, checks,
prerequisites and side effects. Invoke the configured Orrery server with
`POST /api/workspace/plan-definitions/<plan-id>/runs` and JSON
`{ "root": workspace_root, "revisionId": current_revision }`.
Retain `rootRunId`, frozen definition/effective target, step results, logs and
artifacts together with candidate identity. Inspect which checks actually ran;
a green summary or build alone does not prove tests/vet executed. Local platform
and toolchains can differ from hosted CI.

If registration, a required plan, prerequisite or server is unavailable, report
missing acceptance evidence. Do not invent a replacement plan, change its scope,
start services or silently substitute ad hoc tests. Apply the user's existing
authorization to known effects; request clarification only for a necessary effect
that is not already authorized. Ordinary read-only inspection and formatting are
allowed. Release publication remains separate from local package verification.

## Lightweight path

For the non-executable changes defined in Workflow triggers, the primary agent
may make the edits directly. Check content consistency, document references,
relevant configuration syntax and diff formatting (`git diff --check`). Confirm
the diff contains no implementation or executable-configuration change, preserve
unrelated work and report the actual changed files and checks.

Do not start a reviewer, Scryer self-scan, full acceptance pipeline or mandatory
three-agent sequence solely for these edits. No independent review or source/scan
reconciliation is required for this lightweight path. With user authorization,
commit and push after the required lightweight checks pass; do not invent a full
pipeline verdict for those checks.

This is not an exception for all configuration files. Changes to actual commands,
dependencies, tests/fixtures, build/package/CI logic, scan/provider plans, runtime
configuration, machine-consumed contracts or generated/executed behavior require
the full workflow when they affect repository behavior, even in a documentation
directory. Mixed candidates use the full path.

## Repository acceptance entries

Both coding and final independent review use the following complete entries for
changes that trigger the full workflow. Review reruns acceptance on its final accepted candidate.

| Entry | Existing plan ID | Effective target |
| --- | --- | --- |
| Full repository checks | `scryer-lab--daily-development` | project |

The repository entry currently covers: Go race tests and vet; report graph/browser tests and JS syntax checks; standalone CLI build, Core packaging, extracted-package checks and standalone Go installation.
Read the current frozen definition rather than copying these summaries into
commands or assuming they remain exhaustive. Hosted checks are described in
[CI](../.github/workflows/ci.yml); hosted upload/publication is not a local step.

Despite its workspace-prefixed ID, the Core daily plan targets the registered
Core project: commands run at that project root without a `cd scryer` prefix.
The API request still supplies the resolved workspace root. Run Core first when
Core changes, then each affected engine's full checks and relevant workspace
integration. Existing integration IDs by engine:

| Engine | Full engine entry | Workspace integration |
| --- | --- | --- |
| Java | `scryer-java-engine--daily-development` (project) | `scryer-lab--scryer-java-engine-integration` |
| C++ | `scryer-lab--cpp-impact` (workspace) | `scryer-lab--scryer-cpp-engine-integration` |
| Rust | `scryer-rust-engine--daily-development` (project) | `scryer-lab--rust-integration` |
| TS/JS | `scryer-ts-js-engine--daily-development` (project) | `scryer-lab--ts-js-integration` |
| Go | `scryer-go-engine--daily-development` (project) | `scryer-lab--go-engine-integration` |

There is no dedicated C++ daily project plan; its full entry includes all
configured impact, coverage and reflection steps. Consult the affected engine's
working notes and current integration effects. Java integration's `activate-default`
step unsets `SCRYER_HOME` and replaces the user's global Java installation; check
that this exact effect is within the current task authorization. Go's workspace
integration consumes all engine packages, including the Java bundled-runtime
archive: verify prerequisite package identities, not merely their existence.

For official release installation/upgrade use `scryer-lab--all-release-installation`
with its isolated installation home. Source-selection changes also use
`scryer-lab--project-scope`; choose semantic, Tauri or analysis integrations when
those boundaries change. If Orrery/provider implementation changes, also use its
own `orrery--daily-dev-pipeline` and load the rebuilt provider catalog through the
user's service lifecycle. For changes to still-active external project scans,
resolve their owning workspace and plan; the existing
`modernization--scryer-orrery-integration` applies only to its registered active
scans. The retired CSV adapter remains recorded in TD-006, not restored here.
