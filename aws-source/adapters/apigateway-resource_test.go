package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

/*
{
   "id": "string",
   "parentId": "string",
   "path": "string",
   "pathPart": "string",
   "resourceMethods": {
      "string" : {
         "apiKeyRequired": boolean,
         "authorizationScopes": [ "string" ],
         "authorizationType": "string",
         "authorizerId": "string",
         "httpMethod": "string",
         "methodIntegration": {
            "cacheKeyParameters": [ "string" ],
            "cacheNamespace": "string",
            "connectionId": "string",
            "connectionType": "string",
            "contentHandling": "string",
            "credentials": "string",
            "httpMethod": "string",
            "integrationResponses": {
               "string" : {
                  "contentHandling": "string",
                  "responseParameters": {
                     "string" : "string"
                  },
                  "responseTemplates": {
                     "string" : "string"
                  },
                  "selectionPattern": "string",
                  "statusCode": "string"
               }
            },
            "passthroughBehavior": "string",
            "requestParameters": {
               "string" : "string"
            },
            "requestTemplates": {
               "string" : "string"
            },
            "timeoutInMillis": number,
            "tlsConfig": {
               "insecureSkipVerification": boolean
            },
            "type": "string",
            "uri": "string"
         },
         "methodResponses": {
            "string" : {
               "responseModels": {
                  "string" : "string"
               },
               "responseParameters": {
                  "string" : boolean
               },
               "statusCode": "string"
            }
         },
         "operationName": "string",
         "requestModels": {
            "string" : "string"
         },
         "requestParameters": {
            "string" : boolean
         },
         "requestValidatorId": "string"
      }
   }
}
*/

func TestResourceOutputMapper(t *testing.T) {
	resource := &types.Resource{
		Id:       adapterhelpers.PtrString("test-id"),
		ParentId: adapterhelpers.PtrString("parent-id"),
		Path:     adapterhelpers.PtrString("/test-path"),
		PathPart: adapterhelpers.PtrString("test-path-part"),
		ResourceMethods: map[string]types.Method{
			"GET": {
				ApiKeyRequired:      adapterhelpers.PtrBool(true),
				AuthorizationScopes: []string{"scope1", "scope2"},
				AuthorizationType:   adapterhelpers.PtrString("NONE"),
				AuthorizerId:        adapterhelpers.PtrString("authorizer-id"),
				HttpMethod:          adapterhelpers.PtrString("GET"),
				MethodIntegration: &types.Integration{
					CacheKeyParameters: []string{"param1", "param2"},
					CacheNamespace:     adapterhelpers.PtrString("namespace"),
					ConnectionId:       adapterhelpers.PtrString("connection-id"),
					ConnectionType:     types.ConnectionTypeInternet,
					ContentHandling:    types.ContentHandlingStrategyConvertToBinary,
					Credentials:        adapterhelpers.PtrString("credentials"),
					HttpMethod:         adapterhelpers.PtrString("POST"),
					IntegrationResponses: map[string]types.IntegrationResponse{
						"200": {
							ContentHandling: types.ContentHandlingStrategyConvertToText,
							ResponseParameters: map[string]string{
								"param1": "value1",
							},
							ResponseTemplates: map[string]string{
								"template1": "value1",
							},
							SelectionPattern: adapterhelpers.PtrString("pattern"),
							StatusCode:       adapterhelpers.PtrString("200"),
						},
					},
					PassthroughBehavior: adapterhelpers.PtrString("WHEN_NO_MATCH"),
					RequestParameters: map[string]string{
						"param1": "value1",
					},
					RequestTemplates: map[string]string{
						"template1": "value1",
					},
					TimeoutInMillis: int32(29000),
					TlsConfig: &types.TlsConfig{
						InsecureSkipVerification: false,
					},
					Type: types.IntegrationTypeAwsProxy,
					Uri:  adapterhelpers.PtrString("uri"),
				},
				MethodResponses: map[string]types.MethodResponse{
					"200": {
						ResponseModels: map[string]string{
							"model1": "value1",
						},
						ResponseParameters: map[string]bool{
							"param1": true,
						},
						StatusCode: adapterhelpers.PtrString("200"),
					},
				},
				OperationName: adapterhelpers.PtrString("operation"),
				RequestModels: map[string]string{
					"model1": "value1",
				},
				RequestParameters: map[string]bool{
					"param1": true,
				},
				RequestValidatorId: adapterhelpers.PtrString("validator-id"),
			},
		},
	}

	item, err := resourceOutputMapper("rest-api-13", "scope", resource)
	if err != nil {
		t.Fatal(err)
	}

	if err := item.Validate(); err != nil {
		t.Error(err)
	}
}

func TestNewAPIGatewayResourceAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)

	client := apigateway.NewFromConfig(config)

	adapter := NewAPIGatewayResourceAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
