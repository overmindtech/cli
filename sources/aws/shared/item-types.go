package shared

import "github.com/overmindtech/cli/sources/shared"

var (
	KinesisStream         = shared.NewItemType(AWS, Kinesis, Stream)
	KinesisStreamConsumer = shared.NewItemType(AWS, Kinesis, StreamConsumer)
	IAMRole               = shared.NewItemType(AWS, IAM, Role)
)
