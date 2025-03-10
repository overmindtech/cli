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

func TestModelOutputMapper(t *testing.T) {
	awsItem := &types.Model{
		Id:          aws.String("model-id"),
		Name:        aws.String("model-name"),
		Description: aws.String("description"),
		Schema:      aws.String("{\"type\": \"object\"}"),
		ContentType: aws.String("application/json"),
	}

	item, err := modelOutputMapper("rest-api-id/model-name", "scope", awsItem)
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

func TestNewAPIGatewayModelAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)

	client := apigateway.NewFromConfig(config)

	adapter := NewAPIGatewayModelAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
