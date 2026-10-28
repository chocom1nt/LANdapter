package storage

import (
	"context"
	"time"

	"github.com/chocom1nt/LANdapter/internal/common"

	"github.com/google/uuid"
)

type Storage interface {
	// Clients
	UpsertClient(ctx context.Context, client *common.Client) error
	UpdateClientOnline(ctx context.Context, id uuid.UUID, online bool) error
	GetClient(ctx context.Context, id uuid.UUID) (*common.Client, error)
	ListClients(ctx context.Context, online *bool) ([]common.Client, error)
	SetClientTokenHash(ctx context.Context, id uuid.UUID, tokenHash string) error
	GetClientTokenHash(ctx context.Context, id uuid.UUID) (string, error)

	// Jobs
	CreateJob(ctx context.Context, job *common.Job) error
	GetJob(ctx context.Context, id uuid.UUID) (*common.Job, error)

	// JobResults
	CreateJobResult(ctx context.Context, result *common.JobResult) error
	UpdateJobResult(ctx context.Context, result *common.JobResult) error
	GetJobResult(ctx context.Context, jobID, clientID uuid.UUID) (*common.JobResult, error)
	ListJobResultsByJob(ctx context.Context, jobID uuid.UUID) ([]common.JobResult, error)

	// Group policies and interval schedules
	CreatePolicy(ctx context.Context, policy *common.GroupPolicy) error
	ListPolicies(ctx context.Context) ([]common.GroupPolicy, error)
	GetPolicy(ctx context.Context, id uuid.UUID) (*common.GroupPolicy, error)
	DeletePolicy(ctx context.Context, id uuid.UUID) error
	CreateSchedule(ctx context.Context, schedule *common.Schedule) error
	ListSchedules(ctx context.Context) ([]common.Schedule, error)
	GetSchedule(ctx context.Context, id uuid.UUID) (*common.Schedule, error)
	DeleteSchedule(ctx context.Context, id uuid.UUID) error
	UpdateScheduleEnabled(ctx context.Context, id uuid.UUID, enabled bool) error
	ListDueSchedules(ctx context.Context, now time.Time) ([]common.Schedule, error)
	MarkScheduleRun(ctx context.Context, id uuid.UUID, ranAt, nextRun time.Time) error
}
