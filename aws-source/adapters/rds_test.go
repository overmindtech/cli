package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func rdsGetAutoConfig(t *testing.T) (*rds.Client, string, string) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := rds.NewFromConfig(config)

	return client, account, region
}
