# nrepl-go

A Go client library and CLI tool for the [nREPL](https://nrepl.org) protocol (Clojure's network REPL).

## Library

```go
import "github.com/spelufo/nrepl-go"
```

### Connect and eval

```go
client, err := nrepl.DialAndConnect("")  // finds .nrepl-port automatically
if err != nil {
    log.Fatal(err)
}
defer client.Close()

responses, err := client.Eval("(+ 1 2)")
if err != nil {
    log.Fatal(err)
}
for resp := range responses {
    if v, ok := resp.Value(); ok {
        fmt.Println(v)  // "3"
    }
    if out, ok := resp.Out(); ok {
        fmt.Print(out)
    }
    if e, ok := resp.Err(); ok {
        fmt.Fprint(os.Stderr, e)
    }
}
```

### Lower-level access

```go
conn, err := nrepl.Dial("localhost:7888")
defer conn.Close()

conn.Send(nrepl.Message{"op": "clone", "id": "1"})
resp, err := conn.Recv()
sessionID := resp.Msg["new-session"].(string)
```

## CLI

### Install

```
go install github.com/spelufo/nrepl-go/cli@latest
```

### Commands

```sh
# Evaluate code
nrepl eval '(+ 1 2)'

# Interactive REPL
nrepl repl

# Server info
nrepl describe

# Completions
nrepl completions 'map'
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `localhost` | nREPL server host |
| `--port` | from `.nrepl-port` | nREPL server port |
| `--port-file` | auto-discovered | Explicit path to port file |

Port discovery walks up from the current directory looking for `.nrepl-port`.

## Packages

| Package | Description |
|---------|-------------|
| `github.com/spelufo/nrepl-go` | Client library |
| `github.com/spelufo/nrepl-go/bencode` | Standalone bencode codec |
| `github.com/spelufo/nrepl-go/cli` | CLI tool |

## License

MIT
