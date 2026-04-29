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

- [ ] Background recv loop + per-id dispatch
- [ ] `Send` — id generation, channel registration
- [ ] `Clone` / `Close` session management
- [ ] `Eval` — stream responses until done
- [ ] `Describe`, `Completions`, `Lookup`, `Interrupt`, `LsSessions`
- [ ] Integration test: eval round-trip
- [ ] Integration test: concurrent evals

## Phase 4: CLI

- [ ] Cobra scaffold + global flags (`--host`, `--port`, `--port-file`)
- [ ] `nrepl eval <code>`
- [ ] `nrepl describe`
- [ ] `nrepl completions <prefix>`
- [ ] `nrepl repl` (interactive loop)
