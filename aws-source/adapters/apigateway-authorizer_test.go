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

func TestAuthorizerOutputMapper(t *testing.T) {
	awsItem := &types.Authorizer{
		Id:                           aws.String("authorizer-id"),
		Name:                         aws.String("authorizer-name"),
		Type:                         types.AuthorizerTypeRequest,
		ProviderARNs:                 []string{"arn:aws:iam::123456789012:role/service-role"},
		AuthType:                     aws.String("custom"),
		AuthorizerUri:                aws.String("arn:aws:apigateway:us-east-1:lambda:path/2015-03-31/functions/arn:aws:lambda:us-east-1:123456789012:function:my-function/invocations"),
		AuthorizerCredentials:        aws.String("arn:aws:iam::123456789012:role/service-role"),
		IdentitySource:               aws.String("method.request.header.Authorization"),
		IdentityValidationExpression: aws.String(".*"),
		AuthorizerResultTtlInSeconds: aws.Int32(300),
	}

	item, err := authorizerOutputMapper("rest-api-id", "scope", awsItem)
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

func TestNewAPIGatewayAuthorizerAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)

	client := apigateway.NewFromConfig(config)

	adapter := NewAPIGatewayAuthorizerAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
