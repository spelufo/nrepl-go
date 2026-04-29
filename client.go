package nrepl

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
)

// Client is a multiplexed, session-aware nREPL client.
type Client struct {
	conn    *Conn
	session string

	nextID   atomic.Int64
	mu       sync.Mutex
	pending  map[string]chan Response
	done     chan struct{}
	closeErr error
}

// NewClient starts a recv loop and clones an initial session.
func NewClient(conn *Conn) (*Client, error) {
	c := &Client{
		conn:    conn,
		pending: make(map[string]chan Response),
		done:    make(chan struct{}),
	}
	go c.recvLoop()

	session, err := c.Clone()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("nrepl: clone session: %w", err)
	}
	c.session = session
	return c, nil
}

func (c *Client) genID() string {
	return strconv.FormatInt(c.nextID.Add(1), 10)
}

func (c *Client) register(id string) chan Response {
	ch := make(chan Response, 16)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	return ch
}

func (c *Client) recvLoop() {
	defer close(c.done)
	for {
		resp, err := c.conn.Recv()
		if err != nil {
			c.mu.Lock()
			c.closeErr = err
			for _, ch := range c.pending {
				close(ch)
			}
			c.pending = nil
			c.mu.Unlock()
			return
		}

		id := resp.ID()
		c.mu.Lock()
		ch := c.pending[id]
		c.mu.Unlock()

		if ch != nil {
			ch <- resp
			if resp.HasStatus("done") {
				c.mu.Lock()
				delete(c.pending, id)
				c.mu.Unlock()
				close(ch)
			}
		}
	}
}

// Send sends an op with optional extra fields and returns a channel of responses.
// The channel is closed after a "done" status is received.
func (c *Client) Send(op string, fields Message) (<-chan Response, error) {
	id := c.genID()
	msg := Message{"op": op, "id": id}
	if c.session != "" {
		msg["session"] = c.session
	}
	for k, v := range fields {
		msg[k] = v
	}
	ch := c.register(id)
	if err := c.conn.Send(msg); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		close(ch)
		return nil, err
	}
	return (<-chan Response)(ch), nil
}

// sendOne sends an op and collects all responses, returning the first non-done response.
func (c *Client) sendOne(op string, fields Message) (Response, error) {
	ch, err := c.Send(op, fields)
	if err != nil {
		return Response{}, err
	}
	var result Response
	for resp := range ch {
		if result.Msg == nil {
			result = resp
		}
		// If this is the one with the actual data (not just done), prefer it.
		if !resp.HasStatus("done") && len(resp.Msg) > 2 {
			result = resp
		}
	}
	if result.Msg == nil {
		return Response{}, fmt.Errorf("nrepl: no response for %s", op)
	}
	return result, nil
}

// Eval sends code for evaluation. Returns a channel of responses.
func (c *Client) Eval(code string) (<-chan Response, error) {
	return c.Send("eval", Message{"code": code})
}

// EvalIn sends code for evaluation in the given namespace. Returns a channel of responses.
func (c *Client) EvalIn(code, ns string) (<-chan Response, error) {
	return c.Send("eval", Message{"code": code, "ns": ns})
}

// Clone creates a new session, returning its ID.
func (c *Client) Clone() (string, error) {
	id := c.genID()
	msg := Message{"op": "clone", "id": id}
	if c.session != "" {
		msg["session"] = c.session
	}
	ch_ := c.register(id)
	if err := c.conn.Send(msg); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		close(ch_)
		return "", err
	}
	ch := (<-chan Response)(ch_)
	for resp := range ch {
		if s, ok := resp.get("new-session"); ok {
			for range ch {
			}
			return s, nil
		}
	}
	return "", fmt.Errorf("nrepl: clone did not return a session id")
}

// Describe returns server info.
func (c *Client) Describe() (Response, error) {
	return c.sendOne("describe", nil)
}

// Completions returns completions for prefix.
func (c *Client) Completions(prefix string) (Response, error) {
	return c.sendOne("completions", Message{"prefix": prefix})
}

// Lookup returns info about a symbol.
func (c *Client) Lookup(sym string) (Response, error) {
	return c.sendOne("lookup", Message{"sym": sym})
}

// Interrupt sends an interrupt for the given eval ID.
func (c *Client) Interrupt(evalID string) (Response, error) {
	return c.sendOne("interrupt", Message{"interrupt-id": evalID})
}

// LsSessions returns all active session IDs.
func (c *Client) LsSessions() ([]string, error) {
	resp, err := c.sendOne("ls-sessions", nil)
	if err != nil {
		return nil, err
	}
	raw, ok := resp.Msg["sessions"]
	if !ok {
		return nil, fmt.Errorf("nrepl: ls-sessions missing 'sessions'")
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("nrepl: sessions is %T, not list", raw)
	}
	out := make([]string, len(list))
	for i, v := range list {
		switch s := v.(type) {
		case []byte:
			out[i] = string(s)
		case string:
			out[i] = s
		default:
			out[i] = fmt.Sprint(v)
		}
	}
	return out, nil
}

// Close closes the client session and connection.
func (c *Client) Close() error {
	if c.session != "" {
		// Send close and wait for the server's response before closing the
		// TCP connection, otherwise the server gets a SocketException.
		ch, err := c.Send("close", nil)
		if err == nil {
			for range ch {
			}
		}
	}
	err := c.conn.Close()
	<-c.done
	return err
}

// Session returns the current session ID.
func (c *Client) Session() string {
	return c.session
}
