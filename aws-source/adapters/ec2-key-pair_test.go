package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestKeyPairInputMapperGet(t *testing.T) {
	input, err := keyPairInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.KeyNames) != 1 {
		t.Fatalf("expected 1 KeyPair ID, got %v", len(input.KeyNames))
	}

	if input.KeyNames[0] != "bar" {
		t.Errorf("expected KeyPair ID to be bar, got %v", input.KeyNames[0])
	}
}

func TestKeyPairInputMapperList(t *testing.T) {
	input, err := keyPairInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.KeyNames) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestKeyPairOutputMapper(t *testing.T) {
	output := &ec2.DescribeKeyPairsOutput{
		KeyPairs: []types.KeyPairInfo{
			{
				KeyPairId:      adapterhelpers.PtrString("key-04d7068d3a33bf9b2"),
				KeyFingerprint: adapterhelpers.PtrString("df:73:bb:86:a7:cd:9e:18:16:10:50:79:fa:3b:4f:c7:1d:32:cf:58"),
				KeyName:        adapterhelpers.PtrString("dylan.ratcliffe"),
				KeyType:        types.KeyTypeRsa,
				Tags:           []types.Tag{},
				CreateTime:     adapterhelpers.PtrTime(time.Now()),
				PublicKey:      adapterhelpers.PtrString("PUB"),
			},
		},
	}

	items, err := keyPairOutputMapper(context.Background(), nil, "foo", nil, output)

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

}

func TestNewEC2KeyPairAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2KeyPairAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
