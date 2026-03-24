package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Job represents a single async indexing job.
type Job struct {
	ID          string
	WorkspaceID string
	Status      string // queued, running, done, failed
	TotalFiles  int
	Done        int
	Failed      int
	ErrorMsg    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// JobStore provides CRUD operations for the index_jobs table.
type JobStore struct {
	pool *pgxpool.Pool
}

// NewJobStore returns a JobStore backed by the given pool.
func NewJobStore(pool *pgxpool.Pool) *JobStore {
	return &JobStore{pool: pool}
}

// Create inserts a new job and returns the generated ID.
func (s *JobStore) Create(ctx context.Context, workspaceID string, totalFiles int) (*Job, error) {
	row := s.pool.QueryRow(ctx,
		`INSERT INTO index_jobs (workspace_id, total_files)
		 VALUES ($1, $2)
		 RETURNING id, workspace_id, status, total_files, done, failed, error_msg, created_at, updated_at`,
		workspaceID, totalFiles,
	)
	j := &Job{}
	if err := row.Scan(&j.ID, &j.WorkspaceID, &j.Status, &j.TotalFiles, &j.Done, &j.Failed, &j.ErrorMsg, &j.CreatedAt, &j.UpdatedAt); err != nil {
		return nil, fmt.Errorf("jobs: create: %w", err)
	}
	return j, nil
}

// Get retrieves a job by ID.
func (s *JobStore) Get(ctx context.Context, jobID string) (*Job, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, workspace_id, status, total_files, done, failed, error_msg, created_at, updated_at
		 FROM index_jobs WHERE id = $1`,
		jobID,
	)
	j := &Job{}
	if err := row.Scan(&j.ID, &j.WorkspaceID, &j.Status, &j.TotalFiles, &j.Done, &j.Failed, &j.ErrorMsg, &j.CreatedAt, &j.UpdatedAt); err != nil {
		return nil, fmt.Errorf("jobs: get %s: %w", jobID, err)
	}
	return j, nil
}

// UpdateStatus sets the job status and optionally an error message.
func (s *JobStore) UpdateStatus(ctx context.Context, jobID, status, errorMsg string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE index_jobs SET status=$1, error_msg=$2, updated_at=NOW() WHERE id=$3`,
		status, errorMsg, jobID,
	)
	if err != nil {
		return fmt.Errorf("jobs: update status %s: %w", jobID, err)
	}
	return nil
}

// IncrementDone atomically increments the done counter for a job.
func (s *JobStore) IncrementDone(ctx context.Context, jobID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE index_jobs SET done = done + 1, updated_at = NOW() WHERE id = $1`,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("jobs: increment done %s: %w", jobID, err)
	}
	return nil
}

// IncrementFailed atomically increments the failed counter for a job.
func (s *JobStore) IncrementFailed(ctx context.Context, jobID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE index_jobs SET failed = failed + 1, updated_at = NOW() WHERE id = $1`,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("jobs: increment failed %s: %w", jobID, err)
	}
	return nil
}
