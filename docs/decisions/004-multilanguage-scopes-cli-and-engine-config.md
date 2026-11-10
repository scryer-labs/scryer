# Work decision 004: Multilanguage scopes, commands and engine configuration

Date: 2026-11-02
Updated: 2026-11-02, including configuration and report-layer refinements.
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

For a mixed-language project, Core matches each requested fact to every relevant
installed engine that declares support for it. Each engine receives only the
intersection of its capabilities and the requested facts. Core merges successful
results with language and engine provenance and reports unsupported facts for
each relevant engine that lacks them. A fact unsupported by all relevant engines
is reported as unsupported for the current stack. Other requested facts still
return their results, including Core facts such as stack and LOC. Keep a fact
that was never collected distinct from a collected fact with no findings.

The installed package manifest declares its operations and scan fact capabilities.
Core dispatches scan and analyze through the existing C ABI execute entrypoint.
Engine availability, unsupported capabilities and gaps in an executed scan remain
separate outcomes. A Core upgrade uses the capabilities of the installed package.

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

Use YAML for the initial configuration layout. Workspace has only
`.scryer/workspace.yaml`, owning explicit project members and shared scan defaults.
Project has `.scryer/engines/<engine-id>.yaml` files, owning only each engine's
configuration. Do not introduce `project.yaml` or project-level generic defaults
in the initial slice. A directory that is both scopes can contain both layouts.
Exact workspace fields/path rules and template-retrieval protocol remain to be
specified; the division of files and ownership is agreed.

Explicit CLI `--facts` wins. Workspace scan otherwise uses its workspace defaults,
then built-in `stack,loc`. Standalone project scan otherwise uses `stack,loc`, without
searching parent directories for workspace configuration. Workspace forwards its
selected facts to member project scans. There is no project.yaml override layer.
Engine YAML controls extraction details, not a second implicit facts-selection layer.

Engine configuration is deliberately minimal and driven by actual scan/analyze
needs. An engine may initially ship an empty YAML template and accept an empty
configuration. It introduces fields only when a concrete need appears. Core obtains
and initializes that template through a uniform boundary; it does not interpret
engine-specific fields or centrally prescribe build/test option names.

Before engines/configuration stabilize, there is **no old-config compatibility or
migration requirement**. Fields/schema may change directly, with manual editing or
explicit regeneration as necessary. Do not add version fallbacks, migration layers
or compatibility scaffolding now. Revisit compatibility when engine/configuration
maintenance actually requires preserving established user data. This supersedes
the earlier proposal to require missing-field defaults for every new parameter.

1. Workspace scan discovers stacks; the user explicitly installs missing engines.
2. When a requested project scan/analyze needs an engine, use existing engine config.
3. If missing, the engine probes the project and supplies an applicable default
   template; Core writes that section/file without overwriting another writer.
4. Reuse user-edited config on subsequent operations. Do not reselect or overwrite
   it silently on scan or engine upgrade. Invalid config can be reported for manual
   correction; it does not require implementing a migration mechanism.

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

### Structured data flow

The physical scope and execution layers correspond:

```text
workspace operation
  -> project operation(s)
       -> engine(s), each using its own project YAML
       <- engine JSON facts
  <- project JSON results
<- workspace JSON result
```

JSON is the inter-layer fact contract and sole data basis for MD/HTML. Internal code
may use typed structures; this does not require serializing/parsing JSON at every
in-process call. Reports must not become a competing fact source. Workspace consumes
project structured results, never parses their Markdown/HTML to recover facts.

There are three distinct result contracts, sharing useful entity/relation/evidence
primitives rather than one mandatory identical document shape:

| Contract | Producer and consumer |
| --- | --- |
| Engine result | Engine returns structured facts, capabilities/gaps, provenance and engine-specific extensions to project orchestration. |
| Project result | Core combines engine results, project identity and supported intra-project relations; consumed by workspace and project reports. |
| Workspace result | Core aggregates project results and supported cross-project computations; consumed by workspace reports and external tools. |

Engines implement the common engine-facing JSON contract; they do not need a workspace
contract or knowledge of workspace reporting. Exact field names, identities, request/
result schemas and aggregation rules remain the next design task. No independent
engine-report CLI entry is required in this slice.

### Reports and modular rendering

JSON contains all facts actually collected in the requested scope, including entity
identities, relationships, provenance and missing/unsupported/partial limitations.
Markdown is information-equivalent rather than a truncated summary. HTML uses the
same data; collapse/filter/layout affects presentation only. Complete output does
not claim complete understanding of the source. New derived relations/metrics must
enter the structured result before rendering; renderers do not invent facts.

Engines return JSON only and do not implement Markdown or HTML. Core owns modular
report rendering with three boundaries:

- Shared components for entities, relationships, metrics, evidence and missing states.
- Stack-specific modules for supported Java/C++/Rust/etc. data presentation, without
  rerunning language extraction or fact analysis.
- Separate project/workspace views that compose those modules; no single identical
  page/template is required for both scopes.

Accept coordinated engine/Core-report changes for new stack-specific fields. Keep
module interfaces clear so components can be extracted later, but do not introduce
a report plugin registry, runtime module loading or engine-provided executable UI.
If an extension has no specialized renderer yet, preserve its information through
labeled JSON code blocks in Markdown and expandable JSON details in HTML. That is
a completeness fallback, not a claim of a polished semantic view.

Generate MD/HTML at the scope requested by the user. Workspace orchestration does
not need to generate every child project's reports before aggregation. Interactive
architecture HTML, possible Orrery embedding/reuse and layout selection remain a
separate topic. Clean-architecture layer mappings need configuration/evidence;
choosing a circular layout does not establish architecture. Output flags, default
stdout/file delivery and exit-code policy remain open.

## Engine process, installation and local state refinements

Agreed after the initial command design: engine wrappers remain Go executables,
invoked as independent processes through JSON, not statically linked libraries or
Go shared-library plugins. Start a process per operation; no persistent daemon,
listening port or service manager is required. Java's Go wrapper can own a JVM worker
child process. Core owns outer cancellation/result checks; the engine owns its
internal process/runtime details. Exact external protocol is defined during extraction,
not a prerequisite for the first command reorganization.

Deliver installation in small slices:

1. Expose engine list/install/uninstall with mocked boundaries/minimal JSON worker
   fixtures before moving Java. Remote-list is deferred.
2. Use the user-created Java-engine repository to verify real local installation,
   discovery, invocation, listing and uninstall. Uninstall removes owned install
   data, not source checkouts or project engine configuration.
3. Move existing Java scanning/analyze capabilities and their tests into that engine;
   run end-to-end Core/engine checks to preserve current behavior and evidence.
4. Publish an OS/architecture-specific stable GitHub Release package with the Go
   wrapper, JVM worker/dependencies, defaults and bundled Java 21 runtime. Install
   selects the compatible stable asset and validates it. This is a directory
   distribution, not a `.dylib`/static library. Start with macOS support.

Release installation must not require a local Go compiler or separately installed
Java. The engine runtime is distinct from toolchains needed to build/test the target
project. Reuse the current bundled-runtime packaging experience; do not assume
packing the engine makes arbitrary target builds self-contained.

Support source installation for engines under development, beginning with a local
source directory. The engine repository supplies an agreed build/package entry that
produces the same installation layout as release packages. Core does not hardcode
Gradle/Cargo/etc. Source installation requires the developer's build toolchain and
records source identity; installed contents must not depend on the checkout staying
in place. Git URL/ref syntax and build-entry/manifest schema remain open.

Global Scryer state holds local config, engine installations/selected versions and
installation metadata; downloads/source-build intermediates are separate cache data.
Choose platform defaults automatically and create directories only when needed.
Users need not know the home path, initialize global config or set environment
variables. Missing global config uses built-in defaults. `SCRYER_HOME` is an optional
advanced/test override, not a normal installation step. Exact default paths and file
schemas remain open. Engine templates stay in their packages, project options stay
in project YAML, and workspace membership stays in workspace YAML.

After extraction, Core must install independently with `go install` at its published
module path (with Go's binary directory on PATH), without building Kotlin or needing
Java. Default stack/LOC runs without engines. This is a planned acceptance condition,
not what the current combined build already provides.

## Scryer acceptance layers

As repositories separate, evolve the current acceptance plan and CI into Core
checks, per-engine checks, and a composed Core-plus-engine integration plan. Core
checks include standalone installation and engine-free baseline scanning when those
features land. Engine checks cover JSON/default-template/packaging behavior; the
integration plan covers installation/discovery/invocation/uninstall and project config.

An overall acceptance entry may reference these plans without copying their commands.
Identify the exact built artifacts and isolate test homes/project configuration from
normal installations. Order dependent build/install/consume steps and avoid concurrent
cleaning of one checkout.

The immediate acceptance ground is now the existing `test_workspace` (workspace
`modernization`): its workspace integration plan references seven project scans in
parallel, calling Scryer v1.4 directly and retaining JSON. No integration shell script
or Mod collection is required. Provider operations scan their resolved project target;
no directory override is needed. This supersedes the earlier fixture-deferral and
script-driven integration approach, without expanding native workspace scanning.

For routine changes run Scryer's plan followed by this integration plan. Add Orrery's
plan when Orrery/provider implementation changes. Exact local plan roots/IDs and API
invocation rules live in [contributor instructions](../../AGENTS.md#development-acceptance).

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
the workspace command shape with explicit external-input semantics. The first slice
uses `workspace import`; the follow-up removes all legacy entries and immediately
adapts the existing provider, rather than retaining compatibility until extraction. Do not call CSV normalization native source scanning.
Later remove the temporary adapter, as tracked by TD-006. The transitional CSV entry is `workspace import --projects ... --dependencies ...`. Refactor-plan remains removed.

The next implementation slice reorganizes the existing CLI into project/workspace
commands with optional project paths, while preserving current Java scan/analyze
behavior and the temporary CSV capability. New default stack/LOC and engine-free
scanning wait until Java extraction and the existing consumer-adaptation checkpoint.
Engine installation then proceeds through the slices above. Define the external
engine contract during extraction and defer native project-to-workspace aggregation
contracts until needed. Do not block command reorganization on all future schemas,
move every package at once or invent a general plugin/rule framework.


## Immediate consumer adaptation update

Agreed 2026-11-02: remove old CLI entries after committing command reorganization.
Publish an Orrery Scryer provider version using only project scan/analyze and
workspace import, and validate both repositories plus their integration now. Do not
retain hidden aliases until Java extraction. This moves command-level consumer
adaptation ahead of the original 2.1 checkpoint; 2.1 still covers later installed-engine
contract changes. Native workspace CSV-adapter retirement remains a separate task.

## 2026-11-02 extraction refinement

Preserve Core → Go wrapper → JVM for the repository extraction. Core and engine
CI/plans separate with the code; engine produces JSON only, including in its tests.
After acceptance, the next pair session experiments with a C ABI dynamically loaded
wrapper that starts the JVM inside Core's process context. This could remove a
resident wrapper process while preserving independently distributed engine code.
It is not a decision to adopt Go plugin or embed JVM through JNI. The independent
process baseline remains available while ABI/memory/error/lifecycle tradeoffs are
explored. Native workspace scanning and default stack/LOC remain deferred.

## 2026-11-02 configuration/request and future scan clarification

Engine-owned templates initialize persistent project options under
.scryer/engines/<id>.yaml through Core. Build-tool selection and default test
commands are examples of engine-owned settings; their final YAML shape remains
engine-driven. This is not a stored copy of an execution request. Core generates
protocolVersion/requestId/operation, resolves the selected project, refs and immutable
SHAs, creates beforeRoot/afterRoot snapshots, and assembles the request using CLI
selections and project configuration. Engines define their option semantics, not
caller-owned snapshot lifecycle. Neither request envelopes nor ephemeral paths
belong in project YAML. Template initialization remains planned, not implemented
by the local-package installer.

The response envelope is language-neutral; Java-specific data resides inside the
report. The abbreviated {schemaVersion: 1} example is not a complete valid report.
Do not inherit the current Java report's required fields or CLI build-tool flags
as requirements for future engines merely because the transport envelope works.

The existing Java scan focuses on build/dependency/version observations, with
additional local metadata; it is a preserved legacy capability and an integration
validation engine, not the reference design for future scan. C++, Rust, Go and
TypeScript must start from the agreed source-fact goals: architecture/entities,
symbols, ownership and declaration/implementation/call relations as supported,
plus dependencies and explicit evidence/limits. Source facts must not intrinsically
require a runnable project; builds can enhance resolution. Core's stack/LOC baseline
is engine-independent. Define common facts and language-specific extensions from
actual language cases; do not force them into Maven/Gradle/JaCoCo or Java-led report
shapes. Java can remain unsupported for new capabilities until there is demand.

## 2026-11-02 global directory convention

The default global Scryer home is ~/.config/scryer, including on macOS, matching
Edward's nvim/Orrery convention. Do not use os.UserConfigDir()'s macOS
Library/Application Support default. SCRYER_HOME remains an optional explicit
override. The development installation created at the former location was moved
once; no pre-stability legacy-path fallback/migration feature is required.
Global installed-package state remains separate from planned project engine YAML.
