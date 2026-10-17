package model

import (
	"testing"
)

func TestGateBR2AttestationThroughRecord(t *testing.T) {
	mk := func(a *RouteAttestation) Record {
		return Record{Kind: KindEvidence, Version: 1, Metadata: RecordMetadata{RunID: "run", Sequence: 1, HeadAnchor: "record-head", Timestamp: "2026-10-15T12:00:00Z", Actor: "audit"}, Evidence: &EvidenceRef{Executable: "/bin/true", Argv: []string{}, WorkingDir: "/tmp", StartTimestamp: "2026-10-15T12:00:00Z", EndTimestamp: "2026-10-15T12:00:01Z", RepositoryIdentity: "repo", BaseRevision: "base", HeadRevision: "head", CapturingActor: "audit", Route: a}}
	}
	for _, a := range []*RouteAttestation{nil, func() *RouteAttestation { x := validAttestation(); return &x }()} {
		r := mk(a)
		if e := ValidateRecord(&r); e != nil {
			t.Fatal(e)
		}
	}
	a := validAttestation()
	a.ProviderFamily = ""
	r := mk(&a)
	if e := ValidateRecord(&r); e == nil {
		t.Fatal("invalid route attestation accepted through record boundary")
	}
}
