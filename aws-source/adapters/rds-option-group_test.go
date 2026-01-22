package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

func TestOptionGroupOutputMapper(t *testing.T) {
	output := rds.DescribeOptionGroupsOutput{
		OptionGroupsList: []types.OptionGroup{
			{
				OptionGroupName:                       PtrString("default:aurora-mysql-8-0"),
				OptionGroupDescription:                PtrString("Default option group for aurora-mysql 8.0"),
				EngineName:                            PtrString("aurora-mysql"),
				MajorEngineVersion:                    PtrString("8.0"),
				Options:                               []types.Option{},
				AllowsVpcAndNonVpcInstanceMemberships: PtrBool(true),
				OptionGroupArn:                        PtrString("arn:aws:rds:eu-west-2:052392120703:og:default:aurora-mysql-8-0"),
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
