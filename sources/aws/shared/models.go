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
)

// Resources
const (
	APIKey     shared.Resource = "api-key"
	Stage      shared.Resource = "stage"
	RESTAPI    shared.Resource = "rest-api"
	Deployment shared.Resource = "deployment"
	WebACL     shared.Resource = "web-acl"
)
