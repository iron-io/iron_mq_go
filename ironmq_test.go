package ironmq

import (
	"os"
	"testing"
)

func TestFunctionality(t *testing.T) {
	projectId := os.Getenv("IRONIO_PROJECT_ID")
	if projectId == "" {
		t.Fatalf("IRONIO_PROJECT_ID environment variable not set")
	}
	token := os.Getenv("IRONIO_TOKEN")
	if token == "" {
		t.Fatalf("IRONIO_TOKEN environment variable not set")
	}

	client := NewClient(projectId, token)
	queue := client.Queue("test-queue")

	// clear out the queue
	var err error
	for err == nil {
		_, err = queue.Get()
	}
	if err != EmptyQueue {
		t.Fatalf("queue.Get: expected empty queue error, got: %s", err)
	}

	const body = "Hello, IronMQ!"
	id, err := queue.Push(body)
	if err != nil {
		t.Fatalf("queue.Push: error isn't nil: %s", err)
	}
	if len(id) == 0 {
		t.Fatal("queue.Push: no ID returned")
	}

	qi, err := queue.Info()
	if err != nil {
		t.Fatalf("queue.Info: error isn't nil: %s", err)
	}
	if qi.Size != 1 {
		t.Errorf("queue.Info: size isn't 1: %v", qi.Size)
	}

	msg, err := queue.Get()
	if err != nil {
		t.Fatalf("queue.Get: error isn't nil: %s", err)
	}
	if msg == nil {
		t.Fatal("queue.Get: msg is nil")
	}
	if msg.Body != body {
		t.Fatalf("queue.Get: msg has wrong contents: %+v", msg)
	}

	err = msg.Delete()
	if err != nil {
		t.Fatalf("msg.Delete: error isn't nil: %s", err)
	}
}
