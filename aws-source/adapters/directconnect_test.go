package adapters

import (
	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"testing"
)

func directconnectGetAutoConfig(t *testing.T) (*directconnect.Client, string, string) {
	config, account, region := GetAutoConfig(t)
	client := directconnect.NewFromConfig(config)

	return client, account, region
}
