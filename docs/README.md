# Documentation

Updated 2026-11-01. Start with the current reference for existing behavior and the
milestones for next work. Planned features are not proof of implementation.

## Current reference

| Document | Responsibility |
| --- | --- |
| [Project README](../README.md) | Purpose, quick start and documentation entry point. |
| [Command reference](CLI.md) | Current commands, report formats and evidence boundaries. |
| [Architecture](ARCHITECTURE.md) | Current Go/JVM layers and package responsibilities. |
| [Analyzer contracts](contracts/README.md) | Current Go interface, process transport and versioned reports. |
| [Contributor instructions](../AGENTS.md) | Coding/commit conventions and Orrery acceptance-plan rules. Kept at the repository root as the collaboration entry point. |

## Direction and maintenance

| Document | Responsibility |
| --- | --- |
| [Product direction](PRODUCT-DIRECTION.md) | Product boundaries, engine/consumer responsibilities and supported-platform direction. |
| [Milestones](MILESTONES.md) | Single delivery-order/status index; completed items use **Done** and strikethrough. |
| [Technical debt](TECH-DEBT.md) | Existing limitations and temporary capabilities to remove, separate from new features. |

## Decisions and rationale

Read current direction above for the consolidated position. These records preserve
why agreements were made and how later decisions refine them; do not merge their
historical positions into a second current specification.

| Decision | Responsibility |
| --- | --- |
| [001 — Analyzer distribution](decisions/001-analyzer-distribution.md) | Independent Java repository, Go wrapper, installation and packaging boundaries. |
| [002 — Scan scopes and fact ownership](decisions/002-scan-scopes-and-fact-boundary.md) | Project/workspace configuration, source-first collection and JSON outcomes; rule ownership refined by 003. |
| [003 — Stack discovery and shared rules](decisions/003-stack-discovery-and-shared-rules.md) | Engine-independent stack detection, auto-install composition and evidence-backed checks. |

## Historical evidence

[Early Java verification](archive/VERIFICATION.md) preserves past increment checks.
It is not a current acceptance guide, test total or roadmap. New acceptance evidence
belongs to the relevant Orrery run, not an ever-growing root-level verification log.
