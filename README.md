# nrepl-go

A Go client library and CLI for the [nREPL](https://nrepl.org/) protocol.


## Installation

```
# from source
go install ./cmd/nrepl-send

# from the internet
go install github.com/spelufo/nrepl-go
```

There's also an extension for the pi coding agent:

```
cp .pi/extensions/nrepl.ts ~/.pi/extensions/
```

## Usage

By default, `nrepl-send` connects via the `.nrepl-port` file in the current directory.

```bash
# Evaluate an expression
nrepl-send eval "(+ 1 2)"

# Evaluate in a specific namespace
nrepl-send eval "(map inc [1 2 3])" -n my.app

# Get symbol completions
nrepl-send completions "map"

# Describe the nREPL server
nrepl-send describe

# Connect to a specific host/port
nrepl-send eval "(+ 1 2)" --host localhost --port 7888
```


## License

MIT
