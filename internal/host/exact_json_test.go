package host

import "testing"

func TestUnmarshalExactReadsOnlyExactKeys(t *testing.T) {
	var v struct {
		Verdict   string `json:"verdict"`
		Isolation string `json:"isolation"`
		Nested    struct {
			ID string `json:"id"`
		} `json:"nested"`
	}
	raw := `{"verdict":"fix-required","Verdict":"pass","isolation":"partial","iſolation":"isolated","nested":{"id":"a","ID":"b"}}`
	if err := UnmarshalExact([]byte(raw), &v); err != nil {
		t.Fatal(err)
	}
	if v.Verdict != "fix-required" || v.Isolation != "partial" || v.Nested.ID != "a" {
		t.Fatalf("got %+v", v)
	}
	if err := UnmarshalExact([]byte(`{"verdict":"pass"} {}`), &v); err == nil {
		t.Fatal("two documents accepted")
	}
}

// Liz final5: invalid UTF-8 is refused, as Python's utf-8 read refuses it.
func TestInvalidUTF8Refused(t *testing.T) {
	raw := []byte("{\"verdict\":\"pass\",\"target\":\"\xff\"}")
	var v struct {
		Verdict string `json:"verdict"`
	}
	if UnmarshalExact(raw, &v) == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
	if len(ValidateAuditReport(raw)) == 0 {
		t.Fatal("validator accepted invalid UTF-8")
	}
}
