# Engine C ABI v1

Core selects an engine from installed packages. A package can declare a
package-relative `library` entry in `engine.json`. Core checks the platform and
resolves all three v1 symbols before use. Listing reports the installed
inventory. Load failures and execution errors describe operational availability.

The canonical declarations are in
[the v1 header](../../internal/engineabi/scryer_engine_v1.h). Both requests and
request IDs use `scryer_view_v1`, a borrowed read-only pointer and byte length.
The caller keeps the memory valid for the call. The engine copies any bytes
it needs after the call. Text is UTF-8 and sizes exclude a trailing NUL.

- `execute_v1` runs synchronously. Core generates the `requestId` inside the
  existing JSON request before calling it. Status zero carries the existing
  JSON response. Other statuses identify an execution error, cancellation,
  or busy engine and may carry a JSON error message. Java accepts one active
  execution per loaded library.
- `cancel_v1` accepts the same ID from another thread. True means cancellation
  was accepted. False means the ID is currently inactive. If execution is
  still outstanding, Core retries every 10 ms until acceptance or return.
  The caller waits for execution to return after requesting cancellation.
- `release_buffer_v1` returns void. Core copies the response bytes, then hands
  the buffer back exactly once to the same library, including buffers from
  error results. Core caps responses at 256 MiB.

Java runs its JVM in a separate process group and signals the group on
cancellation. Its wrapper resolves jars and the bundled runtime relative to the
library. Core keeps loaded libraries resident for its process lifetime. The
dynamic loader supports macOS and Linux with cgo. Packages can also provide an
executable for standalone use. Core selects the library when both are declared.

The ABI version governs the function layout and lifetime rules. The JSON
`protocolVersion` continues to govern request and response content.
