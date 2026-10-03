package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/steveyegge/beads/internal/httpapi/apigen"
	"github.com/steveyegge/beads/internal/types"
	"github.com/steveyegge/beads/issueops"
)

func TestBatchGetIssuesPreservesItemsAndOptions(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		query                       string
		comments, dependents, brief bool
	}{
		{"defaults", "", false, false, false},
		{"comments", "&include_comments=true", true, false, false},
		{"dependents", "&include_dependents=true", false, true, false},
		{"brief dependencies", "&brief_deps=true", false, false, true},
		{"all options", "&include_comments=true&include_dependents=true&brief_deps=true", true, true, true},
		{"explicit false", "&include_comments=false&include_dependents=false&brief_deps=false", false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			details := &issueops.IssueDetails{
				Issue:        *seededIssue("bd-2", "", types.StatusOpen),
				Comments:     seededComments("bd-2"),
				Dependents:   seededDependents(),
				Dependencies: seededDependents(),
			}
			result := issueops.DetailBatchResult{Items: []issueops.DetailBatchItem{
				{ID: "bd-2", Found: true, Issue: details},
				{ID: "missing"},
				{ID: "bd-1", Found: true, Issue: &issueops.IssueDetails{Issue: *seededIssue("bd-1", "", types.StatusClosed)}},
				{ID: "bd-2", Found: true, Issue: details},
			}}
			batch := &roleDetailBatchReader{result: result}
			reader := &roleReader{}
			ts := newTestServer(t, rolesConfig(Config{DetailBatchReader: batch, Reader: reader}))
			resp := ts.get(t, "/v0/beads/issues:batchGet?issue_id=bd-2&issue_id=missing&issue_id=bd-1&issue_id=bd-2"+tc.query)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", resp.StatusCode, readAll(t, resp))
			}
			want := issueops.DetailBatchRequest{IDs: []string{"bd-2", "missing", "bd-1", "bd-2"},
				IncludeComments: tc.comments, IncludeDependents: tc.dependents, BriefDeps: tc.brief}
			if got := batch.batchRequests(); !reflect.DeepEqual(got, []issueops.DetailBatchRequest{want}) {
				t.Errorf("batch requests = %+v, want %+v", got, want)
			}
			if got := reader.getRequests(); len(got) != 0 {
				t.Errorf("Reader.Get requests = %+v; batch reads must use only DetailBatchReader", got)
			}
			raw := []byte(readAll(t, resp))
			var body apigen.BatchIssueDetails
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("decode generated response: %v", err)
			}
			if len(body.Items) != 4 {
				t.Fatalf("items = %+v, want four ordered items", body.Items)
			}
			for i, want := range result.Items {
				got := body.Items[i]
				if got.IssueId != want.ID || got.Found != want.Found {
					t.Errorf("item %d = %+v, want id %q found %v", i, got, want.ID, want.Found)
				}
				if !reflect.DeepEqual(got.Issue, want.Issue) {
					t.Errorf("item %d details = %+v, want %+v", i, got.Issue, want.Issue)
				}
			}
			var members struct {
				Items []map[string]json.RawMessage `json:"items"`
			}
			if err := json.Unmarshal(raw, &members); err != nil {
				t.Fatal(err)
			}
			if _, ok := members.Items[1]["issue"]; ok {
				t.Error("missing item carries issue; the member must be absent, not null")
			}
			if string(members.Items[1]["found"]) != "false" {
				t.Error("missing item must carry found: false")
			}
		})
	}
}

func TestBatchGetIssuesRejectsInvalidIDs(t *testing.T) {
	for _, tc := range []struct{ name, query string }{
		{"no IDs", ""},
		{"101 IDs", "?issue_id=bd-1" + strings.Repeat("&issue_id=bd-1", 100)},
		{"empty ID", "?issue_id=bd-1&issue_id="},
		{"blank ID", "?issue_id=bd-1&issue_id=%20%09"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			batch := &roleDetailBatchReader{}
			ts := newTestServer(t, rolesConfig(Config{DetailBatchReader: batch}))
			resp := ts.get(t, "/v0/beads/issues:batchGet"+tc.query)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", resp.StatusCode, readAll(t, resp))
			}
			body := decodeBody(t, resp)
			if body["code"] != string(CodeInvalidArgument) || body["param"] != "issue_id" || body["reason"] != string(ReasonInvalidValue) {
				t.Errorf("problem = %v, want invalid_argument on issue_id with invalid_value", body)
			}
			if got := batch.batchRequests(); len(got) != 0 {
				t.Errorf("invalid request reached role: %+v", got)
			}
		})
	}
	t.Run("100 IDs are accepted", func(t *testing.T) {
		batch := &roleDetailBatchReader{}
		ts := newTestServer(t, rolesConfig(Config{DetailBatchReader: batch}))
		resp := ts.get(t, "/v0/beads/issues:batchGet?issue_id=bd-1"+strings.Repeat("&issue_id=bd-1", 99))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", resp.StatusCode, readAll(t, resp))
		}
		if got := batch.batchRequests(); len(got) != 1 || len(got[0].IDs) != 100 {
			t.Errorf("batch requests = %+v, want one request with 100 IDs", got)
		}
	})
}

func TestBatchGetIssuesRejectsInvalidOptions(t *testing.T) {
	for _, tc := range []struct {
		query, param string
		reason       Reason
	}{
		{"&bogus=1", "bogus", ReasonUnknownParameter},
		{"&include_comments=maybe", "include_comments", ReasonInvalidValue},
		{"&include_dependents=true&include_dependents=false", "include_dependents", ReasonInvalidValue},
		{"&brief_deps=maybe", "brief_deps", ReasonInvalidValue},
	} {
		t.Run(tc.query, func(t *testing.T) {
			batch := &roleDetailBatchReader{}
			ts := newTestServer(t, rolesConfig(Config{DetailBatchReader: batch}))
			resp := ts.get(t, "/v0/beads/issues:batchGet?issue_id=bd-1"+tc.query)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", resp.StatusCode, readAll(t, resp))
			}
			body := decodeBody(t, resp)
			if body["param"] != tc.param || body["reason"] != string(tc.reason) {
				t.Errorf("problem = %v, want %s with %s", body, tc.param, tc.reason)
			}
			if got := batch.batchRequests(); len(got) != 0 {
				t.Errorf("invalid options reached role: %+v", got)
			}
		})
	}
}

func TestBatchGetIssuesBackendFailureMatchesGetIssue(t *testing.T) {
	failure := errors.New("backend is unreachable")
	batch := &roleDetailBatchReader{err: failure}
	ts := newTestServer(t, rolesConfig(Config{DetailBatchReader: batch, Reader: &roleReader{err: failure}}))
	resp := ts.get(t, "/v0/beads/issues:batchGet?issue_id=bd-1&issue_id=bd-2")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", resp.StatusCode, readAll(t, resp))
	}
	body := decodeBody(t, resp)
	if body["code"] != string(CodeInternal) {
		t.Errorf("code = %v, want internal", body["code"])
	}
	if _, ok := body["items"]; ok {
		t.Error("backend failure carries partial items")
	}
	single := ts.get(t, "/v0/beads/issues/bd-1")
	if single.StatusCode != resp.StatusCode {
		t.Errorf("getIssue status = %d, batch status = %d", single.StatusCode, resp.StatusCode)
	}
	singleBody := decodeBody(t, single)
	for _, key := range []string{"code", "title", "status", "detail"} {
		if !reflect.DeepEqual(body[key], singleBody[key]) {
			t.Errorf("%s = %v, getIssue returns %v", key, body[key], singleBody[key])
		}
	}
}

func TestBatchGetIssuesSpecMatchesHandler(t *testing.T) {
	doc := loadSpec(t)
	op := specOps(t, doc)[OpBatchGetIssues].op
	if op == nil {
		t.Fatal("batchGetIssues is absent from the spec")
	}
	params := op["parameters"].([]any)
	gotNames := make([]string, 0, len(params))
	for _, raw := range params {
		param := raw.(map[string]any)
		gotNames = append(gotNames, param["name"].(string))
		if param["name"] == "issue_id" {
			schema := mapAt(t, param, "schema")
			if param["required"] != true || param["explode"] != true || schema["minItems"] != 1 || schema["maxItems"] != maxDependencyAnchors {
				t.Errorf("issue_id parameter = %+v, want required repeated IDs bounded 1..100", param)
			}
		}
	}
	if want := []string{"issue_id", "include_comments", "include_dependents", "brief_deps"}; !reflect.DeepEqual(gotNames, want) {
		t.Errorf("parameters = %v, want %v", gotNames, want)
	}
	getParams := specOps(t, doc)[OpGetIssue].op["parameters"].([]any)
	for i := 1; i < len(params); i++ {
		if !reflect.DeepEqual(params[i].(map[string]any)["schema"], getParams[i].(map[string]any)["schema"]) {
			t.Errorf("%s option schema differs from getIssue", gotNames[i])
		}
	}
	schemas := mapAt(t, mapAt(t, doc, "components"), "schemas")
	issue := mapAt(t, mapAt(t, schemas, "BatchIssueDetailItem"), "properties")["issue"].(map[string]any)
	ref := issue["allOf"].([]any)[0].(map[string]any)["$ref"]
	if ref != "#/components/schemas/IssueDetails" {
		t.Errorf("batch issue schema = %v, want getIssue's IssueDetails", ref)
	}
}

// batchDetailIssues answers GetDetailBatch with every ID absent, so the
// provider-backed batch route can run end to end without a database.
type batchDetailIssues struct{ recordingIssues }

func (*batchDetailIssues) GetDetailBatch(_ context.Context, req issueops.DetailBatchRequest) (issueops.DetailBatchResult, error) {
	items := make([]issueops.DetailBatchItem, 0, len(req.IDs))
	for _, id := range req.IDs {
		items = append(items, issueops.DetailBatchItem{ID: id})
	}
	return issueops.DetailBatchResult{Items: items}, nil
}

// TestBatchGetTimesTheUnitOfWorkItsReaderOpens is the batch twin of
// TestAReadRouteTimesTheUnitsOfWorkItsReaderOpens. A batch reader built on
// the untimed provider compiles and answers, but reports uow_ms=0.000.
func TestBatchGetTimesTheUnitOfWorkItsReaderOpens(t *testing.T) {
	provider := &fakeProvider{
		issues:     &fakeIssues{},
		readIssues: &batchDetailIssues{},
		readConfig: emptyConfig{},
		delay:      5 * time.Millisecond,
	}
	ts := newTestServer(t, Config{Provider: provider})

	if resp := ts.get(t, "/v0/beads/issues:batchGet?issue_id=bd-1"); resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", resp.StatusCode, readAll(t, resp))
	}
	if n := len(provider.openedUOWs()); n != 1 {
		t.Fatalf("opened %d units of work, want 1", n)
	}
	line := findLogLine(t, ts.stderr.String(), "op="+OpBatchGetIssues)
	if !strings.Contains(line, "uow_ms=") {
		t.Fatalf("batchGet request line has no uow_ms field:\n%s", line)
	}
	if strings.Contains(line, "uow_ms=0.000") {
		t.Errorf("batchGet request line reports no unit-of-work time though the provider took 5ms; the batch reader is bound to the untimed provider:\n%s", line)
	}
}
