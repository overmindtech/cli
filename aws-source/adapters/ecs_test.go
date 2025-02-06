package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

type ecsTestClient struct{}

func ecsGetAutoConfig(t *testing.T) (*ecs.Client, string, string) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := ecs.NewFromConfig(config)

	return client, account, region
}
