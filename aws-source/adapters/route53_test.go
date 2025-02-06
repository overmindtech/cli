package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func route53GetAutoConfig(t *testing.T) (*route53.Client, string, string) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := route53.NewFromConfig(config)

	return client, account, region
}
