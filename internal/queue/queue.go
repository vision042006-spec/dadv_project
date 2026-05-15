package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Queue struct {
	client *redis.Client
	name   string
}

type JobPayload struct {
	JobID      string    `json:"job_id"`
	FilePath   string    `json:"file_path"`
	FileName   string    `json:"file_name"`
	FileSize   int64     `json:"file_size"`
	CreatedAt  time.Time `json:"created_at"`
	CallbackURL string   `json:"callback_url,omitempty"`
}

type JobStatus struct {
	Status      string    `json:"status"`
	Created     time.Time `json:"created"`
	Started     time.Time `json:"started,omitempty"`
	Completed   time.Time `json:"completed,omitempty"`
	FilePath    string    `json:"file_path,omitempty"`
	FileType    string    `json:"file_type,omitempty"`
	RowCount    int64     `json:"row_count,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
	ResultID    string    `json:"result_id,omitempty"`
}

func New(client *redis.Client, name string) *Queue {
	return &Queue{
		client: client,
		name:   name,
	}
}

func (q *Queue) Enqueue(ctx context.Context, payload *JobPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal job payload: %w", err)
	}

	if err := q.client.LPush(ctx, q.name, data).Err(); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	// Set initial status
	jobKey := fmt.Sprintf("job:%s", payload.JobID)
	status := JobStatus{
		Status:  "pending",
		Created: time.Now().UTC(),
	}
	statusData, _ := json.Marshal(status)
	if err := q.client.Set(ctx, jobKey, statusData, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to set job status: %w", err)
	}

	return nil
}

func (q *Queue) Dequeue(ctx context.Context, timeout time.Duration) (*JobPayload, error) {
	data, err := q.client.BRPop(ctx, timeout, q.name).Result()
	if err != nil {
		return nil, err
	}

	if len(data) < 2 {
		return nil, fmt.Errorf("invalid job data")
	}

	var payload JobPayload
	if err := json.Unmarshal([]byte(data[1]), &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job payload: %w", err)
	}

	return &payload, nil
}

func (q *Queue) GetStatus(ctx context.Context, jobID string) (*JobStatus, error) {
	jobKey := fmt.Sprintf("job:%s", jobID)
	data, err := q.client.Get(ctx, jobKey).Result()
	if err != nil {
		return nil, err
	}

	var status JobStatus
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job status: %w", err)
	}

	return &status, nil
}

func (q *Queue) UpdateStatus(ctx context.Context, jobID string, status *JobStatus) error {
	jobKey := fmt.Sprintf("job:%s", jobID)
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal job status: %w", err)
	}

	if err := q.client.Set(ctx, jobKey, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

func (q *Queue) SetJobProcessing(ctx context.Context, jobID string) error {
	jobKey := fmt.Sprintf("job:%s", jobID)
	status := JobStatus{
		Status:  "processing",
		Created: time.Now().UTC(),
		Started: time.Now().UTC(),
	}
	data, _ := json.Marshal(status)
	return q.client.Set(ctx, jobKey, data, 24*time.Hour).Err()
}

func (q *Queue) SetJobComplete(ctx context.Context, jobID string, rowCount int64) error {
	jobKey := fmt.Sprintf("job:%s", jobID)
	status := JobStatus{
		Status:    "completed",
		Completed: time.Now().UTC(),
		RowCount:  rowCount,
	}
	data, _ := json.Marshal(status)
	return q.client.Set(ctx, jobKey, data, 24*time.Hour).Err()
}

func (q *Queue) SetJobFailed(ctx context.Context, jobID, errMsg string) error {
	jobKey := fmt.Sprintf("job:%s", jobID)
	status := JobStatus{
		Status:      "failed",
		ErrorMessage: errMsg,
		Completed:   time.Now().UTC(),
	}
	data, _ := json.Marshal(status)
	return q.client.Set(ctx, jobKey, data, 24*time.Hour).Err()
}