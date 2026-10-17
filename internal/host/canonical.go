package host

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"unicode/utf16"
)

// PythonCanonicalSHA256 is the sha256 hex of raw JSON re-serialized as
// Python's json.dumps(v, sort_keys=True, separators=(",", ":")) does with its
// default ensure_ascii: sorted keys, no spaces, everything outside printable
// ASCII (space to tilde) as \uXXXX (UTF-16). Number text is kept as written,
// which equals Python's output when the document was written by Python (as a
// verify-loop state file is). verify-loop computes envelope_sha256 so.
func PythonCanonicalSHA256(raw []byte) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := writePy(&b, v); err != nil {
		return "", err
	}
	sum := sha256.Sum256(b.Bytes())
	return hex.EncodeToString(sum[:]), nil
}

func writePy(b *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case json.Number:
		b.WriteString(x.String())
	case string:
		writePyString(b, x)
	case []any:
		b.WriteByte('[')
		for i, y := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := writePy(b, y); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		// Python sorts str keys by code point; Go sorts strings by UTF-8
		// bytes, which orders code points the same way.
		sort.Strings(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			writePyString(b, k)
			b.WriteByte(':')
			if err := writePy(b, x[k]); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	default:
		return fmt.Errorf("unexpected JSON value %T", v)
	}
	return nil
}

func writePyString(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			switch {
			case r < 0x20 || r >= 0x7f && r < 0x10000:
				fmt.Fprintf(b, `\u%04x`, r)
			case r >= 0x10000:
				h, l := utf16.EncodeRune(r)
				fmt.Fprintf(b, `\u%04x\u%04x`, h, l)
			default:
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}
