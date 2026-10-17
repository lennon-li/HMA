package host

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func obsidianScript(t *testing.T, rel string) string {
	t.Helper()
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, "obsidian", "AgentSystem", rel)
	if _, err := os.Stat(p); err != nil {
		t.Skipf("%s not present", rel)
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not present")
	}
	return p
}

// PythonCanonicalSHA256 must equal verify-loop's
// sha256_text(json.dumps(env, sort_keys=True, separators=(",", ":"))).
func TestGateCR3CanonicalHashMatchesPython(t *testing.T) {
	obsidianScript(t, "scripts/verify-loop")
	for _, doc := range []string{
		`{"b":1,"a":[true,null,1.5,"x"],"c":{"z":"é","y":"\u0001\n\t\"\\"}}`,
		`{"goal":"fix 𝄞 ✓","max_attempts":3,"spend":0.25,"del":"\u007f","route":{"worker":"jax","model":null}}`,
		`[]`, `{"n":-0.0,"e":100000.0,"big":12345678901234567890,"f":0.1}`,
	} {
		got, err := PythonCanonicalSHA256([]byte(doc))
		if err != nil {
			t.Fatal(err)
		}
		out, err := exec.Command("python3", "-I", "-c", `import json,hashlib,sys
v=json.loads(sys.argv[1]); print(hashlib.sha256(json.dumps(v,sort_keys=True,separators=(",",":")).encode("utf-8")).hexdigest())`, doc).Output()
		if err != nil {
			t.Fatal(err)
		}
		if want := strings.TrimSpace(string(out)); got != want {
			t.Errorf("%s: go %s python %s", doc, got, want)
		}
	}
}

// ValidateAuditReport must accept and reject exactly what validate_report.py
// does.
func TestGateCR3ReportContractParity(t *testing.T) {
	script := obsidianScript(t, "agent-memory/skills/independent-audit/scripts/validate_report.py")
	finding := func(edit func(map[string]any)) map[string]any {
		f := map[string]any{"id": "F1", "title": "t", "diagnosis": "d", "kind": "defect", "mode": "code", "severity": "medium", "confidence": "observed",
			"evidence": []any{map[string]any{"source": "s", "observation": "o"}},
			"fix":      map[string]any{"action": "a", "why": "w", "tradeoffs": "t", "alternative": "x", "effort": "small", "status": "proposed", "verification": []any{map[string]any{"method": "m", "expected": "e"}}}}
		if edit != nil {
			edit(f)
		}
		return f
	}
	base := func() map[string]any {
		return map[string]any{"audit_id": "a", "target": "t", "modes": []any{"code"}, "verdict": "pass", "isolation": "isolated", "phase1_frozen": true,
			"limitations": []any{}, "requirements": []any{map[string]any{"id": "R1", "status": "pass", "evidence_refs": []any{"e1"}}}, "findings": []any{}}
	}
	cases := map[string]func(map[string]any){
		"valid-pass":           nil,
		"fix-required-finding": func(r map[string]any) { r["verdict"] = "fix-required"; r["findings"] = []any{finding(nil)} },
		"pass-with-finding":    func(r map[string]any) { r["findings"] = []any{finding(nil)} },
		"pass-opportunity": func(r map[string]any) {
			r["findings"] = []any{finding(func(f map[string]any) { f["kind"] = "opportunity"; delete(f, "fix") })}
		},
		"empty-requirements": func(r map[string]any) { r["requirements"] = []any{} },
		"unfrozen":           func(r map[string]any) { r["phase1_frozen"] = false },
		"no-target":          func(r map[string]any) { delete(r, "target") },
		"no-evidence-refs": func(r map[string]any) {
			r["requirements"] = []any{map[string]any{"id": "R1", "status": "pass", "evidence_refs": []any{}}}
		},
		"bad-severity": func(r map[string]any) {
			r["verdict"] = "fix-required"
			r["findings"] = []any{finding(func(f map[string]any) { f["severity"] = "urgent" })}
		},
		"mode-not-audited": func(r map[string]any) {
			r["verdict"] = "fix-required"
			r["findings"] = []any{finding(func(f map[string]any) { f["mode"] = "plan" })}
		},
		"high-conditional": func(r map[string]any) {
			r["verdict"] = "conditional-pass"
			r["findings"] = []any{finding(func(f map[string]any) { f["severity"] = "high" })}
		},
		"partial-pass": func(r map[string]any) { r["isolation"] = "partial" },
		"unknown-requirement": func(r map[string]any) {
			r["requirements"] = []any{map[string]any{"id": "R1", "status": "unknown", "evidence_refs": []any{}}}
		},
		"limitations-not-array": func(r map[string]any) { r["limitations"] = "none" },
		"no-fix-verification": func(r map[string]any) {
			r["verdict"] = "fix-required"
			r["findings"] = []any{finding(func(f map[string]any) { f["fix"].(map[string]any)["verification"] = []any{} })}
		},
		"duplicate-id": func(r map[string]any) {
			r["verdict"] = "fix-required"
			r["findings"] = []any{finding(nil), finding(nil)}
		},
		"null-findings": func(r map[string]any) { r["findings"] = nil },
		"bad-mode":      func(r map[string]any) { r["modes"] = []any{"code", "vibes"} },
	}
	for name, edit := range cases {
		r := base()
		if edit != nil {
			edit(r)
		}
		b, _ := json.Marshal(r)
		p := filepath.Join(t.TempDir(), "r.json")
		os.WriteFile(p, b, 0o600)
		pyValid := exec.Command("python3", "-I", script, p).Run() == nil
		goValid := len(ValidateAuditReport(b)) == 0
		if pyValid != goValid {
			t.Errorf("%s: python valid=%v go valid=%v (%v)", name, pyValid, goValid, ValidateAuditReport(b))
		}
	}
}

// Liz final3 C-F17: the nonempty check strips exactly what Python's
// str.strip strips, over every code point.
func TestGateCR6StripMatchesPython(t *testing.T) {
	obsidianScript(t, "agent-memory/skills/independent-audit/scripts/validate_report.py")
	out, err := exec.Command("python3", "-I", "-c", `print(",".join(str(c) for c in range(0x110000) if chr(c).isspace()))`).Output()
	if err != nil {
		t.Fatal(err)
	}
	want := map[rune]bool{}
	for _, f := range strings.Split(strings.TrimSpace(string(out)), ",") {
		var c rune
		if _, err := fmt.Sscan(f, &c); err != nil {
			t.Fatal(err)
		}
		want[c] = true
	}
	for c := rune(0); c < 0x110000; c++ {
		if pyIsSpace(c) != want[c] {
			t.Errorf("U+%04X: go %v python %v", c, pyIsSpace(c), want[c])
		}
	}
}
