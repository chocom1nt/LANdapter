package master

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chocom1nt/LANdapter/internal/common"
)

func TestCreatePolicyRejectsMissingPackageIDs(t *testing.T) {
	srv := NewServer(&common.MasterConfig{}, common.InitLogger(0), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies", strings.NewReader(`{"name":"Baseline","files":[]}`))
	resp := httptest.NewRecorder()
	srv.handlePolicies(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400: %s", resp.Code, resp.Body.String())
	}
}

func TestCreateScheduleRejectsTooFrequentInterval(t *testing.T) {
	srv := NewServer(&common.MasterConfig{}, common.InitLogger(0), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", strings.NewReader(`{"name":"Too frequent","policy_id":"00000000-0000-4000-8000-000000000001","interval_minutes":1,"enabled":true}`))
	resp := httptest.NewRecorder()
	srv.handleSchedules(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400: %s", resp.Code, resp.Body.String())
	}
}

func TestRunPolicyRequiresValidPolicyID(t *testing.T) {
	srv := NewServer(&common.MasterConfig{}, common.InitLogger(0), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/not-a-uuid/run", nil)
	resp := httptest.NewRecorder()
	srv.handlePolicyOperations(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400: %s", resp.Code, resp.Body.String())
	}
}

func TestRunPolicyRequiresPost(t *testing.T) {
	srv := NewServer(&common.MasterConfig{}, common.InitLogger(0), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/policies/00000000-0000-4000-8000-000000000001/run", nil)
	resp := httptest.NewRecorder()
	srv.handlePolicyOperations(resp, req)
	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got status %d, want 405: %s", resp.Code, resp.Body.String())
	}
}
