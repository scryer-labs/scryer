# Rust call graph

Request `call-graph` from the Rust engine. The collected outcome has
`projectId`, `basis: "tree-sitter-syntax"`, `symbols[]` and `calls[]`.
Symbols use `symbolId`, `name`, `kind`, and a `definition` range. Calls
use `callSiteId`, `callerSymbolId` when known, `callee`, `site`, and
`resolution`. Ranges use repository-relative paths and one-based lines
and byte columns, with exclusive ends. IDs join observations within
one scan. Symbols and calls can carry the shared `testContext` object.

A direct call can have `resolution: "resolved"` and `targetSymbolId`.
The Rust resolver handles unambiguous module-level functions, paths qualified
with `crate::`, `self::`, or `super::`, supported explicit import aliases
and re-exports, visible functions in the binary's own Cargo package library
and in explicit unconditional local Cargo dependencies, and calls through
`self` or `Self` within the same inherent implementation. It follows resolved module declarations and checks visibility.
For example, Ink's `main.rs` calls `scriptorium_ink_lib::run` in `lib.rs`,
and Orrery's binary calls `orrery_backend::api::router`.

Use the symbol definitions to join the call to file and Cargo component
facts. This is bounded source resolution. Conditional declarations, glob
imports, registry or Git crates, indirect calls, type-directed method
dispatch and macro expansion remain outside the resolver. Unresolved calls
retain their site, expression and `resolutionReason`. The graph also lists
its known coverage gaps in `limitations[]`.
