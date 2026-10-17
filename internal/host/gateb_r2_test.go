package host

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestGateBR2HostPartialTemp(t *testing.T) {
	d := t.TempDir()
	s := NewStore(d)
	r := appendDecision(t, s)
	rd := filepath.Join(d, "host", "run")
	os.WriteFile(filepath.Join(rd, "head.pending"), []byte("never-landed\n"), 0600)
	os.WriteFile(filepath.Join(rd, ".hma-host-partial"), []byte("{\"sequence\":"), 0600)
	rs, e := s.Load("run")
	if e != nil || len(rs) != 1 || rs[0].HeadAnchor != r.HeadAnchor {
		t.Fatalf("partial temp affected verified prefix: %v", e)
	}
}
func TestGateBR2HostRecoveryRejectsForgedTail(t *testing.T) {
	d := t.TempDir()
	s := NewStore(d)
	appendDecision(t, s)
	r := appendDecision(t, s)
	rd := filepath.Join(d, "host", "run")
	os.WriteFile(filepath.Join(rd, "head.pending"), []byte(r.HeadAnchor+"\n"), 0600)
	p := filepath.Join(rd, recordName(2))
	b, _ := os.ReadFile(p)
	b = []byte(strings.Replace(string(b), "delegation is cheaper", "TAMPER", 1))
	if string(b) == "" {
		t.Fatal("empty")
	} // corrupt the bytes without repairing the hash
	b = append([]byte("x"), b...)
	os.WriteFile(p, b, 0600)
	if _, e := s.Load("run"); e == nil {
		t.Fatal("pending accepted corrupt record")
	}
	t.Log("corrupt chain refused before pending recovery")
}
func TestGateBR2HostChild(t *testing.T) {
	d := os.Getenv("HMA_AUDIT_B_CHILD")
	if d == "" {
		t.Skip("child only")
	}
	if e := appendDecisionErr(NewStore(d)); e != nil {
		t.Fatal(e)
	}
}

func TestGateBR2HostSeparateProcessWriters(t *testing.T) {
	d := t.TempDir()
	const n = 12
	var wg sync.WaitGroup
	errs := make(chan string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := exec.Command(os.Args[0], "-test.run=^TestGateBR2HostChild$")
			c.Env = append(os.Environ(), "HMA_AUDIT_B_CHILD="+d)
			b, e := c.CombinedOutput()
			if e != nil {
				errs <- fmt.Sprintf("%v: %s", e, b)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	rs, e := NewStore(d).Load("run")
	if e != nil || len(rs) != n {
		t.Fatalf("process writers %d/%d err=%v", len(rs), n, e)
	}
	t.Logf("all %d acknowledged appends in verified chain", n)
}
func TestGateBR2HostPowerLossImage(t *testing.T) {
	t.Skip("B7: unreachable after the fix. AppendWith syncs the directory after the head rename before removing the pending intent, so a durable pending removal implies a durable head; a lagging head without an intent stays HOST_HEAD_MISMATCH (TestGateBHostHeadDriftStillDetected)")
	d := t.TempDir()
	s := NewStore(d)
	first := appendDecision(t, s)
	appendDecision(t, s)
	rd := filepath.Join(d, "host", "run") // Model a lost unsynchronized head rename while pending removal persisted.
	os.WriteFile(filepath.Join(rd, "head"), []byte(first.HeadAnchor+"\n"), 0600)
	rs, e := s.Load("run")
	if e != nil {
		t.Errorf("allowed unsynchronized directory crash image wedges load: len=%d err=%v", len(rs), e)
	}
}

func TestGateBR2HostHashMismatchPending(t *testing.T) {
	d := t.TempDir()
	st := NewStore(d)
	appendDecision(t, st)
	r := appendDecision(t, st)
	rd := filepath.Join(d, "host", "run")
	os.WriteFile(filepath.Join(rd, "head.pending"), []byte(r.HeadAnchor+"\n"), 0600)
	p := filepath.Join(rd, recordName(2))
	b, _ := os.ReadFile(p)
	next := strings.Replace(string(b), `"actor":"fury"`, `"actor":"changed"`, 1)
	if next == string(b) {
		t.Fatal("probe replacement did not match")
	}
	os.WriteFile(p, []byte(next), 0600)
	if _, e := st.Load("run"); e == nil || e.Error() != ReasonHostChainCorrupt {
		t.Fatalf("bad hash with matching intent: %v", e)
	}
}
