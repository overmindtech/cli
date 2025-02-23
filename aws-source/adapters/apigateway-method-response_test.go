package adapters

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func (m *mockAPIGatewayClient) GetMethodResponse(ctx context.Context, params *apigateway.GetMethodResponseInput, optFns ...func(*apigateway.Options)) (*apigateway.GetMethodResponseOutput, error) {
	return &apigateway.GetMethodResponseOutput{
		ResponseModels: map[string]string{
			"application/json": "Empty",
		},
		StatusCode: aws.String("200"),
	}, nil
}

func TestApiGatewayMethodResponseGetFunc(t *testing.T) {
	ctx := context.Background()
	cli := mockAPIGatewayClient{}

	input := &apigateway.GetMethodResponseInput{
		RestApiId:  aws.String("rest-api-id"),
		ResourceId: aws.String("resource-id"),
		HttpMethod: aws.String("GET"),
		StatusCode: aws.String("200"),
	}

	item, err := apiGatewayMethodResponseGetFunc(ctx, &cli, "scope", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err = item.Validate(); err != nil {
		t.Fatal(err)
	}

	methodID := fmt.Sprintf("%s/%s/%s", *input.RestApiId, *input.ResourceId, *input.HttpMethod)

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "apigateway-method",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  methodID,
			ExpectedScope:  "scope",
		},
	}

	tests.Execute(t, item)
}

func TestNewAPIGatewayMethodResponseAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)

	client := apigateway.NewFromConfig(config)

	adapter := NewAPIGatewayMethodResponseAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
