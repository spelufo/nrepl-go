package nrepl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResponseAccessors(t *testing.T) {
	r := Response{Msg: Message{
		"id":      []byte("1"),
		"session": []byte("abc-123"),
		"status":  []any{[]byte("done")},
		"value":   []byte("3"),
		"out":     []byte("hello\n"),
	}}

	if r.ID() != "1" {
		t.Errorf("ID() = %q, want %q", r.ID(), "1")
	}
	if r.Session() != "abc-123" {
		t.Errorf("Session() = %q", r.Session())
	}
	if !r.HasStatus("done") {
		t.Error("expected done status")
	}
	if r.HasStatus("error") {
		t.Error("unexpected error status")
	}
	if v, ok := r.Value(); !ok || v != "3" {
		t.Errorf("Value() = %q, %v", v, ok)
	}
	if v, ok := r.Out(); !ok || v != "hello\n" {
		t.Errorf("Out() = %q, %v", v, ok)
	}
	if _, ok := r.Err(); ok {
		t.Error("Err() should be absent")
	}
}

func TestResponseStringValues(t *testing.T) {
	r := Response{Msg: Message{
		"id":    "42",
		"value": "ok",
	}}
	if r.ID() != "42" {
		t.Errorf("ID() = %q", r.ID())
	}
	if v, _ := r.Value(); v != "ok" {
		t.Errorf("Value() = %q", v)
	}
}

func TestFindPortFileNotFound(t *testing.T) {
	_, err := FindPortFile(t.TempDir())
	if err == nil {
		t.Error("expected error when no .nrepl-port exists")
	}
}

func TestReadPortFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".nrepl-port"), []byte("12345\n"), 0644); err != nil {
		t.Fatal(err)
	}
	addr, err := ReadPortFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if addr != "localhost:12345" {
		t.Errorf("addr = %q", addr)
	}
}

func TestFindPortFileWalksUp(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".nrepl-port"), []byte("9999"), 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	addr, err := FindPortFile(sub)
	if err != nil {
		t.Fatal(err)
	}
	if addr != "localhost:9999" {
		t.Errorf("addr = %q", addr)
	}
}
