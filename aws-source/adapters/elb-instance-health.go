package adapters

import (
	"context"
	"errors"
	"fmt"
	"strings"

	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

// InstanceHealthName Structured representation of an instance health's unique
// name
type InstanceHealthName struct {
	LoadBalancerName string
	InstanceId       string
}

func (i InstanceHealthName) String() string {
	return fmt.Sprintf("%v/%v", i.LoadBalancerName, i.InstanceId)
}

func ParseInstanceName(name string) (InstanceHealthName, error) {
	sections := strings.Split(name, "/")

	if len(sections) != 2 {
		return InstanceHealthName{}, errors.New("instance health name did not have 2 sections separated by a forward slash")
	}

	return InstanceHealthName{
		LoadBalancerName: sections[0],
		InstanceId:       sections[1],
	}, nil
}

func instanceHealthOutputMapper(_ context.Context, _ *elb.Client, scope string, _ *elb.DescribeInstanceHealthInput, output *elb.DescribeInstanceHealthOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, is := range output.InstanceStates {
		attrs, err := adapterhelpers.ToAttributesWithExclude(is)

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "elb-instance-health",
			UniqueAttribute: "InstanceId",
			Attributes:      attrs,
			Scope:           scope,
		}

		if is.State != nil {
			switch *is.State {
			case "InService":
				item.Health = sdp.Health_HEALTH_OK.Enum()
			case "OutOfService":
				item.Health = sdp.Health_HEALTH_ERROR.Enum()
			case "Unknown":
				item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
			}
		}

		if is.InstanceId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-instance",
					Method: sdp.QueryMethod_GET,
					Query:  *is.InstanceId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// These are tightly linked
					In:  true,
					Out: true,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewELBInstanceHealthAdapter(client *elb.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*elb.DescribeInstanceHealthInput, *elb.DescribeInstanceHealthOutput, *elb.Client, *elb.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*elb.DescribeInstanceHealthInput, *elb.DescribeInstanceHealthOutput, *elb.Client, *elb.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "elb-instance-health",
		AdapterMetadata: instanceHealthAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *elb.Client, input *elb.DescribeInstanceHealthInput) (*elb.DescribeInstanceHealthOutput, error) {
			return client.DescribeInstanceHealth(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*elb.DescribeInstanceHealthInput, error) {
			// This has a composite name defined by `InstanceHealthName`
			name, err := ParseInstanceName(query)

			if err != nil {
				return nil, err
			}

			return &elb.DescribeInstanceHealthInput{
				LoadBalancerName: &name.LoadBalancerName,
				Instances: []types.Instance{
					{
						InstanceId: &name.InstanceId,
					},
				},
			}, nil
		},
		InputMapperList: func(scope string) (*elb.DescribeInstanceHealthInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for elb-instance-health, use search",
			}
		},
		OutputMapper: instanceHealthOutputMapper,
	}
}

var instanceHealthAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "elb-instance-health",
	DescriptiveName: "ELB Instance Health",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:             true,
		List:            true,
		GetDescription:  "Get instance health by ID ({LoadBalancerName}/{InstanceId})",
		ListDescription: "List all instance healths",
	},
	PotentialLinks: []string{"ec2-instance"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
})
