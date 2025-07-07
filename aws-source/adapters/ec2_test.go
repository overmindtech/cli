package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func ec2GetAutoConfig(t *testing.T) (*ec2.Client, string, string) {
	t.Helper()

	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := ec2.NewFromConfig(config)

	return client, account, region
}
