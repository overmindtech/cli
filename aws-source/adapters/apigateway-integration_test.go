package adapters

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type mockAPIGatewayIntegrationClient struct{}

func (m *mockAPIGatewayIntegrationClient) GetIntegration(ctx context.Context, params *apigateway.GetIntegrationInput, optFns ...func(*apigateway.Options)) (*apigateway.GetIntegrationOutput, error) {
	return &apigateway.GetIntegrationOutput{
		IntegrationResponses: map[string]types.IntegrationResponse{
			"200": {
				ResponseTemplates: map[string]string{
					"application/json": "",
				},
				StatusCode: aws.String("200"),
			},
		},
		CacheKeyParameters: []string{},
		Uri:                aws.String("arn:aws:apigateway:us-west-2:lambda:path/2015-03-31/functions/arn:aws:lambda:us-west-2:123412341234:function:My_Function/invocations"),
		HttpMethod:         aws.String("POST"),
		CacheNamespace:     aws.String("y9h6rt"),
		Type:               "AWS",
		ConnectionId:       aws.String("vpc-connection-id"),
	}, nil
}

func TestApiGatewayIntegrationGetFunc(t *testing.T) {
	ctx := context.Background()
	cli := mockAPIGatewayIntegrationClient{}

	input := &apigateway.GetIntegrationInput{
		RestApiId:  aws.String("rest-api-id"),
		ResourceId: aws.String("resource-id"),
		HttpMethod: aws.String("GET"),
	}

	item, err := apiGatewayIntegrationGetFunc(ctx, &cli, "scope", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err = item.Validate(); err != nil {
		t.Fatal(err)
	}

	integrationID := fmt.Sprintf("%s/%s/%s", *input.RestApiId, *input.ResourceId, *input.HttpMethod)

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "apigateway-method",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  integrationID,
			ExpectedScope:  "scope",
		},
		{
			ExpectedType:   "apigateway-vpc-link",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpc-connection-id",
			ExpectedScope:  "scope",
		},
	}

	tests.Execute(t, item)
}

func TestNewAPIGatewayIntegrationAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)

	client := apigateway.NewFromConfig(config)

	adapter := NewAPIGatewayIntegrationAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
