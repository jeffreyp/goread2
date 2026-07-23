package services

import (
	"context"
	"fmt"
	"os"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
)

// CloudTasksQueue enqueues async work as Cloud Tasks targeting this App
// Engine service's own task handler endpoints. Tasks are dispatched via the
// AppEngineHttpRequest target, which App Engine authenticates the same way
// it authenticates cron: the X-AppEngine-QueueName header is stripped from
// any inbound request that didn't originate from Cloud Tasks' internal
// dispatch, so the task handlers can trust its presence without a separate
// signature check (see internal/auth.VerifyTaskRequest).
type CloudTasksQueue struct {
	client    *cloudtasks.Client
	projectID string
	location  string
	queueID   string
}

// NewCloudTasksQueue creates a queue client using application default
// credentials. The queue itself (projects/*/locations/*/queues/*) must
// already exist; this only creates the API client used to enqueue tasks
// into it.
func NewCloudTasksQueue(ctx context.Context) (*CloudTasksQueue, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT not set")
	}

	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating cloud tasks client: %w", err)
	}

	return &CloudTasksQueue{
		client:    client,
		projectID: projectID,
		location:  getEnvOrDefault("CLOUD_TASKS_LOCATION", "us-central1"),
		queueID:   getEnvOrDefault("CLOUD_TASKS_QUEUE", "cron-tasks"),
	}, nil
}

// Close releases the underlying gRPC connection.
func (q *CloudTasksQueue) Close() error {
	return q.client.Close()
}

// Enqueue creates a task that Cloud Tasks dispatches as a POST to
// relativeURI on this App Engine service's default target.
func (q *CloudTasksQueue) Enqueue(ctx context.Context, relativeURI string) error {
	_, err := q.client.CreateTask(ctx, &cloudtaskspb.CreateTaskRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s/queues/%s", q.projectID, q.location, q.queueID),
		Task: &cloudtaskspb.Task{
			MessageType: &cloudtaskspb.Task_AppEngineHttpRequest{
				AppEngineHttpRequest: &cloudtaskspb.AppEngineHttpRequest{
					HttpMethod:  cloudtaskspb.HttpMethod_POST,
					RelativeUri: relativeURI,
				},
			},
		},
	})
	return err
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
