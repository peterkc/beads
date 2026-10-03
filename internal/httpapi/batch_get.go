package httpapi

import (
	"fmt"
	"net/http"

	"github.com/steveyegge/beads/internal/httpapi/apigen"
	"github.com/steveyegge/beads/issueops"
)

// handleBatchGetIssues answers GET /v0/beads/issues:batchGet. The role owns
// the snapshot, detail assembly, request order, repeats and per-item absence.
func (s *Server) handleBatchGetIssues(w http.ResponseWriter, r *http.Request) {
	q := newQuery(r.URL.Query())
	req := issueops.DetailBatchRequest{
		IDs:               q.list("issue_id"),
		IncludeComments:   q.boolean("include_comments"),
		IncludeDependents: q.boolean("include_dependents"),
		BriefDeps:         q.boolean("brief_deps"),
	}
	if !s.acceptQuery(w, r, q) {
		return
	}
	if len(req.IDs) == 0 {
		requestInfo(r.Context()).refuse("issue_id")
		s.fail(w, r, InvalidArgument("issue_id", ReasonInvalidValue, "name at least one issue_id"))
		return
	}
	if len(req.IDs) > maxDependencyAnchors {
		requestInfo(r.Context()).refuse("issue_id")
		s.fail(w, r, InvalidArgument("issue_id", ReasonInvalidValue,
			fmt.Sprintf("at most %d issue_id values per request, got %d", maxDependencyAnchors, len(req.IDs))))
		return
	}
	if err := issueops.ValidateDetailBatchRequest(req); err != nil {
		requestInfo(r.Context()).refuse("issue_id")
		s.fail(w, r, InvalidArgument("issue_id", ReasonInvalidValue, err.Error()))
		return
	}
	rd, err := s.detailBatchReader(r)
	if err != nil {
		s.failErr(w, r, err)
		return
	}
	result, err := rd.GetBatch(r.Context(), req)
	if err != nil {
		s.failReadErr(w, r, err)
		return
	}
	body := apigen.BatchIssueDetails{Items: make([]apigen.BatchIssueDetailItem, 0, len(result.Items))}
	for _, item := range result.Items {
		wire := apigen.BatchIssueDetailItem{IssueId: item.ID, Found: item.Found}
		if item.Found {
			wire.Issue = item.Issue
		}
		body.Items = append(body.Items, wire)
	}
	writeJSON(w, body)
}
