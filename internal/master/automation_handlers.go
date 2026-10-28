package master

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chocom1nt/LANdapter/internal/common"
	"github.com/google/uuid"
)

const maxAutomationBody = 1 << 20

type createPolicyRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	OSFilter    string   `json:"os_filter"`
	Files       []string `json:"files"`
	Mode        string   `json:"mode"`
}

type createScheduleRequest struct {
	PolicyID        string `json:"policy_id"`
	Name            string `json:"name"`
	IntervalMinutes int    `json:"interval_minutes"`
	Enabled         bool   `json:"enabled"`
}

func (s *Server) handlePolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		policies, err := s.storage.ListPolicies(r.Context())
		if err != nil {
			s.logger.Error("List policies failed", "error", err)
			http.Error(w, "Unable to list group policies", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, policies)
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, maxAutomationBody)
		defer r.Body.Close()
		var req createPolicyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		req.Description = strings.TrimSpace(req.Description)
		req.OSFilter = strings.TrimSpace(req.OSFilter)
		if req.Name == "" || len([]rune(req.Name)) > 120 || len([]rune(req.Description)) > 500 || len([]rune(req.OSFilter)) > 100 {
			http.Error(w, "Name is required (max 120 characters); description max 500 and OS filter max 100 characters", http.StatusBadRequest)
			return
		}
		if len(req.Files) == 0 || len(req.Files) > 100 {
			http.Error(w, "files must contain between 1 and 100 package IDs", http.StatusBadRequest)
			return
		}
		if req.Mode == "" {
			req.Mode = "quiet"
		}
		if req.Mode != "quiet" && req.Mode != "normal" && req.Mode != "interactive" {
			http.Error(w, "Invalid install mode", http.StatusBadRequest)
			return
		}
		files := make([]string, 0, len(req.Files))
		seen := make(map[string]struct{}, len(req.Files))
		for _, raw := range req.Files {
			id, err := uuid.Parse(raw)
			if err != nil || id == uuid.Nil {
				http.Error(w, "Invalid file ID", http.StatusBadRequest)
				return
			}
			normalized := id.String()
			if _, ok := seen[normalized]; ok {
				continue
			}
			if _, err := findUploadedFile(normalized); err != nil {
				http.Error(w, "One or more packages were not found", http.StatusBadRequest)
				return
			}
			seen[normalized] = struct{}{}
			files = append(files, normalized)
		}
		if len(files) == 0 {
			http.Error(w, "At least one unique package is required", http.StatusBadRequest)
			return
		}
		now := time.Now().UTC()
		policy := &common.GroupPolicy{
			ID: uuid.New(), Name: req.Name, Description: req.Description, OSFilter: req.OSFilter,
			Files: common.JSONStringSlice(files), Mode: req.Mode, Enabled: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := s.storage.CreatePolicy(r.Context(), policy); err != nil {
			s.logger.Error("Create group policy failed", "error", err)
			http.Error(w, "Unable to create group policy", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, policy)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePolicyOperations(w http.ResponseWriter, r *http.Request) {
	rawID := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/")
	if strings.HasSuffix(rawID, "/run") {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id, err := parseAutomationID(strings.TrimSuffix(rawID, "/run"))
		if err != nil {
			http.Error(w, "Invalid policy ID", http.StatusBadRequest)
			return
		}
		job, err := s.dispatchPolicyToOnlineClients(r.Context(), id)
		if err != nil {
			s.logger.Warn("Manual policy dispatch failed", "policy", id, "error", err)
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"job_id": job.ID.String()})
		return
	}
	id, err := parseAutomationID(rawID)
	if err != nil {
		http.Error(w, "Invalid policy ID", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodDelete:
		policy, err := s.storage.GetPolicy(r.Context(), id)
		if err != nil {
			s.logger.Error("Policy lookup failed", "policy", id, "error", err)
			http.Error(w, "Unable to load group policy", http.StatusInternalServerError)
			return
		}
		if policy == nil {
			http.Error(w, "Group policy not found", http.StatusNotFound)
			return
		}
		if err := s.storage.DeletePolicy(r.Context(), id); err != nil {
			s.logger.Error("Delete group policy failed", "policy", id, "error", err)
			http.Error(w, "Unable to delete group policy", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSchedules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		schedules, err := s.storage.ListSchedules(r.Context())
		if err != nil {
			s.logger.Error("List schedules failed", "error", err)
			http.Error(w, "Unable to list schedules", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, schedules)
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, maxAutomationBody)
		defer r.Body.Close()
		var req createScheduleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		policyID, err := uuid.Parse(req.PolicyID)
		if err != nil || policyID == uuid.Nil {
			http.Error(w, "Invalid policy ID", http.StatusBadRequest)
			return
		}
		if req.Name == "" || len([]rune(req.Name)) > 120 {
			http.Error(w, "Schedule name is required (max 120 characters)", http.StatusBadRequest)
			return
		}
		if req.IntervalMinutes < 5 || req.IntervalMinutes > 43200 {
			http.Error(w, "interval_minutes must be between 5 and 43200", http.StatusBadRequest)
			return
		}
		policy, err := s.storage.GetPolicy(r.Context(), policyID)
		if err != nil {
			s.logger.Error("Schedule policy lookup failed", "policy", policyID, "error", err)
			http.Error(w, "Unable to validate policy", http.StatusInternalServerError)
			return
		}
		if policy == nil || !policy.Enabled {
			http.Error(w, "An enabled group policy is required", http.StatusBadRequest)
			return
		}
		now := time.Now().UTC()
		schedule := &common.Schedule{
			ID: uuid.New(), PolicyID: policyID, Name: req.Name, IntervalMinutes: req.IntervalMinutes,
			Enabled: req.Enabled, NextRun: now.Add(time.Duration(req.IntervalMinutes) * time.Minute),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.storage.CreateSchedule(r.Context(), schedule); err != nil {
			s.logger.Error("Create schedule failed", "error", err)
			http.Error(w, "Unable to create schedule", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, schedule)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleScheduleOperations(w http.ResponseWriter, r *http.Request) {
	id, err := parseAutomationID(strings.TrimPrefix(r.URL.Path, "/api/v1/schedules/"))
	if err != nil {
		http.Error(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodDelete:
		schedule, err := s.storage.GetSchedule(r.Context(), id)
		if err != nil {
			s.logger.Error("Schedule lookup failed", "schedule", id, "error", err)
			http.Error(w, "Unable to load schedule", http.StatusInternalServerError)
			return
		}
		if schedule == nil {
			http.Error(w, "Schedule not found", http.StatusNotFound)
			return
		}
		if err := s.storage.DeleteSchedule(r.Context(), id); err != nil {
			s.logger.Error("Delete schedule failed", "schedule", id, "error", err)
			http.Error(w, "Unable to delete schedule", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodPatch:
		schedule, lookupErr := s.storage.GetSchedule(r.Context(), id)
		if lookupErr != nil {
			s.logger.Error("Schedule lookup failed", "schedule", id, "error", lookupErr)
			http.Error(w, "Unable to load schedule", http.StatusInternalServerError)
			return
		}
		if schedule == nil {
			http.Error(w, "Schedule not found", http.StatusNotFound)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		defer r.Body.Close()
		var req struct {
			Enabled *bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Enabled == nil {
			http.Error(w, "Expected an enabled boolean", http.StatusBadRequest)
			return
		}
		if err := s.storage.UpdateScheduleEnabled(r.Context(), id, *req.Enabled); err != nil {
			s.logger.Error("Update schedule failed", "schedule", id, "error", err)
			http.Error(w, "Unable to update schedule", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "DELETE, PATCH")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func parseAutomationID(raw string) (uuid.UUID, error) {
	if strings.Contains(raw, "/") || strings.TrimSpace(raw) == "" {
		return uuid.Nil, errors.New("invalid id")
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, errors.New("invalid id")
	}
	return id, nil
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		// Headers may already be committed; this is intentionally only a server-side log.
		_ = err
	}
}

func (s *Server) schedulerLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.schedulerStop:
			return
		case <-ticker.C:
			s.processDueSchedules()
		}
	}
}

func (s *Server) processDueSchedules() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	now := time.Now().UTC()
	schedules, err := s.storage.ListDueSchedules(ctx, now)
	if err != nil {
		s.logger.Error("Scheduler could not list due schedules", "error", err)
		return
	}
	for _, schedule := range schedules {
		next := now.Add(time.Duration(schedule.IntervalMinutes) * time.Minute)
		if err := s.storage.MarkScheduleRun(ctx, schedule.ID, now, next); err != nil {
			s.logger.Error("Could not claim scheduled run", "schedule", schedule.ID, "error", err)
			continue
		}
		if err := s.dispatchScheduledPolicy(ctx, schedule); err != nil {
			s.logger.Error("Scheduled policy dispatch failed", "schedule", schedule.ID, "policy", schedule.PolicyID, "error", err)
		}
	}
}

func (s *Server) dispatchScheduledPolicy(ctx context.Context, schedule common.Schedule) error {
	job, err := s.dispatchPolicyToOnlineClients(ctx, schedule.PolicyID)
	if err != nil {
		return err
	}
	s.logger.Info("Scheduled policy dispatched", "schedule", schedule.ID, "policy", schedule.PolicyID, "job", job.ID)
	return nil
}

func (s *Server) dispatchPolicyToOnlineClients(ctx context.Context, policyID uuid.UUID) (*common.Job, error) {
	policy, err := s.storage.GetPolicy(ctx, policyID)
	if err != nil {
		return nil, err
	}
	if policy == nil || !policy.Enabled {
		return nil, fmt.Errorf("group policy is missing or disabled")
	}
	if len(policy.Files) == 0 {
		return nil, fmt.Errorf("group policy has no packages")
	}
	for _, fileID := range policy.Files {
		if _, err := findUploadedFile(fileID); err != nil {
			return nil, fmt.Errorf("package %s is unavailable", fileID)
		}
	}
	clients, err := s.storage.ListClients(ctx, nil)
	if err != nil {
		return nil, err
	}
	filter := strings.ToLower(strings.TrimSpace(policy.OSFilter))
	targets := make([]common.Client, 0, len(clients))
	for _, client := range clients {
		if !client.Online {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(client.OS), filter) {
			continue
		}
		targets = append(targets, client)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no online clients matched OS filter %q", policy.OSFilter)
	}
	now := time.Now().UTC()
	job := &common.Job{ID: uuid.New(), Files: common.JSONStringSlice(policy.Files), CreatedAt: now}
	if err := s.storage.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]interface{}{"files": []string(policy.Files), "mode": policy.Mode})
	if err != nil {
		return nil, err
	}
	dispatched := 0
	for _, client := range targets {
		result := &common.JobResult{ID: uuid.New(), JobID: job.ID, ClientID: client.ID, Status: "pending"}
		if err := s.storage.CreateJobResult(ctx, result); err != nil {
			s.logger.Error("Could not create policy job result", "client", client.ID, "job", job.ID, "error", err)
			continue
		}
		started := time.Now().UTC()
		result.Status, result.StartedAt = "running", &started
		if err := s.storage.UpdateJobResult(ctx, result); err != nil {
			s.logger.Error("Could not mark policy job running", "client", client.ID, "job", job.ID, "error", err)
			continue
		}
		cmd := &commandMsg{Type: "install", JobID: job.ID.String(), Payload: payload}
		if err := s.sendCommand(client.ID, cmd); err != nil {
			finished := time.Now().UTC()
			message := "failed to queue policy command: " + err.Error()
			result.Status, result.Error, result.FinishedAt = "failed", &message, &finished
			if updateErr := s.storage.UpdateJobResult(ctx, result); updateErr != nil {
				s.logger.Warn("Could not record policy dispatch failure", "client", client.ID, "error", updateErr)
			}
			continue
		}
		dispatched++
	}
	if dispatched == 0 {
		return job, fmt.Errorf("no policy commands could be queued; check Master logs and client connections")
	}
	s.logger.Info("Group policy dispatched", "policy", policy.ID, "job", job.ID, "targets", dispatched)
	return job, nil
}
