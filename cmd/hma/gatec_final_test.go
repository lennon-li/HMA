package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
	"github.com/lennon-li/HMA/internal/store"
)

// C-F14: hma host dispatch never accepts a caller-supplied packet binding.
func TestGateCFinalDispatchRefusesCallerPacket(t *testing.T) {
	f := newReviewFixture(t, hoststate.IssuePerChange, false)
	h, _ := hoststate.NewStore(f.store).Load("run")
	d := *h[len(h)-1].Dispatch
	p := *d.Packet
	p.ReviewDispatchID, p.ReviewNonce, p.DispatchRunID = "forged", "00112233445566778899aabbccddeeff", "forged-run"
	d.Packet = &p
	input := filepath.Join(t.TempDir(), "d.json")
	writeJSON(t, input, hostDispatchInput{hostEnvelope: fxEnv("2026-09-16T12:04:30Z"), Dispatch: d})
	before := f.records()
	if err := run([]string{"host", "dispatch", "--input", input, "--store", f.store}); err == nil || err.Error() != hoststate.ReasonReviewImportRequired {
		t.Fatalf("caller packet: %v", err)
	}
	if f.records() != before {
		t.Fatal("refused dispatch appended")
	}
}

// C-F15: the loop recapture keeps its output, and the stored streams
// reproduce the evidence output digest.
func TestGateCFinalRecaptureRetainsOutput(t *testing.T) {
	d := t.TempDir()
	root, base := seedLoopRun(t, d)
	if err := runLoopEvidence(t, d, root, base, writeLoopState(t, d, "run", "U1", []string{"python3", "-c", "import sys; print('out'); print('err', file=sys.stderr); sys.exit(1)"}, "green", 0)); err != nil {
		t.Fatal(err)
	}
	core, _ := store.New(d).Load("run")
	ev := core[len(core)-1].Evidence
	if ev.OutputRef == "" {
		t.Fatal("no output ref")
	}
	b, err := os.ReadFile(filepath.Join(d, ev.OutputRef))
	if err != nil {
		t.Fatal(err)
	}
	var art struct{ Stdout, Stderr []byte }
	if err := json.Unmarshal(b, &art); err != nil {
		t.Fatal(err)
	}
	if string(art.Stdout) != "out\n" || string(art.Stderr) != "err\n" {
		t.Fatalf("streams %q %q", art.Stdout, art.Stderr)
	}
	h := sha256.New()
	for _, s := range []struct {
		name string
		data []byte
	}{{"stdout", art.Stdout}, {"stderr", art.Stderr}} {
		h.Write([]byte("hma-evidence-v1\x00" + s.name + "\x00"))
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(s.data)))
		h.Write(n[:])
		h.Write(s.data)
	}
	if hex.EncodeToString(h.Sum(nil)) != ev.OutputDigest {
		t.Fatal("stored streams do not reproduce the output digest")
	}
}
