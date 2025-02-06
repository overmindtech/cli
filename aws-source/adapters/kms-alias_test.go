package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

func TestAliasOutputMapper(t *testing.T) {
	output := &kms.ListAliasesOutput{
		Aliases: []types.AliasListEntry{
			{
				AliasName:       adapterhelpers.PtrString("alias/test-key"),
				TargetKeyId:     adapterhelpers.PtrString("cf68415c-f4ae-48f2-87a7-3b52ce"),
				AliasArn:        adapterhelpers.PtrString("arn:aws:kms:us-west-2:123456789012:alias/test-key"),
				CreationDate:    adapterhelpers.PtrTime(time.Now()),
				LastUpdatedDate: adapterhelpers.PtrTime(time.Now()),
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

	tests := adapterhelpers.QueryTests{
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
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := kms.NewFromConfig(config)

	adapter := NewKMSAliasAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
