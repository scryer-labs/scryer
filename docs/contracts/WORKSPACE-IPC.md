# Project and workspace IPC contract

Scryer records Windows named-pipe and shared-memory evidence. Both project and
workspace reports publish communication relationships and their source locations.
The matching contract is shared by all engines. The engine ID records where
an observation came from.

## Scan and inspect

```sh
scryer project scan --facts architecture,ipc --json > project.json
scryer workspace scan --facts dependencies,ipc --json > workspace.json
```

The workspace root must contain `.scryer/workspace.yaml` with its member paths.
Use a current Scryer build and an installed C++ Engine that advertises `ipc`.
The C++ engine reads source and CMake declarations without building the Windows
target. In Orrery, use the published Scryer provider's workspace or project scan
operation with the same facts. `ipc` alone also
collects executable ownership evidence. `architecture` supplies the component
table for joining and display.

## Read the report

- In project JSON, `communication.ipc` contains matches between distinct
  executable components. Link `from` and `to` refer to component IDs.
  Endpoints with no recovered executable remain visible without an inferred
  component link. Known test executables can participate in project matches.
- In workspace JSON, `communication.ipc` contains matches between distinct
  member projects. Link `from` and `to` refer to member paths. The workspace
  imports normalized endpoints and can enrich named-pipe calls with evidence
  from referenced projects. Imported endpoints have a `projectEndpointId`
  that refers to the ID in that member's `facts.communication.ipc`. Project
  links remain under the member facts.
- `projects[].path` identifies a workspace member. A member contains `facts`
  or `error`. A project's raw IPC facts are in its collected
  `engineFacts[]` outcome for `fact: "ipc"`. The outcome's
  `data.windows.namedPipe` contains endpoint observations and parameter
  evidence. `data.calls` can contain qualified calls with known arguments.
- `dependencies.references[]` holds library references. In the example
  workspace, Brush and Inkstone both reference Felt.
- `communication.ipc.windows.namedPipe` holds normalized `endpoints[]`
  and `links[]`. Each endpoint has an `id`, `role`, original pipe
  `name`, `engine` and `sourceRole`. It can also carry a `componentId`
  and, at workspace scope, a member `project`. Direct observations carry
  `observation`. Derived observations carry `call` and `parameterEvidence`.
  A wrapper resolved across repositories also has `libraryProject`. These objects retain source
  positions relative to the caller or library project that owns them.
  Endpoints and parameter evidence may include `accessMode` (`read`,
  `write`, `read-write` or `unknown`) and the source expression in
  `accessFlags`. Access is relative to the endpoint. The C++ engine
  derives it from the access argument to `CreateNamedPipe` or `CreateFile`
  when it can read the flags directly.
- A link uses `clientEndpointId` and `serverEndpointIds[]` to reference
  endpoint records. `from` identifies the client and `to` identifies
  the server at the scope of the report. A single server endpoint can appear in several
  links. `direction` describes data flow allowed by the observed access
  modes. It does not assert that a message was sent.

  | `direction` | Meaning |
  | --- | --- |
  | `client-to-server` | Client writes and server reads |
  | `server-to-client` | Server writes and client reads |
  | `duplex` | Both endpoints can read and write |
  | `incompatible` | The server does not allow the access requested by the client |
  | `unknown` | No unique server, missing access evidence or differing conclusions across server observations |

  Consumers should treat a missing `direction` in older artifacts as
  `unknown`. Pipe-name matching and its `status` are independent of
  access compatibility.
- `matched` identifies a single peer at the scope of the report and may
  retain several endpoint observations. `ambiguous` lists candidate peers.
  `unmatched` has no peer. Endpoints remain visible even without a match.
  Workspace matching excludes observations marked `sourceRole: "test"`.
- Endpoint IDs are local to one report. Use them for joins in that report.
  The match uses compatible roles and the case-normalized local Windows
  pipe name. Workspace edges represent static evidence of intended
  communication. The current matcher covers local pipe names.

## Windows shared memory

The C++ engine emits this fact at
`engineFacts[].data.windows.sharedMemory` for `fact: "ipc"`. Request
`--facts architecture,ipc` to include CMake component IDs for declared
entrypoint sources. `mappings[].operations[]` records named create/open
operations and `views[]` records mapping calls. Entries preserve API and
source locations, access flags, and available `entrypoints[].callChain`.
Use `entrypoints[].componentIds` to join to project architecture.
`handleSourceCandidateId`, when present, points to an observation within
the same project report. Access modes describe permissions; consumers
should avoid inferring message direction or actual writes from them.

Both scopes publish normalized `communication.ipc.windows.sharedMemory`
endpoints and links when shared-memory evidence is present. Links match
create/open observations by the exact mapping name. `openerEndpointId`
and `creatorEndpointIds[]` reference the participating endpoints.
`from` identifies the opener and `to` identifies the creator. Shared-memory
links describe a named resource association without a data-flow direction.

## Consumer rendering

Show named-pipe access direction independently of endpoint role. A duplex
match can be drawn with an arrowhead at each end. Unknown or incompatible
access can be displayed without a data-flow arrowhead. Preserve source
locations, wrapper parameter evidence, unmatched endpoints and ambiguous
candidate sets. Library dependency edges remain independent of IPC.

Acceptance covers Scriptorium's Brush and Inkstone duplex link, Felt wrapper
parameter evidence, and project-local shared-memory associations.

See the [CLI reference](../CLI.md), [Workspace report type](../../internal/application/workspace_scan.go)
and [IPC types and matcher](../../internal/application/workspace_ipc.go).

## POSIX resources (additive source-development contract)

Both scopes may additionally publish `communication.ipc.posix.unixSocket` and
`communication.ipc.posix.sharedMemory`. Each has `endpoints[]`, `links[]` and
`observations[]`. The engine identity records provenance only: this shape does
not depend on C++ and can be supplied by another engine. C ABI v1 and report
schemaVersion 1 are unchanged. A missing POSIX branch means unavailable evidence.

- Endpoints reuse `IPCEndpoint`: ID, original resource name, role, engine,
  applicable executable component ID, project/projectEndpointId at workspace
  scope, and the original observation. Unix sockets also carry `transport`
  (`datagram` or `stream` when known).
- Observations retain `{engine, data}` plus `project` at workspace scope. They
  include unresolved native calls and must remain available even if there are no
  endpoints or links. Do not interpret an empty endpoint array as proof of no IPC.
- Links reuse `from`, `to`, `name`, `status` and optional candidate participants.
  `sourceEndpointId` and `peerEndpointIds[]` join observations;
  `basis: "resource-name-candidate"` explicitly limits the evidence. `matched`
  means one compatible participant, not established traffic or deployment.
- Initial pairing covers Unix datagram `send` to `bind` with the exact absolute
  socket path (or abstract-namespace name), and POSIX shared-memory `open` to
  `create` with the exact absolute resource name. Names are case-sensitive.
  Datagram links have `direction: "sender-to-bound-receiver"`; memory links have
  `direction: "unknown"`. Relative socket paths, unknown/different transports,
  unresolved names and missing project-level executable ownership do not produce
  confirmed peer matches. Stream connect/listen/accept is not paired yet.
- Project links connect distinct recovered executable components. Workspace
  imports project endpoint identities and matches across member projects, keeping
  internal project links under member facts. Workspace-level project association
  remains possible even when executable ownership is unknown; it never assigns
  a library as an executable. Known test observations are excluded from workspace
  pairing by the existing sourceRole rule. Unknown roles remain unknown.

The matcher assumes compatible local resource namespaces; runtime deployment,
containers, mount namespaces and actual resource existence are not proven.
Source value propagation across arbitrary wrappers and global arrays is not
complete. Consumers must show unresolved observations and candidate relationships
separately from runtime measurements. Markdown has basic POSIX link tables and
retains complete JSON; interactive rendering belongs to consumers.
