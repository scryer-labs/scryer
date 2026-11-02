# Work decision 004: Multilanguage scopes, commands and engine configuration

Date: 2026-11-02
Status: Agreed design direction and core command behavior; not implemented.
Some flags, schemas and output defaults remain open as listed below.

## Authority and superseded agreements

This records the 2026-11-01/02 design discussion so future work does not reconstruct
requirements from chat. It refines decisions [002](002-scan-scopes-and-fact-boundary.md)
and [003](003-stack-discovery-and-shared-rules.md). Current implemented commands and
protocol remain in [CLI](../CLI.md) and [contracts](../contracts/README.md).
[Milestones](../MILESTONES.md) remains the only delivery-order/status index.

Changes from earlier plans: workspace membership gets CLI management rather than
requiring hand editing; first engine use initializes missing project configuration;
install accepts explicit batches, while stack-driven auto-install and install-all
are deferred; the temporary CSV adapter migrates into the new workspace command
shape before eventual removal. Java validates the engine mechanism, not every
future capability. New decisions replace earlier conflicting roadmap statements.

## Domain and responsibility layers

- **Project**: one repository, naturally multilanguage and potentially containing
  several build systems, modules and products. It is not one language/engine.
- **Workspace**: the highest required scan scope, with explicit project membership.
  It orchestrates project operations, aggregates facts and computes supported
  cross-project relationships from project facts. Do not invent a scope above it.
- **Build units and code entities**: targets, crates/packages/components, directories,
  files, types/functions and other symbols. These are not registered projects.
  A file can belong to several targets; declaration/definition and call relations
  form graphs rather than a mandatory single containment tree.
- **Engine**: an installed fact producer with language/build/platform semantics and
  declared capabilities. Projects can invoke several engines; engine and language
  need not be one-to-one. Core owns discovery, orchestration and standard outcomes.
- **Fact/evidence versus finding**: distinguish observations and provenance from
  results of explicit shared rules. Do not infer intended architecture, test adequacy
  or runtime completeness from folder names, coverage counts or static call graphs.

Core must not hardcode Maven/Gradle or implement language resolution. Engines supply
language facts and capabilities; shared cross-language relationship/rule logic uses
those facts without duplicating the same algorithms in each engine repository.
Language-specific extraction/adapters are still necessary. General shared-rule JSON/
YAML syntax waits for concrete cases; configurable mappings alone do not provide
missing semantic analysis.

Scryer remains a lightweight standalone provider, not a history database, centralized
engineering platform, plan executor or automatic refactoring tool. Orrery/people
consume its reports. Deterministic code-smell and bounded test-effectiveness checks
are permitted future capabilities; consultant-style business/design advice is not.

## Real design and acceptance cases

- Scriptorium workspace: brush, inkstone and felt use CMake/C++; ink uses Tauri
  with Rust, Svelte and TypeScript; cli uses Go.
- Orrery project: Rust backend and TypeScript/React frontend in one repository;
  REST request/route facts are a representative relationship-matching case.
- Scriptorium communication: C++/C++ and C++/Rust named-pipe relationships.
- shared_memory_exp: Windows shared-memory relationships, deferred under the
  macOS-first platform agreement.

Multilanguage aggregation comes before complete cross-language linking. REST is
first among network mechanisms; GraphQL/RPC wait. Cross-project computation uses
available project facts and identifies gaps rather than requiring every project to
fully succeed. Protocol/name equality alone does not prove communication. Engine
facts should retain endpoints/types/protocol/evidence; the shared relationship layer
matches them. Exact adapters/metadata are decided from these real cases later.

## Command shape and path behavior

The top-level entries are `project`, `workspace` and `engine`. The first two are
scopes; engine is local tool management. Paths are optional and default to the
current directory as scope root; an explicit path selects a different root.
Do not silently search for a different scope or guess workspace membership.

```sh
scryer project scan
scryer project scan ./orrery --facts architecture
scryer workspace scan --facts stack,loc
scryer workspace init
scryer workspace add ./orrery
scryer workspace remove ./orrery
scryer project analyze --before HEAD --after .
scryer engine install cpp rust ts
```

Workspace init creates minimal `.scryer/` membership configuration. Add registers
an existing project using a workspace-relative path, without cloning; duplicate
paths do not create duplicate members. Remove unregisters only, without deleting
code, outputs or engines. Support `.` as a member when a root is also a project.
Missing workspace configuration produces a clear diagnostic. Exact init/add/remove
flags and path canonicalization rules still need contracts.

## Scan facts, scope selection and filtering

Without explicit selections or configured overrides, scan defaults to **stack,loc**.
This default reads local source/declarations without running build configuration,
tests or remote queries. Basic stack discovery must work without language engines.
LOC counting/exclusion rules and the Core/engine implementation split remain open.

User-facing fact selection focuses on useful observations, not parser jargon:

| Facts | Meaning |
| --- | --- |
| `stack` | Languages, frameworks, build tools/toolchains and relevant versions/signals. |
| `loc` | Code/comment/blank-line metrics by supported source/language scope; exact counting policy pending. |
| `dependencies` | Declared/resolved external/internal package/library/build-unit dependencies, versions and boundaries. |
| `architecture` | Supported entities, ownership/grouping, declaration/implementation, reference and call relationships, with source evidence. |
| `test-coverage` | Existing test execution coverage evidence with freshness/source-state limitations; does not run tests implicitly. |

Existing static test inventory/configuration facts must also survive migration;
`testing` is a candidate category, not a finalized spelling. Symbols/calls are
internal fact concepts and part of architecture, not required separate user commands.
Future code-smell/test-rule/branch operations originate at project scope; workspace
may orchestrate and aggregate them without reimplementing their logic. Branch
strategy is the lowest-priority capability.

`--facts` requests collection, not full collection followed by hiding fields. Engines
may need supporting facts to answer it. Workspace forwards a request to member
projects and aggregates results with project identity; relationships across projects
are a separate computation over those facts. Distinguish collection capabilities,
selected projects/files, and output filtering. Missing collection, unsupported facts,
partial resolution and filtered-out data must not be confused. Exact project/file/
relationship filters and traversal limits are not finalized.

## Analyze and language-specific execution

Analyze is a separate operation, not `scan --analyze`. It compares explicit before/
after snapshots, identifies changed symbols, computes supported caller/callee
reachability, runs the configured after-snapshot full test lifecycle and matches
fresh test/coverage evidence against the relevant symbols. Preserve both snapshots,
removed entities, unresolved cases and independent evidence datasets.

Source comparison does not intrinsically require a successful build. Enhanced
resolution and execution evidence can require build metadata, compilation and tests.
Static reachability is not proof of an end-to-end runtime path; passing tests and
coverage hits do not prove assertion sufficiency. Full tests means the configured
lifecycle's scope, not a claim to discover every possible test.

Engines own build/test command selection, build-model extraction, remote dependency/
tool-version lookups, target runtime and evidence adapters. Core owns generic
requests, execution/network policy, cancellation, identity validation and reporting.
Project engine configuration supplies language-specific choices; CLI must not
centrally enumerate Maven/Gradle. Preserve current Java options/capabilities through
a deliberate migration; exact generic override flags remain open.

## Project engine configuration lifecycle

Workspace `.scryer/` owns members and shared scan defaults, not engine sections.
Project `.scryer/` holds scan defaults and explicit per-engine configuration files,
including build/test commands and engine options. Files are manually editable;
JSON versus YAML, exact names/schema and precedence are not yet finalized.
A candidate layout is `.scryer/engines/<engine-id>.yaml`, not an implemented format.

1. Workspace scan discovers stacks; the user explicitly installs missing engines.
2. When a requested project scan/analyze needs an engine, use existing engine config.
3. If missing, the engine probes the project and supplies an applicable default
   template; Core writes that section/file without overwriting another writer.
4. Reuse user-edited config on subsequent operations. Do not reselect or overwrite
   it silently on scan or engine upgrade. Report incompatible config explicitly.

First-use initialization during scanning is an agreed exception to read-only scan
behavior: report created files in terminal/JSON. Engines provide templates/probes;
Core owns filesystem writes and preservation. Config existence does not bypass
operation execution/network policy. Defaults must not invent arbitrary commands.
No mandatory init-engine-config command or interactive switcher is required.

For Java defaults: Maven-only selects Maven; Gradle-only selects Gradle; both select
Gradle with a reported choice. Existing config always wins. If neither is found,
retain supported source facts and report missing build/test configuration when
needed. Users manually change the generated config to select Maven if desired.
This replaces the earlier suggestion to require a choice whenever both tools exist.

## Engine management and unsupported outcomes

Provide installed list, remote available list, install and uninstall. Install accepts
an explicit batch of engines and reports each outcome. No install-all and no
project/workspace-driven auto-install in this slice. The example uses space-separated
IDs; exact parsing, remote-list spelling, versions and selection policy remain open.
Remote engine inventory is separate from remote dependency/tool-version queries.

Engine capabilities let Core consistently distinguish unsupported, not installed,
incompatible, failed and partial outcomes. Engines do not invent user-facing
unsupported command messages. Unsupported Java capabilities do not block other
engines' valid results or imply that no relationships exist.

## Reports and preservation requirements

JSON contains all facts actually collected in the requested scope, including entity
identities, relationships, provenance and missing/unsupported/partial limitations.
Markdown is information-equivalent rather than a truncated summary. HTML uses the
same data; collapse/filter/layout affects presentation only. Complete output does
not claim complete understanding of the source.

Interactive architecture HTML, possible embedding/reuse by Orrery and layout
selection remain a separate design topic. Clean-architecture layer mappings need
configuration/evidence; choosing a circular layout does not establish architecture.
Output flags, default stdout/file delivery and exit-code policy remain open.

## Migration and next implementation boundary

Preserve current Java scan/analyze features, source/runtime separation, cancellation,
snapshot isolation, retained artifacts, unknowns and protocol/report identity checks.
Use Java to prove independent engine config/install/discovery/invocation/packaging,
then validate existing behavior after extraction. Adapt Orrery under milestone 2.1;
update CI and the current combined-repository Orrery acceptance plan when split.
Then focus on Rust/C++/TypeScript (order not chosen). Java need not implement new
capabilities: return unsupported through Core and add Java features only on demand.

Keep the Mod-fed CSV adapter until native workspace dependency scanning and consumer
migration are usable. During CLI migration move the temporary input capability into
the workspace command shape with explicit external-input semantics; retire the old
`analyze workspace` entry then. Do not call CSV normalization native source scanning.
Later remove the temporary adapter, as tracked by TD-006. Exact transitional grammar
is open. Refactor-plan remains removed.

The next step is a small implementation-ready contract: configuration/request/result
schemas, remaining flag/default/exit semantics and an old-to-new capability mapping.
Then take runnable slices, preserving Java and temporary Orrery input compatibility;
do not start by moving every package or inventing an all-purpose plugin/rule framework.
