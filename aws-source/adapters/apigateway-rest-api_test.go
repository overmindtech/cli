package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

/*
   {
      "apiKeySource": "string",
      "binaryMediaTypes": [ "string" ],
      "createdDate": number,
      "description": "string",
      "disableExecuteApiEndpoint": boolean,
      "endpointConfiguration": {
         "types": [ "string" ],
         "vpcEndpointIds": [ "string" ]
      },
      "id": "string",
      "minimumCompressionSize": number,
      "name": "string",
      "policy": "string",
      "rootResourceId": "string",
      "tags": {
         "string" : "string"
      },
      "version": "string",
      "warnings": [ "string" ]
   }
*/

func TestRestApiOutputMapper(t *testing.T) {
	output := &apigateway.GetRestApiOutput{
		ApiKeySource:              types.ApiKeySourceTypeHeader,
		BinaryMediaTypes:          []string{"application/json"},
		CreatedDate:               adapterhelpers.PtrTime(time.Now()),
		Description:               adapterhelpers.PtrString("Example API"),
		DisableExecuteApiEndpoint: false,
		EndpointConfiguration: &types.EndpointConfiguration{
			Types:          []types.EndpointType{types.EndpointTypePrivate},
			VpcEndpointIds: []string{"vpce-12345678"},
		},
		Id:                     adapterhelpers.PtrString("abc123"),
		MinimumCompressionSize: adapterhelpers.PtrInt32(1024),
		Name:                   adapterhelpers.PtrString("ExampleAPI"),
		Policy:                 adapterhelpers.PtrString("{\"Version\": \"2012-10-17\", \"Statement\": [{\"Effect\": \"Allow\", \"Principal\": \"*\", \"Action\": \"execute-api:Invoke\", \"Resource\": \"*\"}]}"),
		RootResourceId:         adapterhelpers.PtrString("root123"),
		Tags: map[string]string{
			"env": "production",
		},
		Version:  adapterhelpers.PtrString("v1"),
		Warnings: []string{"This is a warning"},
	}

	item, err := restApiOutputMapper("scope", convertGetRestApiOutputToRestApi(output))
	if err != nil {
		t.Fatal(err)
	}

	if err := item.Validate(); err != nil {
		t.Error(err)
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "ec2-vpc-endpoint",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpce-12345678",
			ExpectedScope:  "scope",
		},
		{
			ExpectedType:   "apigateway-resource",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "abc123/root123",
			ExpectedScope:  "scope",
		},
		{
			ExpectedType:   "apigateway-resource",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "abc123",
			ExpectedScope:  "scope",
		},
		{

			ExpectedType:   "apigateway-model",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "abc123",
			ExpectedScope:  "scope",
		},
		{
			ExpectedType:   "apigateway-deployment",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "abc123",
			ExpectedScope:  "scope",
		},
		{
			ExpectedType:   "apigateway-authorizer",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "abc123",
			ExpectedScope:  "scope",
		},
		{
			ExpectedType:   "apigateway-stage",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "abc123",
			ExpectedScope:  "scope",
		},
	}

	tests.Execute(t, item)
}

func TestNewAPIGatewayRestApiAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)

	client := apigateway.NewFromConfig(config)

	adapter := NewAPIGatewayRestApiAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
