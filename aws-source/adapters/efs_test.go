package adapters

import (
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"testing"
)

func efsGetAutoConfig(t *testing.T) (*efs.Client, string, string) {
	config, account, region := GetAutoConfig(t)
	client := efs.NewFromConfig(config)

	return client, account, region
}
