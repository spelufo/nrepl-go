package nrepl

import (
	"os"
	"testing"
)

func dialTest(t *testing.T) *Conn {
	t.Helper()
	addr := os.Getenv("NREPL_ADDR")
	if addr == "" {
		addr = "localhost:7888"
	}
	conn, err := Dial(addr)
	if err != nil {
		t.Skipf("skipping: cannot connect to nREPL at %s: %v", addr, err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestIntegrationDescribe(t *testing.T) {
	conn := dialTest(t)
	if err := conn.Send(Message{"op": "describe", "id": "1"}); err != nil {
		t.Fatal(err)
	}
	resp, err := conn.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasStatus("done") {
		t.Errorf("expected done status, got %v", resp.Status())
	}
	if _, ok := resp.Msg["ops"]; !ok {
		t.Error("describe response missing 'ops'")
	}
}

func TestIntegrationEval(t *testing.T) {
	conn := dialTest(t)
	if err := conn.Send(Message{"op": "eval", "code": "(+ 1 2)", "id": "2"}); err != nil {
		t.Fatal(err)
	}

	var gotValue bool
	var gotDone bool
	for !gotDone {
		resp, err := conn.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if v, ok := resp.Value(); ok {
			if v != "3" {
				t.Errorf("value = %q, want %q", v, "3")
			}
			gotValue = true
		}
		if resp.HasStatus("done") {
			gotDone = true
		}
	}
	if !gotValue {
		t.Error("never received value response")
	}
}
