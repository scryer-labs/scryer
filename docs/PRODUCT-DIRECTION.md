# Product direction and engine handoff

Updated 2026-11-02. Planned boundaries, not implemented commands or configuration.
[Milestones](MILESTONES.md) owns delivery order; [CLI](CLI.md) and
[architecture](ARCHITECTURE.md) describe current behavior.

## Independent facts tool

Scryer is an independent fact-scanning tool with installable language engines.
It produces structured facts and human-readable reports for people, AI and other
consumers, including Orrery and a future Hunch-test comparator. Future plugin
categories are possible, but do not justify a general plugin framework now.

Scryer does not own scan history, tracking, persistent timelines, plan execution
or design judgments. It may scan explicitly supplied Git history or branch-policy
inputs in the future without retaining its own history. Installed-engine settings,
portable `.scryer/` configuration, temporary snapshots, output files and caches are
compatible with this boundary. Existing retained evidence is not deleted or
reinterpreted by this decision.

Language/build-specific parsing, symbol resolution and evidence-backed relationship
extraction belong to engines. Core may later provide shared, reproducible rules
for code smells, bounded test-effectiveness checks and evidence-backed refactoring
suggestions. Business/design trade-offs and consultant-style advice remain human
or consumer responsibilities. Existing Java impact/evidence
matching remains; the CLI review must classify it before deliberately migrating
behavior. Workspace phase/plan generation has been removed; Orrery owns that use
case. The Mod-fed workspace CSV adapter remains temporarily under TD-006. Other
current commands and Java evidence remain unchanged by this cleanup.

Shared CLI/fact contracts and presentation stay in Core. Engine interfaces do not
import Orrery types or have to match Orrery provider interfaces. Terminal, JSON and
other reports derive from the same facts; presentation must not invent evidence.
Engines return only structured JSON; Core aggregates engine/project/workspace results
through distinct contracts. Core generates MD/HTML from JSON using shared components,
stack-specific report modules and separate scope views. Unknown extensions retain
JSON-detail fallbacks; there is no report plugin framework in this slice.
Every command, including engine management, must provide a JSON result contract.
The exact default stdout/file delivery policy remains to be designed.

## Explicit scan scopes

A project is a repository and may contain multiple languages and build modules.
A workspace explicitly groups projects and their cross-repository relationships.
A build-system workspace/module is a project-internal structure, not automatically
Scryer's workspace scope. Workspace must become a first-class scope rather than
remain a special case under `analyze`.

Workspace configuration retains explicit members, managed through init/add/remove.
Workspace initially uses only `.scryer/workspace.yaml`; project configuration uses
`.scryer/engines/<engine-id>.yaml`, with no generic project.yaml. Engines supply
minimal default templates (initially allowed to be empty), and Core initializes
missing files without overwriting existing choices. Derive fields from real needs;
pre-stability engine configs do not require compatibility/migration mechanisms.
Workspace never owns engine-specific configuration. Scope-first commands, default
stack/LOC scanning, analyze, report completeness and outstanding schema decisions
are consolidated in [decision 004](decisions/004-multilanguage-scopes-cli-and-engine-config.md).
These are agreed design, not implemented CLI/configuration capabilities.

## Stack discovery and shared checks

Core should identify project/workspace technology-stack signals without installed
language engines, using lightweight file/manifest/build-declaration detection with
source evidence. Workspace discovery uses configured membership. Users explicitly
batch-install missing engines. Installed inventory and a remote available-engine
list remain separate; no install-all or stack-driven auto-install is planned in
this slice. This refines decision 003's earlier auto-install direction. Ordinary
scans do not install engines, but first engine use may initialize its missing
project configuration as agreed in decision 004.

Shared checks operate on common facts rather than being reimplemented in every
engine repository. Declarative mappings can describe language-to-fact normalization
and rule conditions; JSON configuration does not replace semantic extraction or
shared algorithms. Engine capabilities progressively cover symbols, typed relations,
types, source locations, control/data flow and test/execution evidence. Rules declare
the facts they require; missing capabilities/evidence remain unsupported or unknown.
Findings and raw facts remain distinct. See
[decision 003](decisions/003-stack-discovery-and-shared-rules.md), which refines the
previous blanket exclusion of rule judgments. `suggest`/`check` names and schema
remain open; derive the shared rule format from concrete cross-language cases later.

## Source-first facts and supported platform

Basic source/symbol extraction must not require the target project to build or run.
Richer resolution may need dependencies, generated sources, compiler options or
build metadata. Execution evidence requires explicit execution capabilities/options.
Retain available facts and identify unresolved or unsupported portions rather than
requiring a successful target build before producing anything.

macOS is the current primary development and acceptance platform. Windows migration
for daily input-method development is deferred until Scryer and Orrery are stable;
Ubuntu-specific IPC work is outside the current scope. Existing Ubuntu CI remains
in place. Keep platform boundaries clear without adding Windows work now.

## Engine handoff and consumer loop

First agree CLI/configuration/output contracts. Then establish the minimal external
engine boundary, verify local install/uninstall/list plus an actual protocol call,
and move Java into the user-created `scryer-java-engine` repository. The `engine`
category name is provisional. Keep existing Java capabilities and evidence during
the split, with Java runtime requirements owned by the optional Java engine.
Java validates the engine mechanism and preserves its current abilities; future
capabilities may be unsupported until there is a concrete Java need.

After extraction, milestone **2.1** updates Orrery's Scryer provider for the new CLI
and engine contract. Add a new published provider version; do not edit immutable
published provider YAML. Validate standalone use and the Orrery integration.

The next language is chosen at that point. C++, Rust, TypeScript, Go and Markdown
are target capabilities; their implementation order is undecided. Mixed-language
projects include Scriptorium Ink (Tauri), Orrery (TypeScript/HTML/CSS/Rust) and
Scryer (Go/Kotlin). Compose available engines and expose coverage gaps before
attempting complete cross-language semantic linkage. Tauri is a framework, not a
source language.

Facts should cover symbols, typed relations, architecture structure/pattern signals,
internal dependencies and workspace-local cross-repository dependencies. Prioritize
library/package relationships and concrete REST/named-pipe use cases. REST takes
priority over other network mechanisms; GraphQL/RPC wait. Windows shared-memory
validation remains deferred under the macOS-first agreement. Preserve package,
version, configuration, source and engine/schema provenance. A matching name alone
is not proof that a local checkout supplies a dependency or that two endpoints
communicate. Candidate matches and unresolved relations remain explicit.

Orrery owns durable history, code-state associations, interactive architecture
views and user annotations. Consumers may compare supplied facts and evaluate
architectural expectations; missing evidence must not become a passing judgment.
Do not wait for every language or IPC mechanism before exposing useful project facts.

## Development acceptance

[Contributor instructions](../AGENTS.md#development-acceptance) own the Orrery plan
location, invocation rules and acceptance reporting. The current plan covers the
combined CLI/Java-engine layout; update it alongside CI when repositories split.
