package host

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

// UnmarshalExact decodes one JSON document into v reading object keys the way
// Python's json module does: exactly. encoding/json matches struct field tags
// case-insensitively (with Unicode folding), so a report carrying both
// "verdict" and "Verdict" would be read by Go as the last one while the
// Python report validator reads only "verdict". Every struct tag HMA decodes
// Python-written documents into is lowercase ASCII, so dropping every key that
// is not made of [a-z0-9_] before decoding leaves Go reading exactly the keys
// Python reads. The document must be exactly one JSON value, in valid UTF-8:
// the Python tools read with encoding="utf-8", which refuses invalid bytes
// that encoding/json would silently replace with U+FFFD.
func UnmarshalExact(raw []byte, v any) error {
	if !utf8.Valid(raw) {
		return errors.New("JSON document is not valid UTF-8")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var doc any
	if err := dec.Decode(&doc); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return errors.New("trailing data after JSON document")
	}
	clean, err := json.Marshal(exactKeys(doc))
	if err != nil {
		return err
	}
	return json.Unmarshal(clean, v)
}

func exactKeys(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			if plainKey(k) {
				out[k] = exactKeys(e)
			}
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = exactKeys(e)
		}
		return out
	}
	return v
}

func plainKey(k string) bool {
	for i := 0; i < len(k); i++ {
		c := k[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}
