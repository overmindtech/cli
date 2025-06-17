package shared

import (
	"github.com/overmindtech/cli/sources/shared"
)

const (
	AWS shared.Source = "aws"
)

// APIs
const (
	APIGateway shared.API = "api-gateway"
	WAFv2      shared.API = "wafv2"
	Kinesis    shared.API = "kinesis"
	IAM        shared.API = "iam"
)

// Resources
const (
	APIKey         shared.Resource = "api-key"
	Stage          shared.Resource = "stage"
	RESTAPI        shared.Resource = "rest-api"
	Deployment     shared.Resource = "deployment"
	WebACL         shared.Resource = "web-acl"
	Stream         shared.Resource = "stream"
	StreamConsumer shared.Resource = "stream-consumer"
	Role           shared.Resource = "role"
)
