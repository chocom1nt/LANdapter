package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/chocom1nt/LANdapter/internal/common"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	db *sqlx.DB
}

func NewPostgresStorage(cfg common.DBConfig) (*PostgresStorage, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return &PostgresStorage{db: db}, nil
}

func NewPostgresStorageWithDB(db *sqlx.DB) *PostgresStorage {
    return &PostgresStorage{db: db}
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// ---- Clients ----

func (s *PostgresStorage) UpsertClient(ctx context.Context, client *common.Client) error {
	query := `INSERT INTO clients (id, hostname, os, mac, online, last_seen)
	          VALUES ($1, $2, $3, $4, $5, $6)
	          ON CONFLICT (id) DO UPDATE SET
	              hostname = EXCLUDED.hostname,
	              os = EXCLUDED.os,
	              mac = EXCLUDED.mac,
	              online = EXCLUDED.online,
	              last_seen = EXCLUDED.last_seen`
	_, err := s.db.ExecContext(ctx, query,
		client.ID, client.Hostname, client.OS, client.MAC, client.Online, client.LastSeen)
	return err
}

func (s *PostgresStorage) UpdateClientOnline(ctx context.Context, id uuid.UUID, online bool) error {
	query := `UPDATE clients SET online = $1, last_seen = NOW() WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, online, id)
	return err
}

func (s *PostgresStorage) GetClient(ctx context.Context, id uuid.UUID) (*common.Client, error) {
	var client common.Client
	query := `SELECT id, hostname, os, mac, online, last_seen FROM clients WHERE id = $1`
	err := s.db.GetContext(ctx, &client, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &client, err
}

func (s *PostgresStorage) ListClients(ctx context.Context, online *bool) ([]common.Client, error) {
	var clients []common.Client
	query := `SELECT id, hostname, os, mac, online, last_seen FROM clients`
	args := []interface{}{}
	if online != nil {
		query += ` WHERE online = $1`
		args = append(args, *online)
	}
	err := s.db.SelectContext(ctx, &clients, query, args...)
	return clients, err
}

// ---- Jobs ----

func (s *PostgresStorage) CreateJob(ctx context.Context, job *common.Job) error {
	query := `INSERT INTO jobs (id, files, created_at) VALUES ($1, $2, $3)`
	_, err := s.db.ExecContext(ctx, query, job.ID, job.Files, job.CreatedAt)
	return err
}

func (s *PostgresStorage) GetJob(ctx context.Context, id uuid.UUID) (*common.Job, error) {
	var job common.Job
	query := `SELECT id, files, created_at FROM jobs WHERE id = $1`
	err := s.db.GetContext(ctx, &job, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &job, err
}

// ---- JobResults ----

func (s *PostgresStorage) CreateJobResult(ctx context.Context, result *common.JobResult) error {
	query := `INSERT INTO job_results (id, job_id, client_id, status, output, error, started_at, finished_at, snapshot_before, snapshot_after)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := s.db.ExecContext(ctx, query, result.ID, result.JobID, result.ClientID,
		result.Status, result.Output, result.Error, result.StartedAt, result.FinishedAt,
		result.SnapshotBefore, result.SnapshotAfter)
	return err
}

func (s *PostgresStorage) UpdateJobResult(ctx context.Context, result *common.JobResult) error {
	query := `UPDATE job_results SET status = $1, output = $2, error = $3, started_at = $4, finished_at = $5,
	          snapshot_before = $6, snapshot_after = $7 WHERE id = $8`
	_, err := s.db.ExecContext(ctx, query, result.Status, result.Output, result.Error,
		result.StartedAt, result.FinishedAt, result.SnapshotBefore, result.SnapshotAfter, result.ID)
	return err
}

func (s *PostgresStorage) GetJobResult(ctx context.Context, jobID, clientID uuid.UUID) (*common.JobResult, error) {
	var res common.JobResult
	query := `SELECT id, job_id, client_id, status, output, error, started_at, finished_at, snapshot_before, snapshot_after
	          FROM job_results WHERE job_id = $1 AND client_id = $2`
	err := s.db.GetContext(ctx, &res, query, jobID, clientID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &res, err
}

func (s *PostgresStorage) ListJobResultsByJob(ctx context.Context, jobID uuid.UUID) ([]common.JobResult, error) {
	var results []common.JobResult
	query := `SELECT id, job_id, client_id, status, output, error, started_at, finished_at, snapshot_before, snapshot_after
	          FROM job_results WHERE job_id = $1`
	err := s.db.SelectContext(ctx, &results, query, jobID)
	return results, err
}