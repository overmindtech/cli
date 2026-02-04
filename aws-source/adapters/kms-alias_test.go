package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

func TestAliasOutputMapper(t *testing.T) {
	output := &kms.ListAliasesOutput{
		Aliases: []types.AliasListEntry{
			{
				AliasName:       PtrString("alias/test-key"),
				TargetKeyId:     PtrString("cf68415c-f4ae-48f2-87a7-3b52ce"),
				AliasArn:        PtrString("arn:aws:kms:us-west-2:123456789012:alias/test-key"),
				CreationDate:    PtrTime(time.Now()),
				LastUpdatedDate: PtrTime(time.Now()),
			},
		},
	}

	items, err := aliasOutputMapper(context.Background(), nil, "foo", nil, output)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Error(err)
		}
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", len(items))
	}

	item := items[0]

	tests := QueryTests{
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "cf68415c-f4ae-48f2-87a7-3b52ce",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewKMSAliasAdapter(t *testing.T) {
	config, account, region := GetAutoConfig(t)
	client := kms.NewFromConfig(config)

	adapter := NewKMSAliasAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
