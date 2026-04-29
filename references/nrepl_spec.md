# nREPL Protocol Specification

This document specifies the nREPL (network REPL) protocol as implemented by the
reference Clojure nREPL server (v1.6+). It is intended as a reference for
building Go client libraries and CLI tools.

---

## 1. Architecture Overview

nREPL is a **message-oriented, asynchronous** client–server protocol. The three
core abstractions are:

- **Transports** — encode/decode messages over a network channel (analogous to
  HTTP adapters in Ring). The default and primary transport is **bencode over
  TCP**. EDN and TTY transports also exist.
- **Handlers** — server-side functions that accept a request message and send
  one or more response messages back via the transport. Return values are
  ignored; all output goes through the transport.
- **Middleware** — higher-order functions that compose handlers, forming a
  pipeline. All built-in nREPL functionality (eval, session management, etc.)
  is implemented as middleware.

---

## 2. Transport: Bencode over TCP

### 2.1 Connection

Clients connect via plain **TCP**. The server writes its listening address to a
`.nrepl-port` file in the project root (just the port number as ASCII text).
The default hostname is `localhost`.

There is **no handshake** after the TCP connection is established — the client
immediately begins sending bencode-encoded request messages.

### 2.2 Bencode Format

Bencode is a simple self-delimiting encoding with four types:

| Type        | Syntax                              | Example                     |
|-------------|-------------------------------------|-----------------------------|
| Byte string | `<length>:<bytes>`                  | `4:eval`                    |
| Integer     | `i<decimal>e`                       | `i42e`, `i-7e`              |
| List        | `l<item>...<item>e`                 | `l4:spam4:eggse`            |
| Dictionary  | `d<key><value>...<key><value>e`     | `d3:foo3:bare`              |

Dictionary keys **must be byte strings** and are conventionally sorted
lexicographically (required by the spec; the nREPL Clojure implementation does
this). Values may be any bencode type.

A complete nREPL message is exactly **one bencode dictionary**. Multiple
messages are sent as a stream of consecutive dictionaries on the same TCP
connection (no framing, no length prefix beyond bencode's own self-delimiting
structure).

### 2.3 Type Mapping (nREPL ↔ Bencode)

When nREPL encodes Clojure values to bencode and back, the following
conversions apply:

| Clojure (nREPL internal) | → Bencode   | Bencode → Clojure (client receives) |
|--------------------------|-------------|--------------------------------------|
| String                   | String      | String                               |
| Keyword                  | String      | String (keyword-prefix stripped)     |
| Symbol                   | String      | String                               |
| Integer / Long           | Integer     | Integer                              |
| Map                      | Dictionary  | Map                                  |
| Vector                   | List        | Vector                               |
| List                     | List        | Vector                               |
| Set                      | List        | Vector                               |

**Important for Go clients:**
- All dictionary **keys** arrive as byte strings; treat them as plain strings
  (`:op` → `"op"`, `:status` → `"status"`, etc.).
- The server performs `keywordize-keys` on incoming messages, so a Go client
  may send keys as plain strings (`"op"`, `"code"`, etc.) without colons.
- **Status** values arrive as a **list of strings** (e.g. `["done"]`,
  `["error", "done"]`). Treat status as a set of string tokens.
- Integer values are 64-bit signed.
- Byte strings are raw bytes; nREPL uses **UTF-8** encoding for all text.

### 2.4 Message Stream

- The connection carries a **bidirectional stream** of bencode dictionaries.
- **Requests** flow client → server; **responses** flow server → client.
- Multiple in-flight requests are allowed on a single connection; responses are
  matched to requests via the `:id` field.
- The server may also send **unsolicited messages** (e.g. output captured from
  futures started by earlier evals).

---

## 3. Messages

### 3.1 Request Messages

Every message sent by the client is a **request**. Required fields:

| Field      | Type   | Description                                              |
|------------|--------|----------------------------------------------------------|
| `op`       | string | Operation name (e.g. `"eval"`, `"clone"`)                |
| `id`       | string | Client-generated opaque unique ID for this request       |
| `session`  | string | Session ID (required by most ops; optional for stateless ops) |

The `id` field is technically optional per the spec but **strongly recommended**
for all requests. Using sequential integers per session is a common convention.

Additional fields are op-specific (see §5).

### 3.2 Response Messages

The server sends **one or more** response messages per request. Fields present
in every response:

| Field     | Type           | Description                                          |
|-----------|----------------|------------------------------------------------------|
| `id`      | string         | Echoes the `id` from the originating request         |
| `session` | string         | The session ID this response belongs to              |
| `status`  | list of string | Present on the **final** response for a request (and sometimes intermediate ones) |

Common status tokens:

| Token                     | Meaning                                                  |
|---------------------------|----------------------------------------------------------|
| `done`                    | Request fully processed; no more responses will follow   |
| `error`                   | A general error occurred                                 |
| `unknown-op`              | The server does not recognize the requested `op`         |
| `need-input`              | Session's `*in*` needs content to proceed (see `stdin`) |
| `interrupted`             | An `interrupt` succeeded                                 |
| `session-idle`            | `interrupt` found no running evaluation                  |
| `interrupt-id-mismatch`   | `interrupt-id` did not match the running request         |
| `session-ephemeral`       | Cannot interrupt an ephemeral session                    |
| `namespace-not-found`     | Named namespace does not exist                           |
| `nrepl.middleware.print/truncated` | Print quota was exceeded                       |

A response may carry multiple status tokens simultaneously, e.g.
`["error", "done"]`.

**A response without a `status` field** is an intermediate response (more
responses for this request will follow).

---

## 4. Sessions

### 4.1 Session Types

**Ephemeral sessions** are created automatically when a request arrives without
a `session` field. They exist only for that single message; dynamic bindings are
not persisted. Ephemeral sessions cannot be interrupted.

**Persistent (long-lived) sessions** are created explicitly via the `clone` op.
They:
- Persist Clojure dynamic bindings (e.g. `*ns*`, `*1`, `*2`, `*3`, `*e`) across
  multiple requests.
- Guarantee **serial execution** — requests within a session are processed one
  at a time.
- Can be interrupted.
- Must be closed explicitly with the `close` op when no longer needed.

### 4.2 Typical Client Session Lifecycle

```
1. TCP connect to host:port
2. Send {op: "clone", id: "1"}
   Receive {id: "1", session: <tmp>, new-session: "<session-id>", status: ["done"]}
3. Send {op: "describe", id: "2", session: "<session-id>"}
   Receive capabilities/version info
4. Send {op: "eval", id: "3", session: "<session-id>", code: "(+ 1 2)"}
   Receive one or more responses, last has status ["done"]
5. Send {op: "close", id: "N", session: "<session-id>"}
   Receive {id: "N", session: "<session-id>", status: ["done"]}
6. TCP disconnect
```

### 4.3 Multiple Sessions Best Practice

Maintain at least **two sessions**:
- A **user session** for interactive eval (accumulates `*1`, `*2`, `*3`, `*e`).
- A **tooling session** for background ops (completions, lookups) that must not
  pollute user state.

### 4.4 Cloning a Session

`clone` accepts an optional `session` parameter. If provided, the new session
inherits all dynamic bindings from the source — useful for forking a snapshot
of current state.

---

## 5. Operations (Ops)

All ops are part of the default middleware stack unless otherwise noted.

---

### `clone`

Creates a new persistent session.

**Request:**

| Field            | Required | Description                                          |
|------------------|----------|------------------------------------------------------|
| `op`             | yes      | `"clone"`                                            |
| `session`        | no       | Source session to clone bindings from; omit for defaults |
| `client-name`    | no       | Name of the connecting client (e.g. `"my-nrepl-go"`) |
| `client-version` | no       | Version string of the connecting client              |

**Response:**

| Field         | Description                      |
|---------------|----------------------------------|
| `new-session` | ID of the newly created session  |
| `status`      | `["done"]`                       |

---

### `close`

Closes a session, releasing its thread and bindings.

**Request:**

| Field     | Required | Description              |
|-----------|----------|--------------------------|
| `op`      | yes      | `"close"`                |
| `session` | yes      | Session ID to close      |

**Response:** `status: ["done"]`

---

### `describe`

Returns a directory of all ops supported by the server, plus version
information.

**Request:**

| Field      | Required | Description                                          |
|------------|----------|------------------------------------------------------|
| `op`       | yes      | `"describe"`                                         |
| `verbose?` | no       | If `"true"`, include per-op documentation detail     |

**Response:**

| Field        | Description                                                        |
|--------------|--------------------------------------------------------------------|
| `ops`        | Map of op-name → info map (keys: `doc`, `requires`, `optional`, `returns`) |
| `versions`   | Map of component → version map (keys: `major`, `minor`, `incremental`, `version-string`). Common components: `"nrepl"`, `"clojure"`, `"java"` |
| `aux`        | Map of auxiliary data contributed by active middleware             |
| `middleware` | List of active middleware, inside-out order                        |
| `status`     | `["done"]`                                                         |

---

### `eval`

Evaluates code in a session. Short-circuits on first read error; each form in
the code string is evaluated independently (so later forms still run if an
earlier one throws).

**Request:**

| Field       | Required | Description                                                     |
|-------------|----------|-----------------------------------------------------------------|
| `op`        | yes      | `"eval"`                                                        |
| `code`      | yes      | Source code string to evaluate (may contain multiple forms)     |
| `session`   | yes      | Session ID                                                      |
| `id`        | recommended | Message ID (used to correlate responses and for `interrupt`) |
| `ns`        | no       | Namespace to evaluate in (must already exist); defaults to session `*ns*` |
| `file`      | no       | Full file path (sets `*file*`; used for error reporting)        |
| `file-name` | no       | Short filename (e.g. `"core.clj"`)                              |
| `line`      | no       | Line number of `code` within `file`                             |
| `column`    | no       | Column number of `code` within `file`                           |
| `eval`      | no       | Fully-qualified var name to use as the eval function instead of `clojure.core/eval` |
| `read-cond` | no       | Reader conditional options                                      |

Print middleware options (all optional):

| Field                              | Description                                                         |
|------------------------------------|---------------------------------------------------------------------|
| `nrepl.middleware.print/print`     | Fully-qualified var name of custom print function `[value writer options]` |
| `nrepl.middleware.print/options`   | Map of options passed to the print function                         |
| `nrepl.middleware.print/stream?`   | If truthy, stream printed value over multiple messages              |
| `nrepl.middleware.print/buffer-size` | Buffer size for streaming (default: 1024)                         |
| `nrepl.middleware.print/quota`     | Hard byte limit per printed value                                   |
| `nrepl.middleware.print/keys`      | Keys whose values should be printed (default: `["value"]`)         |

Caught middleware options (all optional):

| Field                               | Description                                                       |
|-------------------------------------|-------------------------------------------------------------------|
| `nrepl.middleware.caught/caught`    | Fully-qualified var to use for conveying interactive errors       |
| `nrepl.middleware.caught/print?`    | If truthy, return printed exception in response                   |

**Response (zero or more intermediate, then one final):**

| Field      | When present                                                             |
|------------|--------------------------------------------------------------------------|
| `value`    | Printed result of a form evaluation; one message per form. Absent if exception. |
| `ns`       | Stringified `*ns*` after successful evaluation                           |
| `out`      | Content written to `*out*` since last flush                              |
| `err`      | Content written to `*err*` since last flush                              |
| `ex`       | Exception type name (e.g. `"class java.lang.ArithmeticException"`) if eval threw |
| `root-ex`  | Root cause exception type, if different from `ex`                        |
| `status`   | Final message carries `["done"]`; error cases may carry additional tokens |

**Response flow example for `(+ 1 2)`:**
```
← {id: "3", session: "abc", value: "3", ns: "user"}
← {id: "3", session: "abc", status: ["done"]}
```

**Response flow example for `(println "hi") (/ 1 0)`:**
```
← {id: "3", session: "abc", out: "hi\n"}
← {id: "3", session: "abc", value: "nil", ns: "user"}
← {id: "3", session: "abc", err: "...", ex: "class java.lang.ArithmeticException", root-ex: "class java.lang.ArithmeticException"}
← {id: "3", session: "abc", status: ["done"]}
```

---

### `stdin`

Feeds content into `*in*` for the current session, used when evaluated code
tries to read from stdin.

**Request:**

| Field     | Required | Description                        |
|-----------|----------|------------------------------------|
| `op`      | yes      | `"stdin"`                          |
| `session` | yes      | Session ID                         |
| `stdin`   | yes      | String content to add to `*in*`    |

**Response:**

| Field    | Description                                                              |
|----------|--------------------------------------------------------------------------|
| `status` | May include `"need-input"` if the session is blocking on a read. `["done"]` when accepted. |

---

### `interrupt`

Attempts to interrupt a running evaluation in a session.

**Request:**

| Field          | Required | Description                                              |
|----------------|----------|----------------------------------------------------------|
| `op`           | yes      | `"interrupt"`                                            |
| `session`      | yes      | Session containing the evaluation to interrupt           |
| `interrupt-id` | no       | The `id` of the specific request to interrupt            |

**Response:**

| Field    | Description                                                              |
|----------|--------------------------------------------------------------------------|
| `status` | One of: `["interrupted", "done"]`, `["session-idle", "done"]`, `["interrupt-id-mismatch", "done"]`, `["session-ephemeral", "done"]` |

---

### `load-file`

Loads a complete file of code. Uses `file-name` and `file-path` for source
metadata. Delegates to the `eval` middleware internally.

**Request:**

| Field       | Required | Description                                                  |
|-------------|----------|--------------------------------------------------------------|
| `op`        | yes      | `"load-file"`                                                |
| `file`      | yes      | Full contents of the file as a string                        |
| `file-name` | no       | Short filename (e.g. `"io.clj"`)                             |
| `file-path` | no       | Source-path-relative path (e.g. `"clojure/java/io.clj"`)    |

Accepts the same print/caught middleware options as `eval`.

**Response:** Same shape as `eval` (`value`, `ex`, `root-ex`, `status`).

---

### `completions`

Returns completion candidates for a prefix string.

**Request:**

| Field         | Required | Description                                                   |
|---------------|----------|---------------------------------------------------------------|
| `op`          | yes      | `"completions"`                                               |
| `prefix`      | yes      | String prefix to complete                                     |
| `session`     | yes      | Session ID (determines namespace context)                     |
| `ns`          | no       | Namespace to complete in; defaults to session `*ns*`          |
| `complete-fn` | no       | Fully-qualified var name of custom completion function        |
| `options`     | no       | Map of options; recognized key: `extra-metadata` (values: `"arglists"`, `"docs"`) |

**Response:**

| Field         | Description                                                              |
|---------------|--------------------------------------------------------------------------|
| `completions` | List of candidate maps. Each map has `candidate` (string) and `type` (string, e.g. `"var"`, `"namespace"`, `"class"`). Var candidates also have `ns`. |
| `status`      | `["done"]`                                                               |

---

### `lookup`

Looks up metadata for a symbol.

**Request:**

| Field       | Required | Description                                                   |
|-------------|----------|---------------------------------------------------------------|
| `op`        | yes      | `"lookup"`                                                    |
| `sym`       | yes      | Symbol name to look up                                        |
| `session`   | yes      | Session ID                                                    |
| `ns`        | no       | Namespace for resolution; defaults to session `*ns*`          |
| `lookup-fn` | no       | Fully-qualified var name of custom lookup function            |

**Response:**

| Field    | Description                                       |
|----------|---------------------------------------------------|
| `info`   | Map of symbol metadata (varies by symbol type)    |
| `status` | `["done"]`                                        |

---

### `ls-sessions`

Lists all active (persistent) session IDs.

**Request:**

| Field | Required | Description      |
|-------|----------|------------------|
| `op`  | yes      | `"ls-sessions"`  |

**Response:**

| Field      | Description                          |
|------------|--------------------------------------|
| `sessions` | List of session ID strings           |
| `status`   | `["done"]`                           |

---

### `forward-system-output`

Enables forwarding of `System/out` and `System/err` to the client. This is a
Clojure-specific op, not part of the general nREPL spec.

**Request:** `{op: "forward-system-output"}`
**Response:** `{status: ["done"]}`

---

## 6. Message ID and Request/Response Correlation

- The client **should** include a unique `id` in every request.
- Every response includes the `id` of the originating request.
- A single request may produce **N intermediate responses** (no `status` or
  non-terminal status) followed by exactly **one final response** containing
  `"done"` in its `status` list.
- Responses for different in-flight requests may be **interleaved** on the
  wire; use `id` to demultiplex them.
- After `"done"` is received, additional responses for the same `id` may still
  arrive (e.g. output from futures spawned during eval). Clients should handle
  this gracefully.

---

## 7. Error Handling

- If the server does not recognize an op: response with `status: ["unknown-op", "done"]`.
- If a request is malformed: response with `status: ["error", "done"]` possibly
  with additional context.
- Eval exceptions: `ex` and `root-ex` fields in the response, `status: ["done"]`
  (the error is reported but evaluation completes).
- Transport-level errors: TCP connection may be closed by either side.

---

## 8. Wire-Level Example

Sending `{:op "eval" :id "1" :session "abc" :code "(+ 1 2)"}` in bencode:

```
d
4:code
7:(+ 1 2)
2:id
1:1
2:op
4:eval
7:session
3:abc
e
```

(Keys are sorted lexicographically: `code`, `id`, `op`, `session`.)

Response `{:id "1" :session "abc" :value "3" :ns "user"}`:

```
d
2:id
1:1
2:ns
4:user
7:session
3:abc
5:value
1:3
e
```

Final response `{:id "1" :session "abc" :status ["done"]}`:

```
d
2:id
1:1
7:session
3:abc
6:status
l4:donee
e
```

---

## 9. Implementation Notes for Go Clients

1. **Bencode library**: Implement or use a bencode encoder/decoder that handles
   all four types. Keys in outgoing dictionaries should be sorted
   lexicographically (required by spec).

2. **Message dispatch**: Maintain a `map[string]chan Response` keyed by message
   `id`. On receiving a message, route it to the appropriate channel. For
   messages without an `id` (server-initiated), use a separate channel.

3. **Session management**: Track open sessions and close them on disconnect.
   Consider a `Session` type that wraps a connection + session ID and exposes
   `Eval`, `Completions`, etc. methods.

4. **Streaming responses**: `Eval` returns a channel or calls a callback for
   each intermediate response; the caller reads until `"done"` appears in
   `status`.

5. **Concurrency**: Multiple goroutines may send requests simultaneously on one
   connection. Use a mutex or dedicated goroutine for the write side.

6. **Port discovery**: Read `.nrepl-port` from the current working directory
   (or parent directories) to find the server port. The format is just a bare
   decimal port number, optionally followed by a newline.

7. **Status as a set**: Parse the bencode list in `status` into a `map[string]bool`
   or `[]string` and check membership; never assume it contains exactly one
   element.

8. **EOF handling**: A closed TCP connection mid-message means the server has
   died. Surface this as an error on all in-flight request channels.
