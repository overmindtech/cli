package adapters

import (
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"testing"
)

func efsGetAutoConfig(t *testing.T) (*efs.Client, string, string) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := efs.NewFromConfig(config)

	return client, account, region
}
