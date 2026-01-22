package adapters

import (
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"testing"
)

func rdsGetAutoConfig(t *testing.T) (*rds.Client, string, string) {
	config, account, region := GetAutoConfig(t)
	client := rds.NewFromConfig(config)

	return client, account, region
}
