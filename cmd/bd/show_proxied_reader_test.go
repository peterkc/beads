package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/steveyegge/beads/internal/storage"
	"github.com/steveyegge/beads/internal/storage/domain"
	"github.com/steveyegge/beads/internal/storage/uow"
	"github.com/steveyegge/beads/internal/types"
	"github.com/steveyegge/beads/issueops"
)

// readerProvider keeps the real per-ID reader over a failing lookup for blank
// IDs and excluded modes. The batch double pins the CLI's response to missing
// items and hard errors; storage conformance covers the batch role itself.
type readerProvider struct{ lookupOnlyProvider }

func (p readerProvider) IssueReader() (issueops.Reader, error) {
	return uow.NewIssueReader(p)
}

func (p readerProvider) DetailBatchReader() (issueops.DetailBatchReader, error) {
	uc, _ := p.issues.(stubLookupIssueUC)
	return &showBatchRecorder{err: uc.hardErr}, nil
}

var _ uow.IssueReaderSource = readerProvider{}

func withStubbedProxiedReader(t *testing.T, hardErr error) {
	t.Helper()
	oldProvider, oldJSON := uowProvider, jsonOutput
	uowProvider = readerProvider{lookupOnlyProvider{issues: stubLookupIssueUC{hardErr: hardErr}}}
	jsonOutput = true
	t.Cleanup(func() {
		uowProvider = oldProvider
		jsonOutput = oldJSON
	})
}

// TestShowProxiedJSONMissingIDKeepsItsContract guards the missing-item
// diagnostics of the proxied detail view. The corpus pins a PAIR
// of outputs for a missing id under --json — the human line on stderr and the
// error envelope on stdout, exit 1. A missing batch item must keep that same
// pair rather than silently disappearing from the command's diagnostics.
func TestShowProxiedJSONMissingIDKeepsItsContract(t *testing.T) {
	withStubbedProxiedReader(t, nil)

	var err error
	var stderr string
	stdout := captureStdout(t, func() error {
		stderr = captureStderrDuring(t, func() {
			err = runShowProxiedServer(&cobra.Command{}, context.Background(), []string{stubMissingID})
		})
		return nil
	})
	var envelope map[string]any
	if jsonErr := json.Unmarshal([]byte(stdout), &envelope); jsonErr != nil || envelope["error"] == nil {
		t.Fatalf("missing ID stdout = %q, want JSON error object: %v", stdout, jsonErr)
	}

	if err == nil {
		t.Error("a batch that found nothing exited zero")
	}
	if got, want := strings.SplitN(stderr, "\n", 2)[0], "Issue "+stubMissingID+" not found"; got != want {
		t.Errorf("stderr first line = %q, want %q", got, want)
	}
	if !strings.Contains(stderr, "bd history "+stubMissingID) {
		t.Errorf("expected stderr to hint at checking 'bd history %s', got: %q", stubMissingID, stderr)
	}
}

// TestShowProxiedJSONBackendFailureAborts pins the one behavior this move
// changed on purpose. Resolution and assembly used to be two calls, and a
// backend failure was reported per id and skipped past when it surfaced in the
// first, aborted on when it surfaced in the second. They are one call now, so
// there is one answer, and abort is the one worth keeping: a JSON array
// missing the rows a database error swallowed is indistinguishable from a
// complete one.
func TestShowProxiedJSONBackendFailureAborts(t *testing.T) {
	withStubbedProxiedReader(t, errors.New(stubBackendError))

	var err error
	var stderr string
	stdout := captureStdout(t, func() error {
		stderr = captureStderrDuring(t, func() {
			err = runShowProxiedServer(&cobra.Command{}, context.Background(), []string{stubMissingID, "bd-hard-error"})
		})
		return nil
	})
	assertShowBatchError(t, stdout, stderr, err)

	if err == nil {
		t.Fatal("a backend failure exited zero")
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		t.Errorf("backend failure reported as a missing issue: %v", err)
	}
}

// TestShowProxiedTextRouteDoesNotOpenAReader pins the other half of the split:
// the terminal rendering needs the raw issue and the wisp flag, which the
// detail contract does not carry, so it stays on the CLI's own resolution and
// must not pay for a reader it cannot use.
func TestShowProxiedTextRouteDoesNotOpenAReader(t *testing.T) {
	withStubbedProxiedLookup(t, nil) // a provider with NO capability accessor

	var err error
	stderr := captureStderrDuring(t, func() {
		err = runShowProxiedServer(&cobra.Command{}, context.Background(), []string{stubMissingID})
	})

	if err == nil {
		t.Error("a batch that found nothing exited zero")
	}
	if got, want := strings.SplitN(stderr, "\n", 2)[0], "Issue "+stubMissingID+" not found"; got != want {
		t.Errorf("stderr first line = %q, want %q", got, want)
	}
	if !strings.Contains(stderr, "bd history "+stubMissingID) {
		t.Errorf("expected stderr to hint at checking 'bd history %s', got: %q", stubMissingID, stderr)
	}
}

// showBatchRecorder records the front door's request and answers only the
// detail reads these dispatch tests exercise. It does not model storage.
type showBatchRecorder struct {
	requests []issueops.DetailBatchRequest
	details  map[string]*issueops.IssueDetails
	err      error
}

func (r *showBatchRecorder) GetBatch(_ context.Context, req issueops.DetailBatchRequest) (issueops.DetailBatchResult, error) {
	r.requests = append(r.requests, req)
	if r.err != nil {
		return issueops.DetailBatchResult{}, r.err
	}
	items := make([]issueops.DetailBatchItem, len(req.IDs))
	for i, id := range req.IDs {
		details := r.details[id]
		items[i] = issueops.DetailBatchItem{ID: id, Found: details != nil, Issue: details}
	}
	return issueops.DetailBatchResult{Items: items}, nil
}

type showGetRecorder struct {
	issueops.Reader
	ids []string
}

func (r *showGetRecorder) Get(_ context.Context, req issueops.GetRequest) (*issueops.IssueDetails, error) {
	r.ids = append(r.ids, req.ID)
	return nil, storage.ErrNotFound
}

type showBatchProvider struct {
	lookupOnlyProvider
	batch  *showBatchRecorder
	reader *showGetRecorder
}

func (p showBatchProvider) DetailBatchReader() (issueops.DetailBatchReader, error) {
	return p.batch, nil
}
func (p showBatchProvider) IssueReader() (issueops.Reader, error) { return p.reader, nil }

func assertShowBatchError(t *testing.T, stdout, stderr string, err error) {
	t.Helper()
	if code, ok := exitCodeFromError(err); !ok || code != 1 {
		t.Fatalf("error = %v, want exit 1", err)
	}
	var envelope map[string]any
	if jsonErr := json.Unmarshal([]byte(stdout), &envelope); jsonErr != nil {
		t.Fatalf("stdout = %q, want JSON error object, not an array: %v", stdout, jsonErr)
	}
	if message, _ := envelope["error"].(string); !strings.Contains(message, stubBackendError) {
		t.Errorf("JSON error = %v, want backend error", envelope)
	}
	if stderr != "" {
		t.Errorf("hard error added stderr output: %q", stderr)
	}
}

func TestShowProxiedJSONBatchKeepsOrderOptionsAndBlankFallback(t *testing.T) {
	t.Setenv("BD_JSON_ENVELOPE", "")
	oldProvider, oldJSON := uowProvider, jsonOutput
	jsonOutput = true
	t.Cleanup(func() { uowProvider, jsonOutput = oldProvider, oldJSON })
	batch := &showBatchRecorder{details: map[string]*issueops.IssueDetails{
		"bd-a": types.NewIssueDetails(types.Issue{ID: "bd-a", Title: "A"}),
		"null": types.NewIssueDetails(types.Issue{ID: "null", Title: "Proxied sentinel is an exact ID"}),
	}}
	reader := &showGetRecorder{}
	uowProvider = showBatchProvider{lookupOnlyProvider: lookupOnlyProvider{issues: stubLookupIssueUC{}}, batch: batch, reader: reader}
	cmd := &cobra.Command{}
	for _, flag := range []string{"include-dependents", "include-comments", "brief-deps"} {
		cmd.Flags().Bool(flag, true, "")
	}
	// The empty string is left out: today it aborts the call (see
	// TestShowProxiedJSONBatchOnlyBlankIDsKeepsGet).
	ids := []string{"bd-a", stubMissingID, "  ", "null", "bd-a"}
	var err error
	var stderr string
	stdout := captureStdout(t, func() error {
		stderr = captureStderrDuring(t, func() { err = runShowProxiedServer(cmd, context.Background(), ids) })
		return nil
	})
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	wantRequest := issueops.DetailBatchRequest{IDs: []string{"bd-a", stubMissingID, "null", "bd-a"}, IncludeDependents: true, IncludeComments: true, BriefDeps: true}
	if !reflect.DeepEqual(batch.requests, []issueops.DetailBatchRequest{wantRequest}) {
		t.Errorf("batch requests = %+v, want one %+v", batch.requests, wantRequest)
	}
	if !reflect.DeepEqual(reader.ids, []string{"  "}) {
		t.Errorf("per-ID Get = %q, want only blank IDs", reader.ids)
	}
	var details []types.IssueDetails
	if err := json.Unmarshal([]byte(stdout), &details); err != nil {
		t.Fatalf("decode: %v\n%s", err, stdout)
	}
	var gotIDs []string
	for _, detail := range details {
		gotIDs = append(gotIDs, detail.ID)
	}
	if !reflect.DeepEqual(gotIDs, []string{"bd-a", "null", "bd-a"}) {
		t.Errorf("output IDs = %v", gotIDs)
	}
	wantStderr := ""
	for _, id := range []string{stubMissingID, "  "} {
		wantStderr += "Issue " + id + " not found\nHint: " + showNotFoundHint(id) + "\n"
	}
	if stderr != wantStderr {
		t.Errorf("stderr = %q, want %q", stderr, wantStderr)
	}
}

// missIssueRepo answers Get the way the real repository answers a miss: a bare
// sql.ErrNoRows. Everything else is a nil call these lookups must not reach.
type missIssueRepo struct{ domain.IssueSQLRepository }

func (missIssueRepo) Get(context.Context, string, domain.IssueTableOpts) (*types.Issue, error) {
	return nil, sql.ErrNoRows
}

// TestShowProxiedJSONBatchOnlyBlankIDsKeepsGet runs blank IDs through the real
// reader and the real domain use case, so the empty-ID guard is today's: the
// empty string aborts the call, while a whitespace-only ID is an ordinary miss.
func TestShowProxiedJSONBatchOnlyBlankIDsKeepsGet(t *testing.T) {
	issues := domain.NewIssueUseCase(missIssueRepo{}, nil, nil, nil, nil, nil, nil, nil, nil)
	for _, tc := range []struct {
		name, id, wantError, wantStderr string
	}{
		{name: "empty aborts", id: "", wantError: "get: id must not be empty"},
		{name: "whitespace is a miss", id: "  ", wantError: "no issues found matching the provided IDs",
			wantStderr: "Issue " + "  " + " not found\nHint: " + showNotFoundHint("  ") + "\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldProvider, oldJSON := uowProvider, jsonOutput
			uowProvider = readerProvider{lookupOnlyProvider{issues: issues}}
			jsonOutput = true
			t.Cleanup(func() { uowProvider, jsonOutput = oldProvider, oldJSON })
			var err error
			var stderr string
			stdout := captureStdout(t, func() error {
				stderr = captureStderrDuring(t, func() { err = runShowProxiedServer(&cobra.Command{}, context.Background(), []string{tc.id}) })
				return nil
			})
			if code, ok := exitCodeFromError(err); !ok || code != 1 {
				t.Fatalf("error = %v, want exit 1", err)
			}
			var envelope map[string]any
			if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
				t.Fatalf("stdout = %q, want a JSON error object, not an array: %v", stdout, err)
			}
			if message, _ := envelope["error"].(string); message != tc.wantError {
				t.Errorf("JSON error = %q, want %q", message, tc.wantError)
			}
			if stderr != tc.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr, tc.wantStderr)
			}
		})
	}
}

// The --current dispatch needs a selected ID before the detail reader runs.
type showCurrentLookupUC struct{ stubLookupIssueUC }

func (showCurrentLookupUC) SearchIssues(context.Context, string, types.IssueFilter) (domain.SearchPage, error) {
	return domain.SearchPage{Items: []*types.Issue{{ID: "bd-current"}}}, nil
}

func TestShowProxiedJSONBatchExcludedModes(t *testing.T) {
	oldProvider, oldJSON, oldActor := uowProvider, jsonOutput, actor
	jsonOutput, actor = true, "tester"
	t.Cleanup(func() { uowProvider, jsonOutput, actor = oldProvider, oldJSON, oldActor })
	for _, mode := range []string{"current", "long"} {
		t.Run(mode, func(t *testing.T) {
			batch, reader := &showBatchRecorder{}, &showGetRecorder{}
			uowProvider = showBatchProvider{lookupOnlyProvider: lookupOnlyProvider{issues: showCurrentLookupUC{}}, batch: batch, reader: reader}
			cmd := &cobra.Command{}
			cmd.Flags().Bool(mode, true, "")
			ids := []string{"bd-current"}
			if mode == "current" {
				ids = nil
			}
			var err error
			_ = captureStdout(t, func() error {
				_ = captureStderrDuring(t, func() { err = runShowProxiedServer(cmd, context.Background(), ids) })
				return nil
			})
			if err == nil {
				t.Fatal("missing per-ID result exited zero")
			}
			if len(batch.requests) != 0 {
				t.Errorf("excluded mode batched: %+v", batch.requests)
			}
			if !reflect.DeepEqual(reader.ids, []string{"bd-current"}) {
				t.Errorf("per-ID Get = %v", reader.ids)
			}
		})
	}
}
