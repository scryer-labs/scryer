# Product direction

[Milestones](MILESTONES.md) holds the delivery order, and the
[command reference](CLI.md) describes the current interface.

## Facts and consumers

Scryer is a lightweight repository fact provider for people and tools. Core
owns scope, engine selection, shared contracts, aggregation and presentation.
Engines own language and framework extraction and return JSON. JSON carries
the collected facts, including source locations, provenance, unresolved
observations and unsupported capabilities. Markdown is a human-readable
projection of those facts.

Orrery owns durable scan and plan history, interactive architecture, user
annotations and design decisions. Scryer may read Git history and explicitly
run target tooling to collect evidence. It keeps no history of its own scans.
Static scanning works without a successful target build. Toolchain resolution
and test execution have separate, explicit evidence boundaries.

## Scope and composition

A project is a single repository, possibly with multiple languages and build
units. A workspace is the highest scope and explicitly groups projects.
Core dispatches workspace scans to projects and project scans to matching
engines. Workspace aggregation uses project facts to match relationships
such as local library references and IPC. A build-system workspace or project
module remains internal to its repository.

Architecture, files and call graphs provide linked detail. Components belong
to a project. Files record component membership where observed. Symbols and
call sites have source locations. Identity and source evidence let consumers
follow the relationships without merging unrelated candidates.

Workspace membership uses `.scryer/workspace.yaml`. The agreed future
project-local configuration is `.scryer/engines/<engine-id>.yaml`. Engines
would supply minimal defaults and Core would initialize missing files.
Ordinary scanning and engine-free stack/LOC discovery leave installation
to an explicit engine command.

## Evidence-backed checks

Future shared checks may evaluate coverage, change impact, test effectiveness
and code smells using defined inputs. Engines supply language-specific
evidence, with common rules shared where the facts support them. Missing
evidence remains unknown. Architectural trade-offs and consultant-style
recommendations stay with users and consumers.

macOS is the primary development and local acceptance platform. Windows
packages are part of the distribution path. Build and execution support
depends on the target toolchain and project. See
[architecture](ARCHITECTURE.md) and [milestones](MILESTONES.md).
