package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestOptionGroupOutputMapper(t *testing.T) {
	output := rds.DescribeOptionGroupsOutput{
		OptionGroupsList: []types.OptionGroup{
			{
				OptionGroupName:                       adapterhelpers.PtrString("default:aurora-mysql-8-0"),
				OptionGroupDescription:                adapterhelpers.PtrString("Default option group for aurora-mysql 8.0"),
				EngineName:                            adapterhelpers.PtrString("aurora-mysql"),
				MajorEngineVersion:                    adapterhelpers.PtrString("8.0"),
				Options:                               []types.Option{},
				AllowsVpcAndNonVpcInstanceMemberships: adapterhelpers.PtrBool(true),
				OptionGroupArn:                        adapterhelpers.PtrString("arn:aws:rds:eu-west-2:052392120703:og:default:aurora-mysql-8-0"),
			},
		},
	}

	items, err := optionGroupOutputMapper(context.Background(), mockRdsClient{}, "foo", nil, &output)

	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("got %v items, expected 1", len(items))
	}

	item := items[0]

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	if item.GetTags()["key"] != "value" {
		t.Errorf("expected key to be value, got %v", item.GetTags()["key"])
	}
}
