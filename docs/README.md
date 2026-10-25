# Documentation

Updated 2026-10-25. Each document has one responsibility. Completed roadmap items
are marked **Done** with strikethrough; current reference contracts remain readable.
Plans and decisions are not proof that a feature exists.

| Document | Responsibility |
| --- | --- |
| [Project README](../README.md) | Purpose, quick start, checks and documentation entry point. |
| [Command reference](CLI.md) | Current CLI, report formats and evidence/analysis boundaries. |
| [Architecture](../ARCHITECTURE.md) | Current Go/JVM layers, package responsibilities and contributor design rules. |
| [Analyzer contracts](contracts/README.md) | Current Go interface, process transport and versioned reports. |
| [Milestones](MILESTONES.md) | Single priority/status index for implemented and planned capabilities. |
| [Analyzer distribution decision](decisions/001-analyzer-distribution.md) | Agreed repository/Go-wrapper/install direction, rationale and unresolved packaging details. |
| [Technical debt](TECH-DEBT.md) | Existing implementation gaps and deferred maintenance, not a duplicate feature roadmap. |
| [Verification history](../VERIFICATION.md) | Preserved early verification observations, not current test totals or current behavior. |

Reference docs say what works **now**. Decisions say **why and what was agreed**.
Milestones say **what comes next**. Do not scatter next-step lists through command
examples or rewrite historical verification as if checks were run today. Update
changed agreements explicitly rather than leaving conflicting directions behind.
