package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// JobStore provides CRUD operations for the index_jobs table.
type JobStore struct {
	db *gorm.DB
}

// NewJobStore returns a JobStore backed by the given GORM DB.
func NewJobStore(db *gorm.DB) *JobStore {
	return &JobStore{db: db}
}

// Create inserts a new job and returns it with the generated ID.
func (s *JobStore) Create(ctx context.Context, workspaceID string, totalFiles int) (*Job, error) {
	j := &Job{
		WorkspaceID: workspaceID,
		Status:      "queued",
		TotalFiles:  totalFiles,
	}
	if err := s.db.WithContext(ctx).Create(j).Error; err != nil {
		return nil, fmt.Errorf("jobs: create: %w", err)
	}
	return j, nil
}

// Get retrieves a job by ID.
// Returns gorm.ErrRecordNotFound (wrapped) when the job does not exist.
func (s *JobStore) Get(ctx context.Context, jobID string) (*Job, error) {
	var j Job
	if err := s.db.WithContext(ctx).First(&j, "id = ?", jobID).Error; err != nil {
		return nil, fmt.Errorf("jobs: get %s: %w", jobID, err)
	}
	return &j, nil
}

// UpdateStatus sets the job status and optionally an error message.
func (s *JobStore) UpdateStatus(ctx context.Context, jobID, status, errorMsg string) error {
	err := s.db.WithContext(ctx).
		Model(&Job{}).
		Where("id = ?", jobID).
		Updates(map[string]any{"status": status, "error_msg": errorMsg}).
		Error
	if err != nil {
		return fmt.Errorf("jobs: update status %s: %w", jobID, err)
	}
	return nil
}

// IncrementDone atomically increments the done counter for a job.
func (s *JobStore) IncrementDone(ctx context.Context, jobID string) error {
	err := s.db.WithContext(ctx).
		Model(&Job{}).
		Where("id = ?", jobID).
		UpdateColumn("done", gorm.Expr("done + 1")).
		Error
	if err != nil {
		return fmt.Errorf("jobs: increment done %s: %w", jobID, err)
	}
	return nil
}

// IncrementFailed atomically increments the failed counter for a job.
func (s *JobStore) IncrementFailed(ctx context.Context, jobID string) error {
	err := s.db.WithContext(ctx).
		Model(&Job{}).
		Where("id = ?", jobID).
		UpdateColumn("failed", gorm.Expr("failed + 1")).
		Error
	if err != nil {
		return fmt.Errorf("jobs: increment failed %s: %w", jobID, err)
	}
	return nil
}
