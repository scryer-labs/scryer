# Scryer contributor instructions

Follow the package responsibilities and readability guidelines in `ARCHITECTURE.md`. Prefer clear branches, named intermediate results and named arguments; retain lambdas and collection pipelines when they help readability.

Commit subjects use `<type>: <summary>`. Allowed types are:

- `feat`: new functionality
- `refactor`: structural/readability changes without intended behavior changes
- `tests`: test additions or changes (use `tests`, not `test`)
- `docs`: documentation changes
- `chore`: build, CI and repository maintenance
- `fix`: corrections to existing behavior
- `revert`: reverting an earlier change

Choose the type for the primary purpose of the commit. Keep summaries concise and specific. Do not rewrite existing history unless the user requests it.

## Development acceptance

Agreed 2026-10-31: Edward will create the scryer-lab workspace and an Orrery plan
aligned with GitHub Actions. Once configured, trigger that plan for acceptance and
inspect its recorded results so Edward can review the same run in Orrery. Do not
substitute separately executed local test/build commands or modify the plan's
commands/scope without agreement. Do not assume a plan ID or that setup is complete;
resolve the actual plan when available. Read-only inspection and documentation
checks remain appropriate. Report exactly which checks ran and any missing
acceptance evidence; documentation checks are not code-pipeline acceptance.
