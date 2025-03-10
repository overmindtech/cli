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

func TestStageOutputMapper(t *testing.T) {
	awsItem := &types.Stage{
		DeploymentId:         aws.String("deployment-id"),
		StageName:            aws.String("stage-name"),
		Description:          aws.String("description"),
		CreatedDate:          aws.Time(time.Now()),
		LastUpdatedDate:      aws.Time(time.Now()),
		Variables:            map[string]string{"key": "value"},
		AccessLogSettings:    &types.AccessLogSettings{},
		CacheClusterEnabled:  true,
		CacheClusterSize:     "0.5",
		CacheClusterStatus:   types.CacheClusterStatusAvailable,
		CanarySettings:       &types.CanarySettings{},
		ClientCertificateId:  aws.String("client-cert-id"),
		DocumentationVersion: aws.String("1.0"),
		MethodSettings:       map[string]types.MethodSetting{},
		TracingEnabled:       true,
		WebAclArn:            aws.String("web-acl-arn"),
		Tags:                 map[string]string{"tag-key": "tag-value"},
	}

	queries := []string{"rest-api-id/stage-name", "rest-api-id/deployment-id", "rest-api-id"}
	for _, query := range queries {
		item, err := stageOutputMapper(query, "scope", awsItem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := item.Validate(); err != nil {
			t.Error(err)
		}

		tests := adapterhelpers.QueryTests{
			{
				ExpectedType:   "apigateway-deployment",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "rest-api-id/deployment-id",
				ExpectedScope:  "scope",
			},
			{
				ExpectedType:   "apigateway-rest-api",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "rest-api-id",
				ExpectedScope:  "scope",
			},
		}

		tests.Execute(t, item)
	}
}

func TestNewAPIGatewayStageAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)

	client := apigateway.NewFromConfig(config)

	adapter := NewAPIGatewayStageAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
