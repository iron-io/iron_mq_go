package ironmq

type Cloud struct {
	scheme string
	host   string
}

func NewCloud(scheme, host string) *Cloud {
	return &Cloud{scheme, host}
}

var (
	IronAWSUSEast    = NewCloud("https", "mq-aws-us-east-1.iron.io")
	IronRackspaceDFW = NewCloud("https", "mq-rackspace-dfw.iron.io")
)
