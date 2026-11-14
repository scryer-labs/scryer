# Toolchain-assisted scan facts

Request `--resolve` alongside a fact supported by the engine's
`resolveFacts` capability. Static scans remain the default. Toolchain
evidence is collected at scan time and can vary with the checkout,
platform, installed tooling and available dependency state.

| Engine | Facts | JSON evidence |
| --- | --- | --- |
| Rust | `dependencies` | `data.resolution` has `basis: "cargo-metadata"`, host `target`, and per-manifest results. A resolved manifest includes Cargo metadata v1, including the dependency graph. Fallback retains a diagnostic alongside the declarations. |
| Go | `files`, `call-graph` | `data.semantic` reports the outcome of the Go package model. The model refines `inBuild` and source-level package and call bindings. |
| TS/JS | `files`, `call-graph`, `ipc` | `data.semantic` includes compiler version, program and diagnostic counts when available. `semanticTargetFile` and `semanticTargetSymbolId` carry confirmed local source targets alongside the static observations. The latter joins to the existing `symbols[]` table. Svelte template evidence uses original source locations. Tauri endpoints still use the common IPC shape. |

Toolchain-assisted facts can have `semantic.status` equal to `resolved`,
`partial`, or `fallback`. The collected outcome can still contain useful
static data in the last two cases. Consumers should display the available
diagnostics and preserve the provenance of any semantic edge. Observed
dynamic dispatch still requires independent runtime evidence.

The TS/JS resolver reads the configured compiler toolchain from
`SCRYER_TS_TOOLCHAIN` or Scryer's `ts-toolchain` config directory. The Svelte
adapter maps positions back before emitting facts. It can emit a call
without a caller ID when its template site has no source function caller.

Language tools provide evidence about the scanned project. Scryer's own
installed engine inventory and matching ABI symbol names supply no project
relationship evidence.
