# Tauri IPC source facts

The Rust and TS/JS engines expose their source observations independently in
the collected `ipc` outcome at `engineFacts[].data.tauri.apps[]`.
Project-scope matching joins observations belonging to the same app.

Each app records a repository-relative `config` path and `endpoints[]`.
An endpoint records `kind`, `role`, `name`, `file`, `line`, and `column`.
Locations use one-based line and byte-column, relative to the project root.
Command endpoints also record the observed Rust `symbol`.

| Kind | Role | Rust evidence |
| --- | --- | --- |
| `command` | `declaration` | Function with a plain `#[tauri::command]` attribute |
| `command` | `registration` | Path in `tauri::generate_handler![…]` passed to `invoke_handler` |
| `event` | `emit` | Literal event name in an `emit` call on a parameter with a recognized Tauri type |

The frontend collector emits `command/invoke`, `event/listen`, and
`event/emit` for literal names and recognized imports from
`@tauri-apps/api/core` or `@tauri-apps/api/event`. It retains source locations
in the original Svelte file. The app config is found in the source's ancestor
directories or their conventional `src-tauri` child. Ambiguous configs and
dynamic names yield no frontend endpoint.

A declaration and a registration are separate observations. The app
identity is taken from the nearest supported Tauri config enclosing the
source. The scanner preserves source evidence for supported syntax. Renamed
command attributes, dynamic event names, macro expansion and semantic type
resolution are outside the initial slice.

## Project aggregation boundary

Core composes Rust and frontend endpoint facts in a dedicated
`internal/framework/tauri` package. The project scan calls the package
with collected observations for the project. The package owns matching by
app, mechanism and name, and emits links with the source evidence from both
sides. The Rust and frontend engines each own their source observations.
Keep the matching rules behind this package boundary so a separate Tauri
engine remains an option when the framework needs its own scan or lifecycle.

The project result is `communication.ipc.tauri.apps[]`. An app contains
`config`, `endpoints[]` and `links[]`. Normalized endpoints include the producing
`engine`. Each link has `kind`, `name`, `from` and `to` source endpoints.
Command links go from `invoke` to the matching `declaration` and include
`registration` evidence. The Rust scanner may add a `declaration` location
to a registration when its explicit path resolves in a unique Cargo module
tree. The location has `file`, `line`, and `column`. Rust endpoints in an
inline module may also include a scan-local `moduleId`. Core requires one
registration for the name. When location evidence is absent, Core can use a
unique exact symbol in the same file and module if the registration has no
`resolutionReason`.

Event links go from `emit` to each matching `listen` on the other engine.
Unmatched invocations and emissions appear in optional `unmatched[]` entries
with `endpoint` and `reason`. Current reasons are
`command-target-not-unique-or-unproven` and
`no-cross-engine-listener-evidence`. Declarations and listeners remain in
`endpoints[]` even without links. An empty match set describes the collected
evidence for the scan. The engine IPC payload's `limitations[]` describes
known gaps in endpoint collection. Workspace reports retain Tauri links inside
the corresponding member's project facts.

Ink exercises two communication mechanisms. Svelte invokes the Rust
`select_candidate` command and listens for the Rust `render_update` event
within one project. Ink's Rust backend also opens the duplex
`scriptorium_ui_protocol_pipe` to communicate with Inkstone across projects.
The Tauri command and event are separate source observations. They can be
shown alongside the workspace pipe link without implying that a particular
event is the correlated reply to a particular invocation.

The Rust named-pipe collector supplies Ink's Tokio endpoint separately in
`engineFacts[].data.windows.namedPipe.endpoints[]`. Core matches that endpoint
to the C++ server using the existing Windows pipe contract. The workspace
link has `direction: "duplex"` when both endpoints allow read and write.
