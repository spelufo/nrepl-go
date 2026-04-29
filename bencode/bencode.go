// Package bencode implements encoding and decoding of the bencode format
// used by the nREPL protocol.
package bencode

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
)

// Encode writes a single bencoded value to w.
// Supported types: string, []byte, int/int64, []any, map[string]any.
func Encode(w io.Writer, v any) error {
	switch val := v.(type) {
	case string:
		return encodeBytes(w, []byte(val))
	case []byte:
		return encodeBytes(w, val)
	case int:
		return encodeInt(w, int64(val))
	case int64:
		return encodeInt(w, val)
	case []any:
		return encodeList(w, val)
	case map[string]any:
		return encodeDict(w, val)
	default:
		return fmt.Errorf("bencode: unsupported type %T", v)
	}
}

func encodeBytes(w io.Writer, b []byte) error {
	header := strconv.Itoa(len(b)) + ":"
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}

func encodeInt(w io.Writer, n int64) error {
	_, err := fmt.Fprintf(w, "i%de", n)
	return err
}

func encodeList(w io.Writer, items []any) error {
	if _, err := io.WriteString(w, "l"); err != nil {
		return err
	}
	for _, item := range items {
		if err := Encode(w, item); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "e")
	return err
}

func encodeDict(w io.Writer, m map[string]any) error {
	if _, err := io.WriteString(w, "d"); err != nil {
		return err
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := encodeBytes(w, []byte(k)); err != nil {
			return err
		}
		if err := Encode(w, m[k]); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "e")
	return err
}

// Decode reads a single bencoded value from r.
// Returns string (as []byte), int64, []any, or map[string]any.
func Decode(r io.Reader) (any, error) {
	return decodeFrom(newByteReader(r))
}

func decodeFrom(r *byteReader) (any, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch {
	case b == 'i':
		return decodeInt(r)
	case b == 'l':
		return decodeList(r)
	case b == 'd':
		return decodeDict(r)
	case b >= '0' && b <= '9':
		return decodeString(r, b)
	default:
		return nil, fmt.Errorf("bencode: unexpected byte %q", b)
	}
}

func decodeInt(r *byteReader) (int64, error) {
	buf := make([]byte, 0, 16)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		if b == 'e' {
			break
		}
		buf = append(buf, b)
	}
	if len(buf) == 0 {
		return 0, errors.New("bencode: empty integer")
	}
	return strconv.ParseInt(string(buf), 10, 64)
}

func decodeString(r *byteReader, first byte) ([]byte, error) {
	// Read length
	buf := []byte{first}
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == ':' {
			break
		}
		buf = append(buf, b)
	}
	length, err := strconv.Atoi(string(buf))
	if err != nil {
		return nil, fmt.Errorf("bencode: invalid string length: %w", err)
	}
	data := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func decodeList(r *byteReader) ([]any, error) {
	var items []any
	for {
		b, err := r.PeekByte()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			r.ReadByte()
			return items, nil
		}
		v, err := decodeFrom(r)
		if err != nil {
			return nil, err
		}
		items = append(items, v)
	}
}

func decodeDict(r *byteReader) (map[string]any, error) {
	m := make(map[string]any)
	for {
		b, err := r.PeekByte()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			r.ReadByte()
			return m, nil
		}
		keyRaw, err := decodeFrom(r)
		if err != nil {
			return nil, err
		}
		key, ok := keyRaw.([]byte)
		if !ok {
			return nil, errors.New("bencode: dict key must be a string")
		}
		val, err := decodeFrom(r)
		if err != nil {
			return nil, err
		}
		m[string(key)] = val
	}
}

// byteReader wraps an io.Reader to provide single-byte reading with peek.
type byteReader struct {
	r      io.Reader
	buf    [1]byte
	peeked bool
}

func newByteReader(r io.Reader) *byteReader {
	return &byteReader{r: r}
}

func (br *byteReader) ReadByte() (byte, error) {
	if br.peeked {
		br.peeked = false
		return br.buf[0], nil
	}
	_, err := io.ReadFull(br.r, br.buf[:])
	return br.buf[0], err
}

func (br *byteReader) PeekByte() (byte, error) {
	if br.peeked {
		return br.buf[0], nil
	}
	_, err := io.ReadFull(br.r, br.buf[:])
	if err != nil {
		return 0, err
	}
	br.peeked = true
	return br.buf[0], nil
}

func (br *byteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if br.peeked {
		p[0] = br.buf[0]
		br.peeked = false
		if len(p) == 1 {
			return 1, nil
		}
		n, err := br.r.Read(p[1:])
		return n + 1, err
	}
	return br.r.Read(p)
}
