package nrepl

import (
	"testing"
)

func newTestClient(t *testing.T) *Client {
	t.Helper()
	conn := dialTest(t)
	client, err := NewClient(conn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestClientEval(t *testing.T) {
	c := newTestClient(t)
	ch, err := c.Eval("(+ 1 2)")
	if err != nil {
		t.Fatal(err)
	}
	var gotValue bool
	for resp := range ch {
		if v, ok := resp.Value(); ok {
			if v != "3" {
				t.Errorf("value = %q, want %q", v, "3")
			}
			gotValue = true
		}
	}
	if !gotValue {
		t.Error("never received value")
	}
}

func TestClientEvalWithOutput(t *testing.T) {
	c := newTestClient(t)
	ch, err := c.Eval(`(println "hello")`)
	if err != nil {
		t.Fatal(err)
	}
	var gotOut bool
	for resp := range ch {
		if out, ok := resp.Out(); ok {
			if out != "hello\n" {
				t.Errorf("out = %q", out)
			}
			gotOut = true
		}
	}
	if !gotOut {
		t.Error("never received out")
	}
}

func TestClientDescribe(t *testing.T) {
	c := newTestClient(t)
	resp, err := c.Describe()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := resp.Msg["ops"]; !ok {
		t.Error("describe missing ops")
	}
}

func TestClientLsSessions(t *testing.T) {
	c := newTestClient(t)
	sessions, err := c.LsSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) == 0 {
		t.Error("no sessions")
	}
	// Our session should be in the list
	found := false
	for _, s := range sessions {
		if s == c.Session() {
			found = true
		}
	}
	if !found {
		t.Errorf("our session %s not in list", c.Session())
	}
}

func TestClientConcurrentEvals(t *testing.T) {
	c := newTestClient(t)

	ch1, err := c.Eval("(+ 1 1)")
	if err != nil {
		t.Fatal(err)
	}
	ch2, err := c.Eval("(+ 2 2)")
	if err != nil {
		t.Fatal(err)
	}

	collect := func(ch <-chan Response) string {
		for resp := range ch {
			if v, ok := resp.Value(); ok {
				return v
			}
		}
		return ""
	}

	v1 := collect(ch1)
	v2 := collect(ch2)

	if v1 != "2" {
		t.Errorf("eval 1: got %q, want %q", v1, "2")
	}
	if v2 != "4" {
		t.Errorf("eval 2: got %q, want %q", v2, "4")
	}
}
