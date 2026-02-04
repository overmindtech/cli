package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/overmindtech/cli/sdpcache"
)

func TestFunctionItemMapper(t *testing.T) {
	summary := types.FunctionSummary{
		FunctionConfig: &types.FunctionConfig{
			Comment: PtrString("test-comment"),
			Runtime: types.FunctionRuntimeCloudfrontJs20,
		},
		FunctionMetadata: &types.FunctionMetadata{
			FunctionARN:      PtrString("arn:aws:cloudfront::123456789012:function/test-function"),
			LastModifiedTime: PtrTime(time.Now()),
			CreatedTime:      PtrTime(time.Now()),
			Stage:            types.FunctionStageLive,
		},
		Name:   PtrString("test-function"),
		Status: PtrString("test-status"),
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

	adapter := NewCloudfrontCloudfrontFunctionAdapter(client, account, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
