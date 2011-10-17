package ironmq

import (
	"os"
	"testing"
)

func TestFunctionality(t *testing.T) {
	projectId := os.Getenv("IRONIO_PROJECT_ID")
	token := os.Getenv("IRONIO_TOKEN")

	client := NewClient(projectId, token)
	queue := client.Queue("test-queue")

	// clear out the queue
	var err os.Error
	for err == nil {
		_, err = queue.Get()
	}

	const body = "Hello, IronMQ!"
	err = queue.Push(body)
	if err != nil {
		t.Fatalf("queue.Push: error isn't nil: %s", err)
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
