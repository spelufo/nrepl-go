# nrepl-go Implementation Plan

## Module & Layout

```
github.com/spelufo/nrepl-go   # client library (root package)
├── bencode/                   # bencode codec
├── cli/                       # cobra CLI tool (main)
├── references/                # protocol spec (not shipped)
└── go.mod
```

Import paths:
- `github.com/spelufo/nrepl-go` — client library
- `github.com/spelufo/nrepl-go/bencode` — bencode codec (also usable standalone)

## Phase 1: bencode codec (`bencode/`)

Implement encoder/decoder for the four bencode types.

**Types:**
- Strings → `[]byte`
- Integers → `int64`
- Lists → `[]any`
- Dictionaries → `map[string]any` (keys sorted on encode)

**API:**
```go
func Encode(w io.Writer, v any) error
func Decode(r io.Reader) (any, error)
```

Decode reads one complete bencode value from the stream. Encode writes one. These compose naturally for the nREPL message stream (consecutive dictionaries).

## Phase 2: Client library (root package `nrepl`)

### Core types

```go
// Message is an nREPL message (request or response).
type Message map[string]any

// Response adds convenience accessors over Message.
type Response struct {
    Msg Message
}
func (r Response) ID() string
func (r Response) Session() string
func (r Response) Status() []string
func (r Response) HasStatus(token string) bool
func (r Response) Value() (string, bool)
func (r Response) Out() (string, bool)
func (r Response) Err() (string, bool)
func (r Response) Ex() (string, bool)
```

### Conn — low-level connection

```go
type Conn struct { ... }

func Dial(addr string) (*Conn, error)
func DialFromPortFile(dir string) (*Conn, error)  // reads .nrepl-port

func (c *Conn) Send(msg Message) error            // bencode-encode + write (mutex-protected)
func (c *Conn) Recv() (Response, error)            // bencode-decode one response
func (c *Conn) Close() error
```

`Conn` is a thin wrapper: send is mutex-protected, recv is single-reader. A higher-level dispatch layer handles demuxing.

### Client — multiplexed, session-aware

```go
type Client struct { ... }

func NewClient(conn *Conn) (*Client, error)        // starts recv loop, clones initial session

func (c *Client) Send(op string, fields Message) (<-chan Response, error)
    // Assigns id, sends, returns channel that yields responses until "done".

func (c *Client) Eval(code string) (<-chan Response, error)
func (c *Client) Clone() (sessionID string, err error)
func (c *Client) Close() error
func (c *Client) Describe() (Response, error)
func (c *Client) Completions(prefix string) (Response, error)
func (c *Client) Lookup(sym string) (Response, error)
func (c *Client) Interrupt(evalID string) (Response, error)
func (c *Client) LsSessions() ([]string, error)
```

Internally:
- Background goroutine reads from `Conn.Recv()` and dispatches to per-id channels (`map[string]chan Response`).
- `Send` generates sequential string IDs, registers a channel, sends the message.
- Channels are closed after `"done"` status is received.
- On EOF/error, all pending channels receive the error.

### Port file discovery

```go
func FindPortFile(startDir string) (string, error)
    // Walks up from startDir looking for .nrepl-port, returns "host:port".
```

## Phase 3: CLI (`cli/`)

Cobra-based CLI at `cli/main.go`.

### Commands

**`nrepl eval <code>`** — Connect, eval code, print output/values/errors, exit.
```
$ nrepl eval '(+ 1 2)'
3
```
Flags: `--port`, `--host`, `--port-file`.

**`nrepl repl`** — Interactive REPL loop. Reads lines from stdin, evals, prints responses. Ctrl-C sends interrupt. Ctrl-D exits.

**`nrepl describe`** — Print server info (versions, ops).

**`nrepl completions <prefix>`** — Print completions.

### Global flags
- `--host` (default `localhost`)
- `--port` (default: read from `.nrepl-port`)
- `--port-file` (explicit path to port file)

## Implementation Order

1. `bencode/` — encode + decode + tests
2. Root package — `Message`, `Response`, `Conn`, port file discovery + tests
3. Root package — `Client` (multiplexed dispatch, session management) + tests
4. `cli/` — `eval` command (proves the full stack end-to-end)
5. `cli/` — `repl`, `describe`, `completions` commands

## Testing Strategy

- `bencode/`: Unit tests with known byte sequences from the spec.
- `Conn`/`Client`: Integration tests against a real nREPL server (skipped if no server available). Use a test helper that starts `clojure -M:nrepl` or checks `.nrepl-port`.
- CLI: Manual testing initially.

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- No bencode library — we write our own (it's ~150 lines and avoids a dependency for a core concern).
