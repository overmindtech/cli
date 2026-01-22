package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

func TestOriginAccessControlItemMapper(t *testing.T) {
	x := types.OriginAccessControl{
		Id: PtrString("test"),
		OriginAccessControlConfig: &types.OriginAccessControlConfig{
			Name:                          PtrString("example-name"),
			OriginAccessControlOriginType: types.OriginAccessControlOriginTypesS3,
			SigningBehavior:               types.OriginAccessControlSigningBehaviorsAlways,
			SigningProtocol:               types.OriginAccessControlSigningProtocolsSigv4,
			Description:                   PtrString("example-description"),
		},
	}

	item, err := originAccessControlItemMapper("", "test", &x)

	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}
}

func TestNewCloudfrontOriginAccessControlAdapter(t *testing.T) {
	client, account, _ := CloudfrontGetAutoConfig(t)

	adapter := NewCloudfrontOriginAccessControlAdapter(client, account, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
