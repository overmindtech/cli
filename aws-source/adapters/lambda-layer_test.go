package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/overmindtech/cli/sdp-go"
)

func TestLayerItemMapper(t *testing.T) {
	layer := types.LayersListItem{
		LatestMatchingVersion: &types.LayerVersionsListItem{
			CompatibleArchitectures: []types.Architecture{
				types.ArchitectureArm64,
				types.ArchitectureX8664,
			},
			CompatibleRuntimes: []types.Runtime{
				types.RuntimeJava11,
			},
			CreatedDate:     PtrString("2018-11-27T15:10:45.123+0000"),
			Description:     PtrString("description"),
			LayerVersionArn: PtrString("arn:aws:service:region:account:type/id"),
			LicenseInfo:     PtrString("info"),
			Version:         10,
		},
		LayerArn:  PtrString("arn:aws:service:region:account:type/id"),
		LayerName: PtrString("name"),
	}

	item, err := layerItemMapper("", "foo", &layer)

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "lambda-layer-version",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "name:10",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewLambdaLayerAdapter(t *testing.T) {
	client, account, region := lambdaGetAutoConfig(t)

	adapter := NewLambdaLayerAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
		SkipGet: true,
	}

	test.Run(t)
}
