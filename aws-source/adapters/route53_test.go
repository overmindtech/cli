package adapters

import (
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"testing"
)

func route53GetAutoConfig(t *testing.T) (*route53.Client, string, string) {
	config, account, region := GetAutoConfig(t)
	client := route53.NewFromConfig(config)

	return client, account, region
}
