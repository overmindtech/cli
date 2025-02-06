package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func directconnectGetAutoConfig(t *testing.T) (*directconnect.Client, string, string) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := directconnect.NewFromConfig(config)

	return client, account, region
}
