package ironmq

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"path"
	"sync"
	"time"
)

// Copied straight from Go's package rand since it's not exported.
type lockedSource struct {
	lk  sync.Mutex
	src rand.Source
}

func (r *lockedSource) Int63() (n int64) {
	r.lk.Lock()
	n = r.src.Int63()
	r.lk.Unlock()
	return
}

func (r *lockedSource) Seed(seed int64) {
	r.lk.Lock()
	r.src.Seed(seed)
	r.lk.Unlock()
}

var localRand = rand.New(&lockedSource{src: rand.NewSource(time.Now().Unix())})

// A Client contains an Iron.io project ID and a token for authentication.
type Client struct {
	Debug     bool
	cloud     *Cloud
	projectId string
	token     string
}

// NewClient returns a new Client using the given project ID and token.
// The network is not used during this call.
func NewClient(projectId, token string, cloud *Cloud) *Client {
	return &Client{projectId: projectId, token: token, cloud: cloud}
}

type Error struct {
	Status int
	Msg    string
}

var EmptyQueue = errors.New("queue is empty")

func (e *Error) Error() string { return fmt.Sprintf("Status %d: %s", e.Status, e.Msg) }

func (c *Client) req(method, endpoint string, body []byte, data interface{}) error {
	const apiVersion = "1"
	url := path.Join(c.cloud.host, apiVersion, "projects", c.projectId, endpoint)
	url = c.cloud.scheme + "://" + url + "?oauth=" + c.token
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(body))
	}

	const maxRetries = 5
	tries := uint(0)
	eofCount := 0
	var resp *http.Response
	for tries < maxRetries {
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			if err == io.EOF && eofCount < 3 {
				eofCount++
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
				continue
			}
			return err
		}
		// ELB sometimes returns this when load is increasing; we retry
		// with exponential backoff
		if resp.StatusCode == http.StatusServiceUnavailable {
			tries++
			// random delay between 0 and (4^tries*100) milliseconds
			pow := int64(1) << (2 * tries) * 100
			delayMs := time.Duration(localRand.Int63n(pow))
			time.Sleep(delayMs * time.Millisecond)
			req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			continue
		}
		break
	}

	if c.Debug {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			fmt.Println("error dumping response:", err)
		} else {
			fmt.Printf("IRONMQ_GO_CLIENT:  %s\n", dump)
		}
	}

	decoder := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		data := map[string]interface{}{}
		decoder.Decode(&data)
		msg, _ := data["msg"].(string)
		return &Error{resp.StatusCode, msg}
	}

	if data != nil {
		err = decoder.Decode(data)
		if err != nil {
			return err
		}
	}
	return nil
}

// Queue represents an IronMQ queue.
type Queue struct {
	Name   string
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
func (q *Queue) Info() (*QueueInfo, error) {
	var qi QueueInfo
	err := q.Client.req("GET", "queues/"+q.Name, nil, &qi)
	if err != nil {
		return nil, err
	}
	return &qi, nil
}

// Get takes one Message off of the queue. The Message will be returned to the queue
// if not deleted before the item's timeout.
func (q *Queue) Get() (*Message, error) {
	var resp struct {
		Msgs []*Message `json:"messages"`
	}
	err := q.Client.req("GET", "queues/"+q.Name+"/messages", nil, &resp)
	if err != nil {
		return nil, err
	}
	if len(resp.Msgs) == 0 {
		return nil, EmptyQueue
	}
	msg := resp.Msgs[0]
	msg.q = q
	return msg, nil
}

// Push adds a message to the end of the queue using IronMQ's defaults:
//	timeout - 60 seconds
//	delay - none
func (q *Queue) Push(msg string) (id string, err error) {
	return q.PushMsg(&Message{Body: msg})
}

// PushMsg adds a message to the end of the queue using the fields of msg as
// parameters. msg.Id is ignored.
func (q *Queue) PushMsg(msg *Message) (id string, err error) {
	msgs := struct {
		Messages []*Message `json:"messages"`
	}{
		[]*Message{msg},
	}
	data, err := json.Marshal(msgs)
	if err != nil {
		return "", err
	}
	var resp struct {
		IDs []string `json:"ids"`
	}
	err = q.Client.req("POST", "queues/"+q.Name+"/messages", data, &resp)
	if err != nil {
		return "", err
	}
	return resp.IDs[0], nil
}

func (q *Queue) DeleteMsg(id string) error {
	return q.Client.req("DELETE", "queues/"+q.Name+"/messages/"+id, nil, nil)
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

func (m *Message) Delete() error {
	return m.q.DeleteMsg(m.Id)
}
