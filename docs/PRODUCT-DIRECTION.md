# Product direction and engine handoff

Updated 2026-10-31. Planned boundaries, not implemented commands or configuration.
[Milestones](MILESTONES.md) owns delivery order; [CLI](CLI.md) and
[architecture](../ARCHITECTURE.md) describe current behavior.

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
extraction belong to engines. Architectural fitness, rule verdicts, test adequacy
and refactoring recommendations belong to consumers. Existing Java impact/evidence
matching and workspace phase/plan suggestions still exist; the CLI review must
classify them before deliberately migrating behavior. This direction does not
silently remove current commands or evidence.

Shared CLI/fact contracts and presentation stay in Core. Engine interfaces do not
import Orrery types or have to match Orrery provider interfaces. Terminal, JSON and
other reports derive from the same facts; presentation must not invent evidence.
Every command, including engine management, must provide a JSON result contract.
The exact default stdout/file delivery policy remains to be designed.

## Explicit scan scopes

A project is a repository and may contain multiple languages and build modules.
A workspace explicitly groups projects and their cross-repository relationships.
A build-system workspace/module is a project-internal structure, not automatically
Scryer's workspace scope. Workspace must become a first-class scope rather than
remain a special case under `analyze`.

Workspace and project `.scryer/` configuration will initially be handwritten.
Workspace configuration retains membership, so repeated scans do not require
re-entering project paths. Project configuration holds project-specific scan
options; project scans can also use defaults without a configuration file.
See [decision 002](decisions/002-scan-scopes-and-fact-boundary.md) for ownership,
output and unsupported/partial-result principles. Filenames, schema and command
grammar are not yet implemented or finalized.

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
Java feature expansion is not the next focus.

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
library/package relationships, then IPC (shared memory, named pipes and supported
macOS mechanisms), then REST provider/consumer relationships. Preserve package,
version, configuration, source and engine/schema provenance. A matching name alone
is not proof that a local checkout supplies a dependency or that two endpoints
communicate. Candidate matches and unresolved relations remain explicit.

Orrery owns durable history, code-state associations, interactive architecture
views and user annotations. Consumers may compare supplied facts and evaluate
architectural expectations; missing evidence must not become a passing judgment.
Do not wait for every language or IPC mechanism before exposing useful project facts.

## Development acceptance

Edward will establish the scryer-lab workspace and its Orrery acceptance plan,
aligned with GitHub Actions. Once available, invoke that plan and inspect its
recorded results; Edward reviews the same run in Orrery. Do not replace it with
separately executed local test/build commands or change its scope without agreement.
The plan is not claimed to exist yet. See [contributor instructions](../AGENTS.md).
