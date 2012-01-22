IronMQ Go Client
----------------

Getting Started
===============

Install the package:

    goinstall github.com/iron-io/iron_mq_go

The API is documented [here](http://iron-io.github.com/iron_mq_go/).

The Basics
==========
**Initialize** a client and get a queue object:

    client := ironmq.NewClient("my project", "my token")
    queue := client.Queue("my_queue")

**Push** a message on the queue:

    id, err := queue.Push("Hello, world!")

**Pop** a message off the queue:

    msg, err := queue.Get()

When you pop/get a message from the queue, it will *not* be deleted. It will
eventually go back onto the queue after a timeout if you don't delete it. (The
default timeout is 10 minutes.)

**Delete** a message from the queue:

    err := msg.Delete()
