package inventory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func discoverOne(t *testing.T, command Command, mutate func(*Request)) Entry {
	t.Helper()
	request := Request{Commands: []Command{command}, CapturedAt: time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC), Freshness: time.Minute}
	if mutate != nil {
		mutate(&request)
	}
	inventory, err := Discover(context.Background(), request)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(inventory.Entries) != 1 {
		t.Fatalf("entries = %+v", inventory.Entries)
	}
	return inventory.Entries[0]
}

func TestDiscoverCapturesNormalizedCredentialFreeInventory(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 123456789, time.FixedZone("test", -4*60*60))
	commands := []Command{
		{ID: "models", Kind: KindModelAvailability, Subject: "review-models", Executable: "/usr/bin/printf", Argv: []string{"%s\\n", "model-z", "model-a", "model-z"}},
		{ID: "version", Kind: KindVersion, Subject: "printf", Executable: "/usr/bin/printf", Argv: []string{"%s\\n", "tool version 1.2"}},
		{ID: "presence", Kind: KindExecutablePresence, Subject: "printf", Executable: "/usr/bin/printf"},
	}
	inventory, err := Discover(context.Background(), Request{Commands: commands, CapturedAt: now, Freshness: 10 * time.Minute})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if inventory.Version != VersionV1 {
		t.Fatalf("version = %q, want %q", inventory.Version, VersionV1)
	}
	if inventory.CapturedAt != "2026-09-03T16:00:00.123456789Z" || inventory.FreshUntil != "2026-09-03T16:10:00.123456789Z" {
		t.Fatalf("timestamps = %q / %q", inventory.CapturedAt, inventory.FreshUntil)
	}
	if len(inventory.Entries) != 3 || inventory.Entries[0].ID != "models" || inventory.Entries[1].ID != "presence" || inventory.Entries[2].ID != "version" {
		t.Fatalf("entries were not normalized by ID: %+v", inventory.Entries)
	}
	models := inventory.Entries[0]
	if models.Status != StatusCompleted || models.ExitCode != 0 || !reflect.DeepEqual(models.Models, []string{"model-a", "model-z"}) || models.OutputDigest == "" {
		t.Fatalf("model entry = %+v", models)
	}
	if inventory.Entries[2].Version != "tool version 1.2" || inventory.Entries[2].Status != StatusCompleted {
		t.Fatalf("version entry = %+v", inventory.Entries[2])
	}
	if inventory.Entries[1].Status != StatusPresent || inventory.Entries[1].ExitCode != -1 || inventory.Entries[1].OutputDigest != "" {
		t.Fatalf("presence entry = %+v", inventory.Entries[1])
	}
	if !inventory.Fresh(now.Add(9*time.Minute+time.Nanosecond)) || inventory.Fresh(now.Add(10*time.Minute)) {
		t.Fatal("freshness boundary was not enforced")
	}
	digest, err := Digest(inventory)
	if err != nil || digest != inventory.Digest {
		t.Fatalf("Digest() = %q, %v; inventory digest = %q", digest, err, inventory.Digest)
	}

	reversed := append([]Command(nil), commands...)
	reversed[0], reversed[2] = reversed[2], reversed[0]
	other, err := Discover(context.Background(), Request{Commands: reversed, CapturedAt: now, Freshness: 10 * time.Minute})
	if err != nil {
		t.Fatalf("Discover(reordered) error = %v", err)
	}
	if other.Digest != inventory.Digest {
		t.Fatalf("equivalent command ordering changed digest: %s != %s", other.Digest, inventory.Digest)
	}
}

func TestDiscoverRecordsMissingExecutableWithoutGuessing(t *testing.T) {
	entry := discoverOne(t, Command{ID: "missing", Kind: KindVersion, Subject: "missing-tool", Executable: "/hma/missing/tool"}, nil)
	if entry.Status != StatusAbsent || entry.ExitCode != -1 || entry.Version != "" || entry.OutputDigest != "" {
		t.Fatalf("missing entry = %+v", entry)
	}
}

func TestDiscoverRejectsNonAllowlistedCommandShape(t *testing.T) {
	for name, command := range map[string]Command{
		"shell string":  {ID: "shell", Kind: KindVersion, Subject: "bad", Executable: "sh -c", Argv: []string{"echo unsafe"}},
		"bare name":     {ID: "bare", Kind: KindVersion, Subject: "bad", Executable: "printf"},
		"relative path": {ID: "relative", Kind: KindVersion, Subject: "bad", Executable: "bin/tool"},
		"unclean path":  {ID: "unclean", Kind: KindVersion, Subject: "bad", Executable: "/usr/bin/../bin/printf"},
		"unknown kind":  {ID: "kind", Kind: "shell", Subject: "bad", Executable: "/usr/bin/printf"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Discover(context.Background(), Request{Commands: []Command{command}, CapturedAt: time.Now(), Freshness: time.Minute})
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("error = %v, want ErrInvalidRequest", err)
			}
		})
	}
}

func TestDiscoverRejectsDuplicateAllowlistIDs(t *testing.T) {
	_, err := Discover(context.Background(), Request{
		Commands: []Command{
			{ID: "same", Kind: KindExecutablePresence, Subject: "one", Executable: "/usr/bin/printf"},
			{ID: "same", Kind: KindExecutablePresence, Subject: "two", Executable: "/usr/bin/printf"},
		},
		CapturedAt: time.Now(),
		Freshness:  time.Minute,
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestDiscoverDoesNotInheritAmbientEnvironment(t *testing.T) {
	t.Setenv("HMA_INVENTORY_SECRET", "must-not-leak")
	entry := discoverOne(t, Command{ID: "env", Kind: KindModelAvailability, Subject: "env", Executable: "/usr/bin/env"}, nil)
	if entry.Status != StatusCompleted || len(entry.Models) != 0 {
		t.Fatalf("ambient environment reached the discovery command: %+v", entry)
	}
	supplied := discoverOne(t, Command{ID: "env", Kind: KindModelAvailability, Subject: "env", Executable: "/usr/bin/env"}, func(r *Request) {
		r.Environment = []string{"LANG=C"}
	})
	if !reflect.DeepEqual(supplied.Models, []string{"LANG=C"}) {
		t.Fatalf("host-supplied environment = %+v", supplied.Models)
	}
}

func TestDiscoverRecordsTimeoutAsUnavailable(t *testing.T) {
	entry := discoverOne(t, Command{ID: "slow", Kind: KindVersion, Subject: "sleep", Executable: "/bin/sleep", Argv: []string{"5"}}, func(r *Request) {
		r.CommandTimeout = 100 * time.Millisecond
	})
	if entry.Status != StatusTimedOut || entry.Version != "" || entry.OutputDigest != "" {
		t.Fatalf("timed-out entry = %+v", entry)
	}
}

func TestDiscoverAbortsWhenCallerCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Discover(ctx, Request{
		Commands:   []Command{{ID: "slow", Kind: KindVersion, Subject: "sleep", Executable: "/bin/sleep", Argv: []string{"5"}}},
		CapturedAt: time.Now(),
		Freshness:  time.Minute,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestDiscoverRecordsStartFailureAsUnavailable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-program")
	if err := os.WriteFile(path, []byte("not an executable format\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	entry := discoverOne(t, Command{ID: "broken", Kind: KindVersion, Subject: "broken", Executable: path}, nil)
	if entry.Status != StatusStartFailed || entry.ExitCode != -1 || entry.Version != "" {
		t.Fatalf("start-failure entry = %+v", entry)
	}
}

func TestDiscoverRecordsNonzeroExitWithoutParsing(t *testing.T) {
	entry := discoverOne(t, Command{ID: "fail", Kind: KindVersion, Subject: "sh", Executable: "/bin/sh", Argv: []string{"-c", "echo 1.0; exit 3"}}, nil)
	if entry.Status != StatusExitedNonzero || entry.ExitCode != 3 || entry.Version != "" || entry.OutputDigest == "" {
		t.Fatalf("nonzero entry = %+v", entry)
	}
}

func TestDiscoverDropsUnboundedOrUnprintableParsedFields(t *testing.T) {
	long := strings.Repeat("v", maxFieldBytes+1)
	entry := discoverOne(t, Command{ID: "long", Kind: KindVersion, Subject: "printf", Executable: "/usr/bin/printf", Argv: []string{"%s\\n", long}}, nil)
	if entry.Version != "" || !entry.OutputUnparsed {
		t.Fatalf("oversized version was retained: %+v", entry)
	}
	models := discoverOne(t, Command{ID: "models", Kind: KindModelAvailability, Subject: "printf", Executable: "/usr/bin/printf", Argv: []string{"%s\\n", "model-a", "bad\x1b[0m"}}, nil)
	if models.Models != nil || !models.OutputUnparsed {
		t.Fatalf("unprintable model line was retained: %+v", models)
	}
}

func TestDiscoverDoesNotParseTruncatedOutput(t *testing.T) {
	entry := discoverOne(t, Command{ID: "models", Kind: KindModelAvailability, Subject: "printf", Executable: "/usr/bin/printf", Argv: []string{"%s\\n", "model-a", "model-b"}}, func(r *Request) {
		r.MaxOutputBytes = 8
	})
	if !entry.OutputTruncated || !entry.OutputUnparsed || entry.Models != nil {
		t.Fatalf("truncated output was parsed: %+v", entry)
	}
}

func TestOutputDigestIsUnambiguous(t *testing.T) {
	if digestOutput([]byte("ab"), []byte("c")) == digestOutput([]byte("a"), []byte("bc")) {
		t.Fatal("stdout/stderr split is not framed: digests collide")
	}
}

func TestInventoryDigestExcludesSelfDigest(t *testing.T) {
	inventory := Inventory{Version: VersionV1, CapturedAt: "2026-09-03T16:00:00Z", FreshUntil: "2026-09-03T16:01:00Z", Entries: []Entry{{ID: "one", Kind: KindExecutablePresence, Subject: "one", Executable: "/usr/bin/printf", Status: StatusPresent, ExitCode: -1}}}
	first, err := Digest(inventory)
	if err != nil {
		t.Fatal(err)
	}
	inventory.Digest = "sha256:some-previous-value"
	second, err := Digest(inventory)
	if err != nil || first != second {
		t.Fatalf("self digest changed content digest: %q != %q (err %v)", first, second, err)
	}
}
