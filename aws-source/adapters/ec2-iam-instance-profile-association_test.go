package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestIamInstanceProfileAssociationOutputMapper(t *testing.T) {
	output := ec2.DescribeIamInstanceProfileAssociationsOutput{
		IamInstanceProfileAssociations: []types.IamInstanceProfileAssociation{
			{
				AssociationId: adapterhelpers.PtrString("eipassoc-1234567890abcdef0"),
				IamInstanceProfile: &types.IamInstanceProfile{
					Arn: adapterhelpers.PtrString("arn:aws:iam::123456789012:instance-profile/webserver"), // link
					Id:  adapterhelpers.PtrString("AIDACKCEVSQ6C2EXAMPLE"),
				},
				InstanceId: adapterhelpers.PtrString("i-1234567890abcdef0"), // link
				State:      types.IamInstanceProfileAssociationStateAssociated,
				Timestamp:  adapterhelpers.PtrTime(time.Now()),
			},
		},
	}

	items, err := iamInstanceProfileAssociationOutputMapper(context.Background(), nil, "foo", nil, &output)

	if err != nil {
		t.Error(err)
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Error(err)
		}
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %v", len(items))
	}

	item := items[0]

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "iam-instance-profile",
			ExpectedQuery:  "arn:aws:iam::123456789012:instance-profile/webserver",
			ExpectedScope:  "123456789012",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
		},
		{
			ExpectedType:   "ec2-instance",
			ExpectedQuery:  "i-1234567890abcdef0",
			ExpectedScope:  "foo",
			ExpectedMethod: sdp.QueryMethod_GET,
		},
	}

	tests.Execute(t, item)
}

func TestNewEC2IamInstanceProfileAssociationAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2IamInstanceProfileAssociationAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
