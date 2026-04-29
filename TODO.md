# TODO

## Phase 1: bencode codec

- [x] Encode strings
- [x] Encode integers
- [x] Encode lists
- [x] Encode dictionaries (sorted keys)
- [x] Decode strings
- [x] Decode integers
- [x] Decode lists
- [x] Decode dictionaries
- [x] Round-trip tests with nREPL wire examples from spec

## Phase 2: Connection & types

- [x] `Message` type and `Response` accessors
- [x] `Conn` — Dial, Send, Recv, Close
- [x] `FindPortFile` — walk up directories for `.nrepl-port`
- [x] `DialFromPortFile`
- [x] Integration test: connect to live server, send `describe`

## Phase 3: Client (multiplexed)

- [x] Background recv loop + per-id dispatch
- [x] `Send` — id generation, channel registration
- [x] `Clone` / `Close` session management
- [x] `Eval` — stream responses until done
- [x] `Describe`, `Completions`, `Lookup`, `Interrupt`, `LsSessions`
- [x] Integration test: eval round-trip
- [x] Integration test: concurrent evals

## Phase 4: Document the client library

- [x] Write README.md with installation, quick start, API reference, and examples


## Phase 5: CLI

- [x] Cobra scaffold + global flags (`--host`, `--port`, `--port-file`)
- [x] `nrepl eval <code>`
- [x] `nrepl describe`
- [x] `nrepl completions <prefix>`
- [x] `nrepl repl` (interactive loop)
