package services

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCreateTaskRequestRoutesToEnqueuingVersion(t *testing.T) {
	t.Setenv("GAE_SERVICE", "default")
	t.Setenv("GAE_VERSION", "staging-a1b2c3d")

	q := &CloudTasksQueue{
		projectID: "proj",
		location:  "us-central1",
		queueID:   "cron-tasks",
		routing:   appEngineRoutingFromEnv(),
	}
	schedule := timestamppb.New(time.Unix(1700000000, 0))

	req := q.createTaskRequest("/tasks/delete-account?user_id=7", schedule)

	if want := "projects/proj/locations/us-central1/queues/cron-tasks"; req.Parent != want {
		t.Errorf("Parent = %q, want %q", req.Parent, want)
	}
	if req.Task.ScheduleTime != schedule {
		t.Errorf("ScheduleTime = %v, want %v", req.Task.ScheduleTime, schedule)
	}
	httpReq := req.Task.GetAppEngineHttpRequest()
	if httpReq == nil {
		t.Fatal("expected an AppEngineHttpRequest target")
	}
	if httpReq.RelativeUri != "/tasks/delete-account?user_id=7" {
		t.Errorf("RelativeUri = %q", httpReq.RelativeUri)
	}
	routing := httpReq.GetAppEngineRouting()
	if routing.GetService() != "default" || routing.GetVersion() != "staging-a1b2c3d" {
		t.Errorf("AppEngineRouting = %v, want service=default version=staging-a1b2c3d", routing)
	}
}

func TestAppEngineRoutingFromEnvOutsideAppEngine(t *testing.T) {
	t.Setenv("GAE_SERVICE", "")
	t.Setenv("GAE_VERSION", "")

	if routing := appEngineRoutingFromEnv(); routing != nil {
		t.Errorf("expected nil routing outside App Engine, got %v", routing)
	}
}
