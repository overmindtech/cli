package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestKeyGroupItemMapper(t *testing.T) {
	group := types.KeyGroup{
		Id: adapterhelpers.PtrString("test-id"),
		KeyGroupConfig: &types.KeyGroupConfig{
			Items: []string{
				"some-identity",
			},
			Name:    adapterhelpers.PtrString("test-name"),
			Comment: adapterhelpers.PtrString("test-comment"),
		},
		LastModifiedTime: adapterhelpers.PtrTime(time.Now()),
	}

	item, err := KeyGroupItemMapper("", "test", &group)

	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}
}

func TestNewCloudfrontKeyGroupAdapter(t *testing.T) {
	client, account, _ := CloudfrontGetAutoConfig(t)

	adapter := NewCloudfrontKeyGroupAdapter(client, account)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
