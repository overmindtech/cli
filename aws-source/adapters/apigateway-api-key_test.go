package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestApiKeyOutputMapper(t *testing.T) {
	awsItem := &types.ApiKey{
		Id:              aws.String("api-key-id"),
		Name:            aws.String("api-key-name"),
		Enabled:         true,
		CreatedDate:     aws.Time(time.Now()),
		LastUpdatedDate: aws.Time(time.Now()),
		StageKeys:       []string{"rest-api-id/stage"},
		Tags:            map[string]string{"key": "value"},
	}

	item, err := apiKeyOutputMapper("scope", awsItem)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := item.Validate(); err != nil {
		t.Error(err)
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "apigateway-rest-api",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "rest-api-id",
			ExpectedScope:  "scope",
		},
	}

	tests.Execute(t, item)
}

func TestNewAPIGatewayApiKeyAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)

	client := apigateway.NewFromConfig(config)

	adapter := NewAPIGatewayApiKeyAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
