// Friction summary: a small, best-effort, in-memory counter the runner
// accumulates across one full run() call and prints once at the end,
// whether the run succeeds, is blocked, or fails. This is the "measure
// friction" instrument the handoff's §4 P2 calls for (approval requests,
// blocked/recoverable-vs-not operations, recovery attempts and outcomes,
// final outcome) -- deliberately not a new host/core record kind, not a
// new file format, and not a dashboard: just counters bumped as steps
// execute and one printed line.
//
// What is and is not counted here (see each call site's own comment for
// why): every r.confirm() call is an approval request. stepWorkerDispatch's
// one bounded retry counts as exactly one recovery attempt the moment it
// starts, no matter which outcome it then hits (a clean success, an
// out-of-scope commit, a non-zero exit, a launch error, or exhaustion with
// no commit) -- it is a recoverable block only on success, unrecoverable
// otherwise. stepReviewerDispatch has no retry at all: agent-run-ledger's
// run-id exclusivity means a crashed/no-output reviewer dispatch cannot be
// retried with the same invocation, so it escalates immediately as an
// unrecoverable block with zero recovery attempts (see its doc comment;
// an earlier version here referred to "stepReviewerDispatch's new
// [retry]", which this describes was removed). A step that fails with no
// retry possible at all (a non-AVAILABLE preflight, a forbidden-file
// commit, an agent-dispatch launch failure) is an unrecoverable block
// immediately. Plain local errors unrelated to the dispatch/retry
// machinery (a failed git command, a store read, doTransition's own
// non-retried failures, and -- not previously called out here -- a
// `host review-import` or `host reconcile` rejection) are not separately
// instrumented as a counted "block": they still fail the run and still
// show up in outcome and in hmarun's own error output, just not broken out
// into these counters. A real operational stop can therefore show 0 in
// "blocked operation(s)" here; the outcome and the error text are the
// authoritative record of what happened, these counters are a friction
// estimate, not a complete tally.
package main

import "fmt"

// runSummary is the friction counters for one `hmarun run` invocation.
type runSummary struct {
	approvals            int
	blockedRecoverable   int
	blockedUnrecoverable int
	recoveryAttempts     int
	recoverySucceeded    int
	outcome              string
}

// printSummary prints r.summary's one-line friction summary to stdout.
func (r *runner) printSummary() {
	s := r.summary
	fmt.Fprintf(r.stdout, "Run summary: %d approval(s) requested, %d blocked operation(s) (%d recoverable, %d unrecoverable), %d recovery attempt(s) (%d succeeded), outcome: %s.\n",
		s.approvals, s.blockedRecoverable+s.blockedUnrecoverable, s.blockedRecoverable, s.blockedUnrecoverable, s.recoveryAttempts, s.recoverySucceeded, s.outcome)
}
