# nrepl-go

A Go client library for the [nREPL](https://nrepl.org/) protocol.

## Installation

```
go get github.com/spelufo/nrepl-go
```

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	nrepl "github.com/spelufo/nrepl-go"
)

func main() {
	// Connect using .nrepl-port file discovery
	conn, err := nrepl.DialFromPortFile(".")
	if err != nil {
		log.Fatal(err)
	}

	client, err := nrepl.NewClient(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Evaluate code
	ch, err := client.Eval("(+ 1 2)")
	if err != nil {
		log.Fatal(err)
	}
	for resp := range ch {
		if val, ok := resp.Value(); ok {
			fmt.Println(val) // "3"
		}
		if out, ok := resp.Out(); ok {
			fmt.Print(out)
		}
		if errMsg, ok := resp.Err(); ok {
			fmt.Print(errMsg)
		}
	}
}
```

## API Reference

### Connection

```go
// Dial connects to an nREPL server at "host:port".
func Dial(addr string) (*Conn, error)

// DialFromPortFile reads .nrepl-port from dir and connects to localhost:port.
func DialFromPortFile(dir string) (*Conn, error)

// FindPortFile walks up from startDir looking for .nrepl-port, returns "localhost:port".
func FindPortFile(startDir string) (string, error)

// ReadPortFile reads .nrepl-port from a specific directory, returns "localhost:port".
func ReadPortFile(dir string) (string, error)
```

### Client

`Client` is a multiplexed, session-aware nREPL client. It starts a background goroutine that dispatches responses to callers by message ID.

```go
// NewClient creates a client from a connection. It starts a recv loop and
// clones an initial session.
func NewClient(conn *Conn) (*Client, error)

// Session returns the current session ID.
func (c *Client) Session() string

// Close closes the session and underlying connection.
func (c *Client) Close() error
```

#### Operations

```go
// Send sends an op with optional extra fields. Returns a channel that yields
// responses until "done" status is received, then closes.
func (c *Client) Send(op string, fields Message) (<-chan Response, error)

// Eval sends code for evaluation. Returns a channel of responses.
func (c *Client) Eval(code string) (<-chan Response, error)

// Describe returns server info (versions, supported ops).
func (c *Client) Describe() (Response, error)

// Completions returns completions for the given prefix.
func (c *Client) Completions(prefix string) (Response, error)

// Lookup returns info about a symbol.
func (c *Client) Lookup(sym string) (Response, error)

// Interrupt interrupts an in-progress eval by its ID.
func (c *Client) Interrupt(evalID string) (Response, error)

// LsSessions returns all active session IDs.
func (c *Client) LsSessions() ([]string, error)

// Clone creates a new session, returning its ID.
func (c *Client) Clone() (string, error)
```

### Response

```go
type Response struct {
	Msg Message // the raw message map
}

func (r Response) ID() string
func (r Response) Session() string
func (r Response) Status() []string
func (r Response) HasStatus(token string) bool
func (r Response) Value() (string, bool)   // evaluation result
func (r Response) Out() (string, bool)     // stdout output
func (r Response) Err() (string, bool)     // stderr output
func (r Response) Ex() (string, bool)      // exception class
```

### Message

```go
// Message is an nREPL message (request or response).
type Message map[string]any
```

### bencode

The `bencode` subpackage is also usable standalone:

```go
import "github.com/spelufo/nrepl-go/bencode"

func bencode.Encode(w io.Writer, v any) error
func bencode.Decode(r io.Reader) (any, error)
```

## Examples

### Streaming eval output

```go
ch, _ := client.Eval(`(do (println "hello") (+ 1 1))`)
for resp := range ch {
	if out, ok := resp.Out(); ok {
		fmt.Print(out) // "hello\n"
	}
	if val, ok := resp.Value(); ok {
		fmt.Println("=>", val) // "=> 2"
	}
}
```

### Using Send for custom ops

```go
ch, _ := client.Send("eval", nrepl.Message{
	"code": "(System/getProperty \"user.dir\")",
	"ns":   "user",
})
for resp := range ch {
	if val, ok := resp.Value(); ok {
		fmt.Println(val)
	}
}
```

## License

MIT
