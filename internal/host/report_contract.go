package host

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// pyIsSpace is Python's str.isspace for one code point, which str.strip
// uses: unicode.IsSpace plus the information separators U+001C..U+001F.
func pyIsSpace(r rune) bool { return unicode.IsSpace(r) || r >= 0x1c && r <= 0x1f }

// ValidateAuditReport applies the independent-audit report contract
// (skills/independent-audit/scripts/validate_report.py) to raw report bytes
// and returns every violation. It checks form and verdict consistency only,
// not the truth of findings. It must stay equal to that validator; a parity
// test runs both.
func ValidateAuditReport(raw []byte) []string {
	var errs []string
	e := func(format string, a ...any) { errs = append(errs, fmt.Sprintf(format, a...)) }
	var report map[string]any
	if !utf8.Valid(raw) {
		return []string{"report must be UTF-8"}
	}
	if json.Unmarshal(raw, &report) != nil || report == nil {
		return []string{"report must be a JSON object"}
	}
	str := func(v any) (string, bool) { s, ok := v.(string); return s, ok }
	nonempty := func(v any) bool { s, ok := str(v); return ok && strings.TrimFunc(s, pyIsSpace) != "" }
	oneOf := func(v any, set ...string) bool {
		s, ok := str(v)
		if !ok {
			return false
		}
		for _, x := range set {
			if s == x {
				return true
			}
		}
		return false
	}
	modesSet := []string{"plan", "code", "reasoning", "outcome"}
	for _, k := range []string{"audit_id", "target"} {
		if !nonempty(report[k]) {
			e("%s: required nonempty string", k)
		}
	}
	var modes []string
	if m, ok := report["modes"].([]any); ok && len(m) > 0 {
		for _, x := range m {
			if !oneOf(x, modesSet...) {
				modes = nil
				e("modes: nonempty array of valid audit modes required")
				break
			}
			modes = append(modes, x.(string))
		}
	} else {
		e("modes: nonempty array of valid audit modes required")
	}
	if !oneOf(report["verdict"], "pass", "conditional-pass", "fix-required", "blocked", "not-evaluated") {
		e("verdict: invalid value")
	}
	if !oneOf(report["isolation"], "isolated", "partial", "compromised") {
		e("isolation: invalid value")
	}
	if _, ok := report["phase1_frozen"].(bool); !ok {
		e("phase1_frozen: boolean required")
	}
	if lim, ok := report["limitations"].([]any); !ok {
		e("limitations: required array of nonempty strings")
	} else {
		for _, x := range lim {
			if !nonempty(x) {
				e("limitations: required array of nonempty strings")
				break
			}
		}
	}
	requirements, ok := report["requirements"].([]any)
	if !ok {
		e("requirements: required array")
		requirements = nil
	}
	for i, x := range requirements {
		r, ok := x.(map[string]any)
		if !ok || !nonempty(r["id"]) || !oneOf(r["status"], "pass", "fail", "unknown") {
			e("requirements[%d]: requires id and status pass/fail/unknown", i)
			continue
		}
		refs, ok := r["evidence_refs"].([]any)
		if !ok {
			e("requirements[%d].evidence_refs: array of nonempty strings required", i)
		} else {
			for _, ref := range refs {
				if !nonempty(ref) {
					e("requirements[%d].evidence_refs: array of nonempty strings required", i)
					break
				}
			}
		}
		if oneOf(r["status"], "pass", "fail") && len(refs) == 0 {
			e("requirements[%d]: observed pass/fail must reference evidence", i)
		}
	}
	findings, ok := report["findings"].([]any)
	if !ok {
		e("findings: required array")
		findings = nil
	}
	seen := map[string]bool{}
	actionable, blocking := 0, 0
	for i, x := range findings {
		f, ok := x.(map[string]any)
		if !ok {
			e("findings[%d]: object required", i)
			continue
		}
		for _, k := range []string{"id", "title", "diagnosis"} {
			if !nonempty(f[k]) {
				e("findings[%d].%s: required nonempty string", i, k)
			}
		}
		if id, ok := str(f["id"]); ok {
			if seen[id] {
				e("findings[%d].id: duplicate ID", i)
			}
			seen[id] = true
		}
		for _, kv := range []struct {
			k   string
			set []string
		}{
			{"kind", []string{"defect", "risk", "gap", "opportunity"}},
			{"mode", modesSet},
			{"severity", []string{"critical", "high", "medium", "low"}},
			{"confidence", []string{"observed", "inferred", "hypothesis"}},
		} {
			if !oneOf(f[kv.k], kv.set...) {
				e("findings[%d].%s: invalid value", i, kv.k)
			}
		}
		if !oneOf(f["mode"], modes...) {
			e("findings[%d].mode: not among audited modes", i)
		}
		ev, ok := f["evidence"].([]any)
		if !ok {
			e("findings[%d].evidence: required array", i)
			ev = nil
		}
		if s, _ := str(f["confidence"]); s == "observed" && len(ev) == 0 {
			e("findings[%d]: observed finding requires evidence", i)
		}
		for j, y := range ev {
			m, ok := y.(map[string]any)
			if !ok || !nonempty(m["source"]) || !nonempty(m["observation"]) {
				e("findings[%d].evidence[%d]: source and observation required", i, j)
			}
		}
		if oneOf(f["kind"], "defect", "risk", "gap") {
			actionable++
			if oneOf(f["severity"], "critical", "high") {
				blocking++
			}
		}
		fixRaw, present := f["fix"]
		if s, _ := str(f["kind"]); s == "opportunity" && (!present || fixRaw == nil) {
			continue
		}
		fix, ok := fixRaw.(map[string]any)
		if !ok {
			e("findings[%d].fix: required for actionable finding", i)
			continue
		}
		for _, k := range []string{"action", "why", "tradeoffs", "alternative"} {
			if !nonempty(fix[k]) {
				e("findings[%d].fix.%s: required nonempty string", i, k)
			}
		}
		if !oneOf(fix["effort"], "small", "medium", "large", "unknown") {
			e("findings[%d].fix.effort: invalid value", i)
		}
		if !oneOf(fix["status"], "proposed", "validated", "remediation_blocked") {
			e("findings[%d].fix.status: invalid value", i)
		}
		checks, ok := fix["verification"].([]any)
		if !ok || len(checks) == 0 {
			e("findings[%d].fix.verification: nonempty array required", i)
		} else {
			for j, c := range checks {
				m, ok := c.(map[string]any)
				if !ok || !nonempty(m["method"]) || !nonempty(m["expected"]) {
					e("findings[%d].fix.verification[%d]: method and expected required", i, j)
				}
			}
		}
	}
	verdict := report["verdict"]
	if oneOf(verdict, "pass", "conditional-pass") {
		if len(requirements) == 0 {
			e("cannot pass without evaluated requirements")
		}
		for _, x := range requirements {
			if r, ok := x.(map[string]any); ok && oneOf(r["status"], "fail", "unknown") {
				e("cannot pass with failed or unknown mandatory requirements")
				break
			}
		}
		if frozen, _ := report["phase1_frozen"].(bool); !frozen {
			e("cannot pass without frozen Phase 1 assessment")
		}
		if blocking > 0 {
			e("cannot pass with high/critical actionable findings")
		}
	}
	if s, _ := str(verdict); s == "pass" {
		if s, _ := str(report["isolation"]); s != "isolated" {
			e("pass requires isolated auditor")
		}
		if actionable > 0 {
			e("pass with unresolved findings; use conditional-pass")
		}
	}
	return errs
}
