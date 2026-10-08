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
