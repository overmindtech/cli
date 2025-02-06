package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestFunctionItemMapper(t *testing.T) {
	summary := types.FunctionSummary{
		FunctionConfig: &types.FunctionConfig{
			Comment: adapterhelpers.PtrString("test-comment"),
			Runtime: types.FunctionRuntimeCloudfrontJs20,
		},
		FunctionMetadata: &types.FunctionMetadata{
			FunctionARN:      adapterhelpers.PtrString("arn:aws:cloudfront::123456789012:function/test-function"),
			LastModifiedTime: adapterhelpers.PtrTime(time.Now()),
			CreatedTime:      adapterhelpers.PtrTime(time.Now()),
			Stage:            types.FunctionStageLive,
		},
		Name:   adapterhelpers.PtrString("test-function"),
		Status: adapterhelpers.PtrString("test-status"),
	}

	item, err := functionItemMapper("", "test", &summary)

	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}
}

func TestNewCloudfrontCloudfrontFunctionAdapter(t *testing.T) {
	client, account, _ := CloudfrontGetAutoConfig(t)

	adapter := NewCloudfrontCloudfrontFunctionAdapter(client, account)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
