//go:build cgo

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/internal/configfile"
	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/types"
	"github.com/steveyegge/beads/issueops"
)

// The embedded store exercises real resolution and detail assembly. This
// wrapper observes only the batching and fallback boundary owned by show.
type showBatchStore struct {
	storage.DoltStorage
	requests []issueops.DetailBatchRequest
	getIDs   []string
	batchErr error
}

func (s *showBatchStore) DetailBatchReader() (issueops.DetailBatchReader, error) {
	rd, err := s.DoltStorage.DetailBatchReader()
	if err != nil {
		return nil, err
	}
	return &showStoreBatchRecorder{DetailBatchReader: rd, store: s}, nil
}

func (s *showBatchStore) GetIssue(ctx context.Context, id string) (*types.Issue, error) {
	s.getIDs = append(s.getIDs, id)
	return s.DoltStorage.GetIssue(ctx, id)
}

type showStoreBatchRecorder struct {
	issueops.DetailBatchReader
	store *showBatchStore
}

func (r *showStoreBatchRecorder) GetBatch(ctx context.Context, req issueops.DetailBatchRequest) (issueops.DetailBatchResult, error) {
	r.store.requests = append(r.store.requests, req)
	if r.store.batchErr != nil {
		return issueops.DetailBatchResult{}, r.store.batchErr
	}
	return r.DetailBatchReader.GetBatch(ctx, req)
}

func setupShowBatchEmbedded(t *testing.T) (*showBatchStore, string) {
	t.Helper()
	isolateBeadsDirForTest(t)
	ensureCleanGlobalState(t)
	saveAndRestoreGlobals(t)
	initConfigForTest(t)
	dir := t.TempDir()
	initGitRepoAt(t, dir)
	t.Chdir(dir)
	beadsDir := filepath.Join(dir, ".beads")
	s := openStore(t, beadsDir, "batchshow")
	if err := s.SetConfig(t.Context(), "issue_prefix", "bs"); err != nil {
		t.Fatal(err)
	}
	cfg := &configfile.Config{Backend: configfile.BackendDolt, DoltDatabase: "batchshow"}
	if err := cfg.Save(beadsDir); err != nil {
		t.Fatal(err)
	}
	wrapped := &showBatchStore{DoltStorage: s}
	store = wrapped
	dbPath = filepath.Join(beadsDir, "dolt")
	oldJSON, oldProxied, oldCtx := jsonOutput, proxiedServerMode, rootCtx
	jsonOutput, proxiedServerMode, rootCtx = true, false, t.Context()
	t.Cleanup(func() { jsonOutput, proxiedServerMode, rootCtx = oldJSON, oldProxied, oldCtx })
	t.Setenv("BD_JSON_ENVELOPE", "")
	return wrapped, dir
}

func createShowBatchIssue(t *testing.T, s storage.DoltStorage, id string, ephemeral bool) {
	t.Helper()
	if err := s.CreateIssue(t.Context(), &types.Issue{ID: id, Title: id, Status: types.StatusOpen, Priority: 2, IssueType: types.TypeTask, Ephemeral: ephemeral}, "tester"); err != nil {
		t.Fatal(err)
	}
}

func TestShowJSONBatchEmbeddedResolutionAndOrder(t *testing.T) {
	s, dir := setupShowBatchEmbedded(t)
	// Stored sentinel IDs must still be refused before any exact lookup.
	for _, id := range []string{"bs-abcdef01", "bs-abc99999", "bs-second", "null", "UNDEFINED", "None", "nIL"} {
		createShowBatchIssue(t, s, id, false)
	}
	createShowBatchIssue(t, s, "bs-wisp-eee11111", true)
	// Reuse the routing shape in comment_routed_exact_test.go, but with
	// embedded stores and metadata so the foreign open is read-only.
	rigDir := filepath.Join(dir, "rig", ".beads")
	rig := openStore(t, rigDir, "routedshow")
	if err := rig.SetConfig(t.Context(), "issue_prefix", "gt"); err != nil {
		t.Fatal(err)
	}
	createShowBatchIssue(t, rig, "gt-routed", false)
	cfg := &configfile.Config{Backend: configfile.BackendDolt, DoltDatabase: "routedshow"}
	if err := cfg.Save(rigDir); err != nil {
		t.Fatal(err)
	}
	if err := rig.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".beads", "routes.jsonl"), []byte(`{"prefix":"gt-","path":"rig"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	ids := []string{"bs-second", "abcd", "bs-missing", "bs-second", "bs-wisp-eee11111", "", "  ", "null", "UNDEFINED", "None", "nIL", "abc", "gt-routed", "bs-abcdef01"}
	var err error
	var stderr string
	cmd := &cobra.Command{}
	for _, flag := range []string{"include-dependents", "include-comments", "brief-deps"} {
		cmd.Flags().Bool(flag, true, "")
	}
	stdout := captureStdout(t, func() error {
		stderr = captureStderrDuring(t, func() { err = showCmd.RunE(cmd, ids) })
		return nil
	})
	if err != nil {
		t.Fatalf("show: %v\n%s\n%s", err, stdout, stderr)
	}
	wantBatch := []string{"bs-second", "abcd", "bs-missing", "bs-second", "bs-wisp-eee11111", "abc", "gt-routed", "bs-abcdef01"}
	wantRequest := issueops.DetailBatchRequest{IDs: wantBatch, IncludeDependents: true, IncludeComments: true, BriefDeps: true}
	if !reflect.DeepEqual(s.requests, []issueops.DetailBatchRequest{wantRequest}) {
		t.Errorf("batch requests = %+v, want one %+v", s.requests, wantRequest)
	}
	if !reflect.DeepEqual(s.getIDs, []string{"bs-abcdef01"}) {
		t.Errorf("local per-ID Get = %v, want partial only", s.getIDs)
	}
	var details []types.IssueDetails
	if err := json.Unmarshal([]byte(stdout), &details); err != nil {
		t.Fatalf("decode: %v\n%s", err, stdout)
	}
	var got []string
	for _, d := range details {
		got = append(got, d.ID)
	}
	want := []string{"bs-second", "bs-abcdef01", "bs-second", "bs-wisp-eee11111", "gt-routed", "bs-abcdef01"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("IDs = %v, want %v", got, want)
	}
	for _, line := range []string{"Issue bs-missing not found", "refusing an empty string", `refusing "null"`, `refusing "UNDEFINED"`, `refusing "None"`, `refusing "nIL"`, "ambiguous issue ID", `note: "abcd" is not an exact issue ID; resolved to bs-abcdef01`} {
		if !strings.Contains(stderr, line) {
			t.Errorf("stderr missing %q: %s", line, stderr)
		}
	}
	if strings.Count(stderr, "refusing an empty string") != 2 {
		t.Errorf("blank refusals: %q", stderr)
	}
}

func TestShowJSONBatchEmbeddedFailureDoesNotEmitArray(t *testing.T) {
	s, _ := setupShowBatchEmbedded(t)
	s.batchErr = errors.New(stubBackendError)
	var err error
	var stderr string
	stdout := captureStdout(t, func() error {
		stderr = captureStderrDuring(t, func() { err = showCmd.RunE(&cobra.Command{}, []string{stubMissingID, "bs-hard-error"}) })
		return nil
	})
	assertShowBatchError(t, stdout, stderr, err)
	if len(s.requests) != 1 || !reflect.DeepEqual(s.requests[0].IDs, []string{stubMissingID, "bs-hard-error"}) {
		t.Errorf("batch requests = %+v", s.requests)
	}
	if len(s.getIDs) != 0 {
		t.Errorf("fallback ran after failure: %v", s.getIDs)
	}
}

func TestShowJSONBatchExcludedModes(t *testing.T) {
	s, _ := setupShowBatchEmbedded(t)
	createShowBatchIssue(t, s, "bs-current", false)
	if err := s.UpdateIssue(t.Context(), "bs-current", map[string]interface{}{"status": string(types.StatusInProgress), "assignee": "tester"}, "tester"); err != nil {
		t.Fatal(err)
	}
	oldActor := actor
	actor = "tester"
	t.Cleanup(func() { actor = oldActor })
	for _, tc := range []struct {
		name string
		ids  []string
	}{
		{"current", nil}, {"long", []string{"bs-current"}}, {"thread", []string{"bs-missing", "bs-current"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s.requests, s.getIDs = nil, nil
			cmd := &cobra.Command{}
			cmd.Flags().Bool(tc.name, true, "")
			var err error
			out := captureStdout(t, func() error {
				_ = captureStderrDuring(t, func() { err = showCmd.RunE(cmd, tc.ids) })
				return nil
			})
			if err != nil {
				t.Fatalf("show %s: %v\n%s", tc.name, err, out)
			}
			if len(s.requests) != 0 {
				t.Errorf("excluded mode batched: %+v", s.requests)
			}
			if !reflect.DeepEqual(s.getIDs, []string{"bs-current"}) {
				t.Errorf("per-ID Get = %v, want current issue", s.getIDs)
			}
			if !strings.Contains(out, `"id": "bs-current"`) {
				t.Errorf("output = %s", out)
			}
		})
	}
}

// One exact ID also takes the batch path; it is the call agents make most.
// For every option combination, stdout must be byte for byte what the old JSON
// path printed from Reader.Get, with no per-ID lookup.
func TestShowJSONBatchEmbeddedSingleIDMatchesReaderGet(t *testing.T) {
	s, _ := setupShowBatchEmbedded(t)
	ctx := t.Context()
	for _, id := range []string{"bs-one", "bs-dep", "bs-child"} {
		createShowBatchIssue(t, s, id, false)
	}
	for _, dep := range []*types.Dependency{
		{IssueID: "bs-one", DependsOnID: "bs-dep", Type: types.DepBlocks},
		{IssueID: "bs-child", DependsOnID: "bs-one", Type: types.DepBlocks},
	} {
		if err := s.AddDependency(ctx, dep, "tester"); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.AddLabel(ctx, "bs-one", "batch", "tester"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddIssueComment(ctx, "bs-one", "tester", "a comment"); err != nil {
		t.Fatal(err)
	}
	reader, err := s.DoltStorage.IssueReader()
	if err != nil {
		t.Fatal(err)
	}
	for mask := 0; mask < 8; mask++ {
		dependents, comments, brief := mask&1 != 0, mask&2 != 0, mask&4 != 0
		t.Run(fmt.Sprintf("dependents=%t,comments=%t,brief=%t", dependents, comments, brief), func(t *testing.T) {
			s.requests, s.getIDs = nil, nil
			cmd := &cobra.Command{}
			cmd.Flags().Bool("include-dependents", dependents, "")
			cmd.Flags().Bool("include-comments", comments, "")
			cmd.Flags().Bool("brief-deps", brief, "")
			var err error
			got := captureStdout(t, func() error {
				_ = captureStderrDuring(t, func() { err = showCmd.RunE(cmd, []string{"bs-one"}) })
				return nil
			})
			if err != nil {
				t.Fatalf("show: %v\n%s", err, got)
			}
			wantRequest := issueops.DetailBatchRequest{IDs: []string{"bs-one"}, IncludeDependents: dependents, IncludeComments: comments, BriefDeps: brief}
			if !reflect.DeepEqual(s.requests, []issueops.DetailBatchRequest{wantRequest}) {
				t.Errorf("batch requests = %+v, want one %+v", s.requests, wantRequest)
			}
			if len(s.getIDs) != 0 {
				t.Errorf("per-ID Get = %v, want none", s.getIDs)
			}
			details, err := reader.Get(ctx, showGetRequest("bs-one", dependents, comments, brief))
			if err != nil {
				t.Fatal(err)
			}
			want := captureStdout(t, func() error { return outputJSON([]interface{}{details}) })
			if got != want {
				t.Errorf("stdout differs from Reader.Get output\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}
