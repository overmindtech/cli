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

type mockAPIGatewayClient struct{}

func (m *mockAPIGatewayClient) GetMethod(ctx context.Context, params *apigateway.GetMethodInput, optFns ...func(*apigateway.Options)) (*apigateway.GetMethodOutput, error) {
	return &apigateway.GetMethodOutput{
		ApiKeyRequired:     aws.Bool(false),
		HttpMethod:         aws.String("GET"),
		AuthorizationType:  aws.String("NONE"),
		AuthorizerId:       aws.String("authorizer-id"),
		RequestParameters:  map[string]bool{},
		RequestValidatorId: aws.String("request-validator-id"),
		MethodResponses: map[string]types.MethodResponse{
			"200": {
				ResponseModels: map[string]string{
					"application/json": "Empty",
				},
				StatusCode: aws.String("200"),
			},
		},
		MethodIntegration: &types.Integration{
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
		},
	}, nil

}

func TestApiGatewayGetFunc(t *testing.T) {
	ctx := context.Background()
	cli := mockAPIGatewayClient{}

	input := &apigateway.GetMethodInput{
		RestApiId:  aws.String("rest-api-id"),
		ResourceId: aws.String("resource-id"),
		HttpMethod: aws.String("GET"),
	}

	item, err := apiGatewayMethodGetFunc(ctx, &cli, "scope", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err = item.Validate(); err != nil {
		t.Fatal(err)
	}

	methodID := fmt.Sprintf("%s/%s/%s", *input.RestApiId, *input.ResourceId, *input.HttpMethod)
	authorizerID := fmt.Sprintf("%s/%s", *input.RestApiId, "authorizer-id")
	validatorID := fmt.Sprintf("%s/%s", *input.RestApiId, "request-validator-id")

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "apigateway-integration",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  methodID,
			ExpectedScope:  "scope",
		},
		{
			ExpectedType:   "apigateway-authorizer",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  authorizerID,
			ExpectedScope:  "scope",
		},
		{
			ExpectedType:   "apigateway-request-validator",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  validatorID,
			ExpectedScope:  "scope",
		},
	}

	tests.Execute(t, item)
}

func TestNewAPIGatewayMethodAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)

	client := apigateway.NewFromConfig(config)

	adapter := NewAPIGatewayMethodAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
