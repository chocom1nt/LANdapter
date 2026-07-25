package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chocom1nt/LANdapter/internal/common"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestClients(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	store := &PostgresStorage{db: sqlxDB}
	ctx := context.Background()
	clientID := uuid.New()

	t.Run("UpsertClient", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO clients`).
			WithArgs(clientID, "host1", "windows", "00:11:22:33:44:55", true, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		client := &common.Client{
			ID:       clientID,
			Hostname: "host1",
			OS:       "windows",
			MAC:      "00:11:22:33:44:55",
			Online:   true,
			LastSeen: time.Now(),
		}
		err = store.UpsertClient(ctx, client)
		if err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("GetClient", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "hostname", "os", "mac", "online", "last_seen"}).
			AddRow(clientID, "host1", "windows", "00:11:22:33:44:55", true, time.Now())
		mock.ExpectQuery(`SELECT .+ FROM clients`).WithArgs(clientID).WillReturnRows(rows)

		client, err := store.GetClient(ctx, clientID)
		if err != nil {
			t.Fatal(err)
		}
		if client == nil || client.ID != clientID {
			t.Error("expected client, got nil or wrong ID")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("UpdateClientOnline", func(t *testing.T) {
		mock.ExpectExec(`UPDATE clients SET online = \$1, last_seen = NOW\(\) WHERE id = \$2`).
			WithArgs(false, clientID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err = store.UpdateClientOnline(ctx, clientID, false)
		if err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("ListClients", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "hostname", "os", "mac", "online", "last_seen"}).
			AddRow(clientID, "host1", "windows", "00:11:22:33:44:55", true, time.Now())
		mock.ExpectQuery(`SELECT .+ FROM clients`).WillReturnRows(rows)

		clients, err := store.ListClients(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(clients) != 1 {
			t.Errorf("expected 1 client, got %d", len(clients))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
}

func TestJobs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	store := &PostgresStorage{db: sqlxDB}
	ctx := context.Background()
	jobID := uuid.New()
	files := common.JSONStringSlice{"file1", "file2"}

	t.Run("CreateJob", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO jobs`).
			WithArgs(jobID, files, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		job := &common.Job{
			ID:        jobID,
			Files:     files,
			CreatedAt: time.Now(),
		}
		err = store.CreateJob(ctx, job)
		if err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("GetJob", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "files", "created_at"}).
			AddRow(jobID, files, time.Now())
		mock.ExpectQuery(`SELECT .+ FROM jobs`).WithArgs(jobID).WillReturnRows(rows)

		job, err := store.GetJob(ctx, jobID)
		if err != nil {
			t.Fatal(err)
		}
		if job == nil || job.ID != jobID {
			t.Error("job not found")
		}
	})
}

func TestJobResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	store := &PostgresStorage{db: sqlxDB}
	ctx := context.Background()
	jobID := uuid.New()
	clientID := uuid.New()
	resultID := uuid.New()
	now := time.Now()
	snapshot := json.RawMessage(`{"cpu": 12}`)

	t.Run("CreateJobResult", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO job_results`).
			WithArgs(resultID, jobID, clientID, "pending", nil, nil, nil, nil, nil, nil).
			WillReturnResult(sqlmock.NewResult(1, 1))

		res := &common.JobResult{
			ID:       resultID,
			JobID:    jobID,
			ClientID: clientID,
			Status:   "pending",
		}
		err = store.CreateJobResult(ctx, res)
		if err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("UpdateJobResult", func(t *testing.T) {
		mock.ExpectExec(`UPDATE job_results`).
			WithArgs("success", "output", nil, now, now, snapshot, snapshot, resultID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		rawSnapshot := json.RawMessage(snapshot)
		res := &common.JobResult{
			ID:             resultID,
			Status:         "success",
			Output:         strPtr("output"),
			StartedAt:      &now,
			FinishedAt:     &now,
			SnapshotBefore: &rawSnapshot,
			SnapshotAfter:  &rawSnapshot,
		}
		err = store.UpdateJobResult(ctx, res)
		if err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("GetJobResult", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "job_id", "client_id", "status", "output", "error", "started_at", "finished_at", "snapshot_before", "snapshot_after"}).
			AddRow(resultID, jobID, clientID, "success", "output", nil, now, now, snapshot, snapshot)
		mock.ExpectQuery(`SELECT .+ FROM job_results`).WithArgs(jobID, clientID).WillReturnRows(rows)

		res, err := store.GetJobResult(ctx, jobID, clientID)
		if err != nil {
			t.Fatal(err)
		}
		if res == nil || res.ID != resultID {
			t.Error("result not found")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("ListJobResultsByJob", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "job_id", "client_id", "status", "output", "error", "started_at", "finished_at", "snapshot_before", "snapshot_after"}).
			AddRow(resultID, jobID, clientID, "success", "output", nil, now, now, snapshot, snapshot)
		mock.ExpectQuery(`SELECT .+ FROM job_results`).WithArgs(jobID).WillReturnRows(rows)

		results, err := store.ListJobResultsByJob(ctx, jobID)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}
	})
}

func strPtr(s string) *string { return &s }