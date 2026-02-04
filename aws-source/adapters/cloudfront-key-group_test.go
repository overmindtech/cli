package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/overmindtech/cli/sdpcache"
)

func TestKeyGroupItemMapper(t *testing.T) {
	group := types.KeyGroup{
		Id: PtrString("test-id"),
		KeyGroupConfig: &types.KeyGroupConfig{
			Items: []string{
				"some-identity",
			},
			Name:    PtrString("test-name"),
			Comment: PtrString("test-comment"),
		},
		LastModifiedTime: PtrTime(time.Now()),
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

	adapter := NewCloudfrontKeyGroupAdapter(client, account, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
