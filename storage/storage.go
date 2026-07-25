package storage

import (
	"context"

	"github.com/chocom1nt/LANdapter/internal/common"

	"github.com/google/uuid"
)

type Storage interface {
    // Clients
    UpsertClient(ctx context.Context, client *common.Client) error
    UpdateClientOnline(ctx context.Context, id uuid.UUID, online bool) error
    GetClient(ctx context.Context, id uuid.UUID) (*common.Client, error)
    ListClients(ctx context.Context, online *bool) ([]common.Client, error)

    // Jobs
    CreateJob(ctx context.Context, job *common.Job) error
    GetJob(ctx context.Context, id uuid.UUID) (*common.Job, error)

    // JobResults
    CreateJobResult(ctx context.Context, result *common.JobResult) error
    UpdateJobResult(ctx context.Context, result *common.JobResult) error
    GetJobResult(ctx context.Context, jobID, clientID uuid.UUID) (*common.JobResult, error)
    ListJobResultsByJob(ctx context.Context, jobID uuid.UUID) ([]common.JobResult, error)
}