package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func instanceEventWindowInputMapperGet(scope, query string) (*ec2.DescribeInstanceEventWindowsInput, error) {
	return &ec2.DescribeInstanceEventWindowsInput{
		InstanceEventWindowIds: []string{
			query,
		},
	}, nil
}

func instanceEventWindowInputMapperList(scope string) (*ec2.DescribeInstanceEventWindowsInput, error) {
	return &ec2.DescribeInstanceEventWindowsInput{}, nil
}

func instanceEventWindowOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeInstanceEventWindowsInput, output *ec2.DescribeInstanceEventWindowsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, ew := range output.InstanceEventWindows {
		attrs, err := adapterhelpers.ToAttributesWithExclude(ew, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-instance-event-window",
			UniqueAttribute: "InstanceEventWindowId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(ew.Tags),
		}

		if at := ew.AssociationTarget; at != nil {
			for _, id := range at.DedicatedHostIds {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-host",
						Method: sdp.QueryMethod_GET,
						Query:  id,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the host won't affect the window
						In: false,
						// Changing the windows will affect the host
						Out: true,
					},
				})
			}

			for _, id := range at.InstanceIds {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-instance",
						Method: sdp.QueryMethod_GET,
						Query:  id,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the host won't affect the window
						In: false,
						// Changing the windows will affect the instance
						Out: true,
					},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2InstanceEventWindowAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeInstanceEventWindowsInput, *ec2.DescribeInstanceEventWindowsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeInstanceEventWindowsInput, *ec2.DescribeInstanceEventWindowsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-instance-event-window",
		AdapterMetadata: instanceEventWindowAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeInstanceEventWindowsInput) (*ec2.DescribeInstanceEventWindowsOutput, error) {
			return client.DescribeInstanceEventWindows(ctx, input)
		},
		InputMapperGet:  instanceEventWindowInputMapperGet,
		InputMapperList: instanceEventWindowInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeInstanceEventWindowsInput) adapterhelpers.Paginator[*ec2.DescribeInstanceEventWindowsOutput, *ec2.Options] {
			return ec2.NewDescribeInstanceEventWindowsPaginator(client, params)
		},
		OutputMapper: instanceEventWindowOutputMapper,
	}
}

var instanceEventWindowAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-instance-event-window",
	DescriptiveName: "EC2 Instance Event Window",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an event window by ID",
		ListDescription:   "List all event windows",
		SearchDescription: "Search for event windows by ARN",
	},
	PotentialLinks: []string{"ec2-host", "ec2-instance"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
})
