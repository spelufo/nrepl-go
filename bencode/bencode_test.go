package bencode

import (
	"bytes"
	"reflect"
	"testing"
)

func TestEncodeString(t *testing.T) {
	var buf bytes.Buffer
	Encode(&buf, "spam")
	if got := buf.String(); got != "4:spam" {
		t.Errorf("got %q, want %q", got, "4:spam")
	}
}

func TestEncodeBytes(t *testing.T) {
	var buf bytes.Buffer
	Encode(&buf, []byte("hello"))
	if got := buf.String(); got != "5:hello" {
		t.Errorf("got %q, want %q", got, "5:hello")
	}
}

func TestEncodeEmptyString(t *testing.T) {
	var buf bytes.Buffer
	Encode(&buf, "")
	if got := buf.String(); got != "0:" {
		t.Errorf("got %q, want %q", got, "0:")
	}
}

func TestEncodeInt(t *testing.T) {
	tests := []struct {
		in   any
		want string
	}{
		{42, "i42e"},
		{int64(-7), "i-7e"},
		{0, "i0e"},
	}
	for _, tt := range tests {
		var buf bytes.Buffer
		Encode(&buf, tt.in)
		if got := buf.String(); got != tt.want {
			t.Errorf("Encode(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEncodeList(t *testing.T) {
	var buf bytes.Buffer
	Encode(&buf, []any{"spam", 42})
	if got := buf.String(); got != "l4:spami42ee" {
		t.Errorf("got %q", got)
	}
}

func TestEncodeDict(t *testing.T) {
	var buf bytes.Buffer
	Encode(&buf, map[string]any{"cow": "moo", "a": int64(1)})
	// keys sorted: a, cow
	want := "d1:ai1e3:cow3:mooe"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecodeString(t *testing.T) {
	v, err := Decode(bytes.NewReader([]byte("4:spam")))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(v.([]byte), []byte("spam")) {
		t.Errorf("got %v", v)
	}
}

func TestDecodeInt(t *testing.T) {
	v, err := Decode(bytes.NewReader([]byte("i42e")))
	if err != nil {
		t.Fatal(err)
	}
	if v.(int64) != 42 {
		t.Errorf("got %v", v)
	}
}

func TestDecodeNegativeInt(t *testing.T) {
	v, err := Decode(bytes.NewReader([]byte("i-3e")))
	if err != nil {
		t.Fatal(err)
	}
	if v.(int64) != -3 {
		t.Errorf("got %v", v)
	}
}

func TestDecodeList(t *testing.T) {
	v, err := Decode(bytes.NewReader([]byte("l4:spami42ee")))
	if err != nil {
		t.Fatal(err)
	}
	list := v.([]any)
	if len(list) != 2 {
		t.Fatalf("len = %d", len(list))
	}
	if !bytes.Equal(list[0].([]byte), []byte("spam")) {
		t.Errorf("list[0] = %v", list[0])
	}
	if list[1].(int64) != 42 {
		t.Errorf("list[1] = %v", list[1])
	}
}

func TestDecodeDict(t *testing.T) {
	v, err := Decode(bytes.NewReader([]byte("d3:cow3:moo4:spam4:eggse")))
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	if !bytes.Equal(m["cow"].([]byte), []byte("moo")) {
		t.Errorf("cow = %v", m["cow"])
	}
	if !bytes.Equal(m["spam"].([]byte), []byte("eggs")) {
		t.Errorf("spam = %v", m["spam"])
	}
}

func TestRoundTrip(t *testing.T) {
	// Simulates an nREPL eval request
	msg := map[string]any{
		"op":      "eval",
		"code":    "(+ 1 2)",
		"id":      "1",
		"session": "abc123",
	}
	var buf bytes.Buffer
	if err := Encode(&buf, msg); err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	m := decoded.(map[string]any)
	// Convert []byte values to string for comparison
	result := make(map[string]any)
	for k, v := range m {
		if b, ok := v.([]byte); ok {
			result[k] = string(b)
		} else {
			result[k] = v
		}
	}
	if !reflect.DeepEqual(result, msg) {
		t.Errorf("round-trip failed:\ngot  %v\nwant %v", result, msg)
	}
}

func TestRoundTripNREPLResponse(t *testing.T) {
	// nREPL response with status list
	msg := map[string]any{
		"id":      "1",
		"session": "xxx",
		"value":   "3",
		"status":  []any{"done"},
	}
	var buf bytes.Buffer
	if err := Encode(&buf, msg); err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	m := decoded.(map[string]any)
	status := m["status"].([]any)
	if string(status[0].([]byte)) != "done" {
		t.Errorf("status = %v", status)
	}
}

func TestMultipleMessagesOnStream(t *testing.T) {
	// nREPL sends consecutive bencoded dicts on one stream
	var buf bytes.Buffer
	Encode(&buf, map[string]any{"op": "clone", "id": "1"})
	Encode(&buf, map[string]any{"op": "eval", "id": "2", "code": "(+ 1 2)"})

	r := bytes.NewReader(buf.Bytes())
	v1, err := Decode(r)
	if err != nil {
		t.Fatal(err)
	}
	m1 := v1.(map[string]any)
	if string(m1["op"].([]byte)) != "clone" {
		t.Errorf("msg1 op = %v", m1["op"])
	}

	v2, err := Decode(r)
	if err != nil {
		t.Fatal(err)
	}
	m2 := v2.(map[string]any)
	if string(m2["op"].([]byte)) != "eval" {
		t.Errorf("msg2 op = %v", m2["op"])
	}
}
