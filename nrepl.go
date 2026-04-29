// Package nrepl provides a client library for the nREPL protocol.
package nrepl

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spelufo/nrepl-go/bencode"
)

// Message is an nREPL message (request or response).
type Message map[string]any

// Response adds convenience accessors over Message.
type Response struct {
	Msg Message
}

func (r Response) get(key string) (string, bool) {
	v, ok := r.Msg[key]
	if !ok {
		return "", false
	}
	switch val := v.(type) {
	case string:
		return val, true
	case []byte:
		return string(val), true
	default:
		return fmt.Sprint(val), true
	}
}

func (r Response) ID() string      { s, _ := r.get("id"); return s }
func (r Response) Session() string  { s, _ := r.get("session"); return s }
func (r Response) Value() (string, bool) { return r.get("value") }
func (r Response) Out() (string, bool)   { return r.get("out") }
func (r Response) Err() (string, bool)   { return r.get("err") }
func (r Response) Ex() (string, bool)    { return r.get("ex") }

func (r Response) Status() []string {
	v, ok := r.Msg["status"]
	if !ok {
		return nil
	}
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, len(list))
	for i, item := range list {
		switch val := item.(type) {
		case []byte:
			out[i] = string(val)
		case string:
			out[i] = val
		default:
			out[i] = fmt.Sprint(val)
		}
	}
	return out
}

func (r Response) HasStatus(token string) bool {
	for _, s := range r.Status() {
		if s == token {
			return true
		}
	}
	return false
}

// Conn is a low-level nREPL connection.
type Conn struct {
	conn net.Conn
	mu   sync.Mutex // protects Send
}

// Dial connects to an nREPL server at the given "host:port" address.
func Dial(addr string) (*Conn, error) {
	nc, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Conn{conn: nc}, nil
}

// DialFromPortFile reads .nrepl-port from dir and connects to localhost:port.
func DialFromPortFile(dir string) (*Conn, error) {
	addr, err := ReadPortFile(dir)
	if err != nil {
		return nil, err
	}
	return Dial(addr)
}

// Send writes a single nREPL message. It is safe for concurrent use.
func (c *Conn) Send(msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Convert Message to map[string]any for bencode.
	return bencode.Encode(c.conn, map[string]any(msg))
}

// Recv reads a single nREPL response from the connection.
func (c *Conn) Recv() (Response, error) {
	v, err := bencode.Decode(c.conn)
	if err != nil {
		return Response{}, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return Response{}, fmt.Errorf("nrepl: expected dict, got %T", v)
	}
	return Response{Msg: Message(m)}, nil
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// ReadPortFile reads .nrepl-port from dir and returns "localhost:port".
func ReadPortFile(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, ".nrepl-port"))
	if err != nil {
		return "", err
	}
	port := strings.TrimSpace(string(data))
	return "localhost:" + port, nil
}

// FindPortFile walks up from startDir looking for .nrepl-port and returns "localhost:port".
func FindPortFile(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		p := filepath.Join(dir, ".nrepl-port")
		if _, err := os.Stat(p); err == nil {
			return ReadPortFile(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("nrepl: .nrepl-port not found (searched up from %s)", startDir)
		}
		dir = parent
	}
}
