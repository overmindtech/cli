package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestLayerVersionGetInputMapper(t *testing.T) {
	tests := []struct {
		Query     string
		ExpectNil bool
	}{
		{
			Query:     "foo:1",
			ExpectNil: false,
		},
		{
			Query:     "foo:1:2",
			ExpectNil: false,
		},
		{
			Query:     "",
			ExpectNil: true,
		},
		{
			Query:     "bar",
			ExpectNil: true,
		},
		{
			Query:     ":",
			ExpectNil: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Query, func(t *testing.T) {
			input := layerVersionGetInputMapper("foo", test.Query)

			if input == nil && !test.ExpectNil {
				t.Error("input was nil unexpectedly")
			}

			if input != nil && test.ExpectNil {
				t.Error("input was non-nil when expected to be nil")
			}
		})
	}
}

func (t *TestLambdaClient) GetLayerVersion(ctx context.Context, params *lambda.GetLayerVersionInput, optFns ...func(*lambda.Options)) (*lambda.GetLayerVersionOutput, error) {
	return &lambda.GetLayerVersionOutput{
		CompatibleArchitectures: []types.Architecture{
			types.ArchitectureArm64,
		},
		CompatibleRuntimes: []types.Runtime{
			types.RuntimeDotnet6,
		},
		Content: &types.LayerVersionContentOutput{
			CodeSha256:               adapterhelpers.PtrString("sha"),
			CodeSize:                 100,
			Location:                 adapterhelpers.PtrString("somewhere"),
			SigningJobArn:            adapterhelpers.PtrString("arn:aws:service:region:account:type/id"),
			SigningProfileVersionArn: adapterhelpers.PtrString("arn:aws:service:region:account:type/id"),
		},
		CreatedDate:     adapterhelpers.PtrString("YYYY-MM-DDThh:mm:ss.sTZD"),
		Description:     adapterhelpers.PtrString("description"),
		LayerArn:        adapterhelpers.PtrString("arn:aws:service:region:account:type/id"),
		LayerVersionArn: adapterhelpers.PtrString("arn:aws:service:region:account:type/id"),
		LicenseInfo:     adapterhelpers.PtrString("info"),
		Version:         *params.VersionNumber,
	}, nil
}

func (t *TestLambdaClient) ListLayerVersions(context.Context, *lambda.ListLayerVersionsInput, ...func(*lambda.Options)) (*lambda.ListLayerVersionsOutput, error) {
	return &lambda.ListLayerVersionsOutput{}, nil
}

func TestLayerVersionGetFunc(t *testing.T) {
	item, err := layerVersionGetFunc(context.Background(), &TestLambdaClient{}, "foo", &lambda.GetLayerVersionInput{
		LayerName:     adapterhelpers.PtrString("layer"),
		VersionNumber: adapterhelpers.PtrInt64(999),
	})

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "signer-signing-job",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "signer-signing-profile",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
	}

	tests.Execute(t, item)
}

func TestNewLambdaLayerVersionAdapter(t *testing.T) {
	client, account, region := lambdaGetAutoConfig(t)

	adapter := NewLambdaLayerVersionAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
