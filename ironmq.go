package ironmq

import (
	"bytes"
	"fmt"
	"http"
	"json"
	"os"
	"path"
)

// A Client contains an Iron.io project ID and a token for authentication.
type Client struct {
	Debug     bool
	projectId string
	token     string
}

// NewClient returns a new Client using the given project ID and token.
// The network is not used during this call.
func NewClient(projectId, token string) *Client {
	return &Client{projectId: projectId, token: token}
}

type Error struct {
	Status int
	Msg    string
}

var EmptyQueue = os.NewError("queue is empty")

func (e *Error) String() string { return fmt.Sprintf("Status %d: %s", e.Status, e.Msg) }

func (c *Client) req(method, endpoint string, body []byte) (map[string]interface{}, os.Error) {
	const host = "mq-aws-us-east-1.iron.io"
	const apiVersion = "1"
	url := path.Join(host, apiVersion, "projects", c.projectId, endpoint)
	url = "http://" + url + "?oauth=" + c.token
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(body))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if c.Debug {
		dump, err := http.DumpResponse(resp, true)
		if err != nil {
			fmt.Println("error dumping response:", err)
		} else {
			fmt.Printf("%s\n", dump)
		}
	}

	jDecoder := json.NewDecoder(resp.Body)
	data := map[string]interface{}{}
	err = jDecoder.Decode(&data)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		msg, _ := data["msg"].(string)
		return nil, &Error{resp.StatusCode, msg}
	}
	return data, nil
}

// Queue represents an IronMQ queue.
type Queue struct {
	name   string
	Client *Client
}

// Queue returns a Queue using the given name.
// The network is not used during this call.
func (c *Client) Queue(name string) *Queue {
	return &Queue{name, c}
}

// QueueInfo provides general information about a queue.
type QueueInfo struct {
	Size int // number of items available on the queue
}

// Info retrieves a QueueInfo structure for the queue.
func (q *Queue) Info() (*QueueInfo, os.Error) {
	var qi QueueInfo
	resp, err := q.Client.req("GET", "queues/"+q.name, nil)
	if err != nil {
		return nil, err
	}
	size, ok := resp["size"].(float64)
	if !ok {
		return nil, os.NewError("no queue size")
	}
	qi.Size = int(size)
	return &qi, nil
}

// Get takes one Message off of the queue. The Message will be returned to the queue
// if not deleted before the item's timeout.
func (q *Queue) Get() (*Message, os.Error) {
	resp, err := q.Client.req("GET", "queues/"+q.name+"/messages", nil)
	if err != nil {
		return nil, err
	}
	body, ok := resp["body"].(string)
	if !ok {
		return nil, EmptyQueue
	}
	id, ok := resp["id"].(string)
	if !ok {
		return nil, os.NewError("ID is not a string")
	}
	return &Message{Id: id, Body: body, q: q}, nil
}

// Push adds a message to the end of the queue using IronMQ's defaults:
//	timeout - 60 seconds
//	delay - none
func (q *Queue) Push(msg string) os.Error {
	return q.PushMsg(&Message{Body: msg})
}

// PushMsg adds a message to the end of the queue using the fields of msg as
// parameters. msg.Id is ignored.
func (q *Queue) PushMsg(msg *Message) os.Error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = q.Client.req("POST", "queues/"+q.name+"/messages", data)
	if err != nil {
		return err
	}
	return nil
}

type Message struct {
	Id   string `json:"id,omitempty"`
	Body string `json:"body"`
	// Timeout is the amount of time in seconds allowed for processing the
	// message.
	Timeout int64 `json:"timeout,omitempty"`
	// Delay is the amount of time in seconds to wait before adding the
	// message to the queue.
	Delay int64 `json:"delay,omitempty"`
	q     *Queue
}

func (m *Message) Delete() os.Error {
	_, err := m.q.Client.req("DELETE", "queues/"+m.q.name+"/messages/"+m.Id, nil)
	return err
}
