package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func efsGetAutoConfig(t *testing.T) (*efs.Client, string, string) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := efs.NewFromConfig(config)

	return client, account, region
}
