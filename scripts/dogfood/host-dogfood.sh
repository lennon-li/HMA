#!/usr/bin/env bash
# Synthetic dogfood of the HMA host-orchestration approval-binding workflow
# (docs/task14-host-orchestration.md).
#
# Runs the real `hma host ...` and `hma transition` CLIs against a throwaway
# store. No human approval is fabricated:
#   - the only approval-shaped record is a fixture labelled
#     "SYNTHETIC-DOGFOOD-FIXTURE (not a human approval)", written only into a
#     scratch store that hostinput marked as synthetic;
#   - stage-transition inputs carry no approval, so after the host gate is
#     evaluated the core transition path must refuse to record anything.
# No worker is started and no `hi` message is sent; preflight results are
# synthetic telemetry.
#
# Usage: scripts/dogfood/host-dogfood.sh [--keep]
#   --keep  keep the scratch directory for inspection
set -euo pipefail

keep=0
case "${1:-}" in
  "") ;;
  --keep) keep=1 ;;
  *) echo "usage: $0 [--keep]" >&2; exit 2 ;;
esac

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
if ! command -v go >/dev/null 2>&1 && [ -x "$HOME/.local/go/bin/go" ]; then
  PATH="$HOME/.local/go/bin:$PATH"
fi
command -v go >/dev/null 2>&1 || { echo "go toolchain not found" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "jq not found" >&2; exit 1; }

work="$(mktemp -d "${TMPDIR:-/tmp}/hma-host-dogfood.XXXXXX")"
cleanup() {
  if [ "$keep" -eq 1 ]; then echo "kept: $work"; else rm -rf "$work"; fi
}
trap cleanup EXIT

bin="$work/bin"; store="$work/store"; in="$work/input.json"
mkdir -p "$bin" "$store"
run_id="synthetic-dogfood-run"; unit="SYNTHETIC-U1"
approved="synthetic-route-approval-digest"; forged="forged-route-approval-digest"

(cd "$repo" && go build -o "$bin/hma" ./cmd/hma && go build -o "$bin/hostinput" ./scripts/dogfood/hostinput)
hma="$bin/hma"; hostinput="$bin/hostinput"

pass=0
ok() { pass=$((pass + 1)); echo "ok   $*"; }
die() { echo "FAIL $*" >&2; exit 1; }

host_count() { "$hma" host show --store "$store" --run_id "$run_id" | jq 'length'; }
core_lines() { wc -l <"$store/$run_id.jsonl"; }
# store_fingerprint digests every file name and byte in the store (core stream,
# core head, host records, host head, marker). Lock files are excluded: taking
# a lock may create an empty one without changing any recorded state.
store_fingerprint() {
  (cd "$store" && find . \( -type f -o -type l \) ! -name '*.lock' -print0 | LC_ALL=C sort -z |
    xargs -0 sha256sum) | sha256sum | cut -d' ' -f1
}

emit() { "$hostinput" emit "$1" --store "$store" --run_id "$run_id" --unit_id "$unit" --digest "$2" --out "$in"; }

# expect_ok <label> <hma args...>: command must succeed and append one host record.
expect_ok() {
  local label="$1"; shift
  local before after
  before="$(host_count)"
  "$hma" "$@" --input "$in" --store "$store" >"$work/out.json" || die "$label: unexpected refusal"
  after="$(host_count)"
  [ "$after" -eq $((before + 1)) ] || die "$label: expected one appended host record ($before -> $after)"
  ok "$label -> $(jq -r '.next_action' "$work/out.json")"
}

# expect_refusal <label> <reason> <hma args...>: command must fail with exactly
# <reason> and leave both the host chain and the core chain unchanged.
expect_refusal() {
  local label="$1" reason="$2"; shift 2
  local hb cb fb err
  hb="$(host_count)"; cb="$(core_lines)"; fb="$(store_fingerprint)"
  if "$hma" "$@" --input "$in" --store "$store" >/dev/null 2>"$work/err"; then
    die "$label: accepted, expected $reason"
  fi
  err="$(cat "$work/err")"
  [ "$err" = "$reason" ] || die "$label: got '$err', expected '$reason'"
  [ "$(host_count)" -eq "$hb" ] || die "$label: host chain changed on refusal"
  [ "$(core_lines)" -eq "$cb" ] || die "$label: core chain changed on refusal"
  [ "$(store_fingerprint)" = "$fb" ] || die "$label: store bytes changed on refusal"
  ok "$label refused: $reason"
}

echo "== synthetic HMA host dogfood (scratch: $work)"

"$hostinput" seed-core --store "$store" --run_id "$run_id"
ok "core run seeded with no approvals"

# Phase 1: no route approval exists in core.
emit decision "$approved"
expect_refusal "decision without core route approval" HOST_ROUTE_APPROVAL_MISSING host decision
emit transition-impl-review "$approved"
expect_refusal "impl->review gate without host records" HOST_ORCHESTRATION_DECISION_MISSING transition

# Phase 2: synthetic route-approval fixture (explicitly not human).
"$hostinput" seed-synthetic-route-approval --store "$store" --run_id "$run_id" --unit_id "$unit" --digest "$approved"
ok "synthetic route-approval fixture seeded (approver labelled non-human)"

emit decision "$forged"
expect_refusal "forged decision" HOST_ROUTE_APPROVAL_MISMATCH host decision
emit decision "$approved"
expect_ok "decision" host decision

emit preflight-impl "$forged"
expect_refusal "forged implementation preflight" HOST_ROUTE_APPROVAL_MISMATCH host preflight
emit preflight-impl "$approved"
expect_ok "implementation preflight" host preflight

emit dispatch-impl "$forged"
expect_refusal "forged implementation dispatch" HOST_ROUTE_APPROVAL_MISMATCH host dispatch
emit dispatch-impl "$approved"
expect_ok "implementation dispatch" host dispatch

# Host gate now passes; core must still refuse because no approval is supplied.
emit transition-impl-review "$approved"
expect_refusal "impl->review with host gate satisfied but no human approval" "missing host input" transition

emit transition-review-verification "$approved"
expect_refusal "review->verification gate before review" HOST_INDEPENDENT_REVIEW_MISSING transition

emit preflight-review "$approved"
expect_ok "review preflight" host preflight
emit dispatch-review "$approved"
expect_ok "review dispatch" host dispatch
emit review "$approved"
expect_ok "independent review APPROVE" host review

emit transition-review-verification "$approved"
expect_refusal "review->verification with host gate satisfied but no human approval" "missing host input" transition

# Final state: six host records, core holds only the seed and the synthetic fixture.
[ "$(host_count)" -eq 6 ] || die "expected 6 host records, got $(host_count)"
approvers="$(jq -r 'select(.approval) | .approval.approver' "$store/$run_id.jsonl" | sort -u)"
[ "$approvers" = "SYNTHETIC-DOGFOOD-FIXTURE (not a human approval)" ] || die "unexpected approvers in core: $approvers"
ok "core contains no approval other than the labelled synthetic fixture"

echo "== PASS ($pass checks)"
