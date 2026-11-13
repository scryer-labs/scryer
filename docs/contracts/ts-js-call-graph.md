# TS/JS call graph

Request `call-graph` from the TS/JS engine. It uses the same public fields as
the [Rust call graph](rust-call-graph.md), including `symbols[]`, `calls[]`,
`callerSymbolId`, `targetSymbolId` and `resolution`. Svelte script ranges
refer back to the original file.

The resolver matches stable lexical bindings and direct named/default imports.
It supports local JSONC `tsconfig.json` path mappings, relative `extends`,
explicit re-exports and single wildcard forwarding. Dynamic calls, namespace
calls, mutable bindings and external symbols remain unresolved. Use
`resolutionReason` for observed but unresolved calls and `limitations[]`
for known gaps in scan coverage.

An optional `testEvidence` on a call records the source import's `module`,
`export`, and `kind`. It identifies an explicit registration call from
Vitest, Jest, Playwright or Node's test API, at the call site only.

JSX render relationships stay in `architecture.react`.
`architecture.svelte` exposes component files and template tags with source
locations. A unique default import pointing to a Svelte file produces an
`import-candidate`; template scope and dynamic selection remain unverified.

For SvelteKit, `files.imports[]` can report `resolution: "convention-candidate"`
and a `targetFile` when the default `$lib` layout points to one file and no
explicit alias resolves. This remains a candidate because app configuration
can override the default.
