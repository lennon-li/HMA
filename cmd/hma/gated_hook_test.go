package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	hoststate "github.com/lennon-li/HMA/internal/host"
)

// hookCall runs verify-loop's own invoke_authority_hook (the D5 controller
// code path) with the real hma binary as the hook and returns its
// (response, error) pair.
func hookCall(t *testing.T, bin string, f *attemptFixture, req map[string]any) (map[string]any, string) {
	t.Helper()
	home, _ := os.UserHomeDir()
	vl := filepath.Join(home, "obsidian", "AgentSystem", "scripts", "verify-loop")
	if _, err := os.Stat(vl); err != nil {
		t.Skip("verify-loop not present")
	}
	f.syncPending(req)
	argv := []string{bin, "host", "attempt", "--store", f.store, "--plan", f.plan, "--contract", f.contract, "--policy", f.policy}
	in, _ := json.Marshal(map[string]any{"env": map[string]any{"authority_hook": map[string]any{"argv": argv}}, "request": req, "root": f.root})
	script := `import json, sys, importlib.machinery, importlib.util
loader = importlib.machinery.SourceFileLoader("verify_loop", sys.argv[1])
spec = importlib.util.spec_from_loader("verify_loop", loader)
vl = importlib.util.module_from_spec(spec); loader.exec_module(vl)
a = json.load(sys.stdin)
resp, err = vl.invoke_authority_hook(a["env"], a["request"], a["root"])
print(json.dumps({"response": resp, "error": err}))`
	cmd := exec.Command("python3", "-I", "-c", script, vl)
	cmd.Stdin = bytes.NewReader(in)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("verify-loop hook call: %v", err)
	}
	var got struct {
		Response map[string]any `json:"response"`
		Error    *string        `json:"error"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if got.Error != nil {
		return got.Response, *got.Error
	}
	return got.Response, ""
}

func buildHMA(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "hma")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build hma: %v %s", err, out)
	}
	return bin
}

// test_hma_bound_launch_requires_eligible_true (Gate C §0.1 row 5): the
// controller accepts the real hook's grant, and refuses a replay of a used
// slot because launch_eligible is false.
func TestD5HMABoundLaunchRequiresEligibleTrue(t *testing.T) {
	bin := buildHMA(t)
	f := newAttemptFixture(t, 2)
	req := f.request(f.loopDir, 1, "d-1", nil)
	resp, errText := hookCall(t, bin, f, req)
	if errText != "" || resp["launch_eligible"] != true || resp["reservation_id"] == "" {
		t.Fatalf("first launch refused: %v %s", resp, errText)
	}
	os.MkdirAll(filepath.Join(f.loopDir, "dispatch-ledger", "runs", "d-1"), 0o700)
	if _, errText := hookCall(t, bin, f, req); errText != "authority hook did not set launch_eligible to true" {
		t.Fatalf("used slot: %q", errText)
	}
	// A refused attempt (wrong route) makes the hook exit non-zero.
	bad := f.request(f.loopDir, 2, "d-2", nil)
	bad["route"] = map[string]any{"worker": "liz", "model": "x", "effort": "high", "cost": map[string]any{"spend_cap_usd": 1, "per_attempt_usd": 1}}
	if _, errText := hookCall(t, bin, f, bad); errText == "" {
		t.Fatal("wrong route granted")
	}
}

// test_resume_replay_uses_same_dispatch_id: a resume before launch repeats
// the identical request and gets the same reservation, charged once.
func TestD5ResumeReplayUsesSameDispatchID(t *testing.T) {
	bin := buildHMA(t)
	f := newAttemptFixture(t, 2)
	req := f.request(f.loopDir, 1, "d-1", nil)
	first, e1 := hookCall(t, bin, f, req)
	again, e2 := hookCall(t, bin, f, req)
	if e1 != "" || e2 != "" || first["reservation_id"] != again["reservation_id"] || again["dispatch_run_id"] != "d-1" || again["replay"] != true {
		t.Fatalf("first %v (%s) again %v (%s)", first, e1, again, e2)
	}
	if f.count(hoststate.KindReservation) != 1 {
		t.Fatal("resume charged twice")
	}
}
