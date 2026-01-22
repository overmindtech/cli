package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func TestGroupItemMapper(t *testing.T) {
	zone := types.Group{
		Path:       PtrString("/"),
		GroupName:  PtrString("power-users"),
		GroupId:    PtrString("AGPA3VLV2U27T6SSLJMDS"),
		Arn:        PtrString("arn:aws:iam::801795385023:group/power-users"),
		CreateDate: PtrTime(time.Now()),
	}

	item, err := groupItemMapper(nil, "foo", &zone)

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

}

func TestNewIAMGroupAdapter(t *testing.T) {
	config, account, _ := GetAutoConfig(t)
	client := iam.NewFromConfig(config, func(o *iam.Options) {
		o.RetryMode = aws.RetryModeAdaptive
		o.RetryMaxAttempts = 10
	})

	adapter := NewIAMGroupAdapter(client, account, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 30 * time.Second,
	}

	test.Run(t)
}
