package adapters

import (
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"testing"
)

func rdsGetAutoConfig(t *testing.T) (*rds.Client, string, string) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := rds.NewFromConfig(config)

	return client, account, region
}
