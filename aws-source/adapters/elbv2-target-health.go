package adapters

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type TargetHealthUniqueID struct {
	TargetGroupArn   string
	Id               string
	AvailabilityZone *string
	Port             *int32
}

// String returns a string representation of the TargetHealthUniqueID in the
// format: TargetGroupArn|Id|AvailabilityZone|Port
func (id TargetHealthUniqueID) String() string {
	var az string
	var port string

	if id.AvailabilityZone != nil {
		az = *id.AvailabilityZone
	}

	if id.Port != nil {
		port = fmt.Sprint(*id.Port)
	}

	return strings.Join([]string{
		id.TargetGroupArn,
		id.Id,
		az,
		port,
	}, "|")
}

// ToTargetHealthUniqueID converts a string to a TargetHealthUniqueID
func ToTargetHealthUniqueID(id string) (TargetHealthUniqueID, error) {
	sections := strings.Split(id, "|")

	if len(sections) != 4 {
		return TargetHealthUniqueID{}, fmt.Errorf("cannot parse TargetHealthUniqueID, must have 4 sections, got %v", len(sections))
	}

	healthId := TargetHealthUniqueID{
		TargetGroupArn: sections[0],
		Id:             sections[1],
	}

	if sections[2] != "" {
		healthId.AvailabilityZone = &sections[2]
	}

	if sections[3] != "" {
		port, err := strconv.ParseInt(sections[3], 10, 32)

		if err != nil {
			return TargetHealthUniqueID{}, err
		}

		pint32 := int32(port)

		healthId.Port = &pint32
	}

	return healthId, nil
}

func targetHealthOutputMapper(_ context.Context, _ *elbv2.Client, scope string, input *elbv2.DescribeTargetHealthInput, output *elbv2.DescribeTargetHealthOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, desc := range output.TargetHealthDescriptions {
		attrs, err := adapterhelpers.ToAttributesWithExclude(desc)

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "elbv2-target-health",
			UniqueAttribute: "UniqueId",
			Attributes:      attrs,
			Scope:           scope,
		}

		if desc.TargetHealth != nil {
			switch desc.TargetHealth.State { //nolint:exhaustive // handled by default case
			case types.TargetHealthStateEnumInitial:
				item.Health = sdp.Health_HEALTH_PENDING.Enum()
			case types.TargetHealthStateEnumHealthy:
				item.Health = sdp.Health_HEALTH_OK.Enum()
			case types.TargetHealthStateEnumUnhealthy:
				item.Health = sdp.Health_HEALTH_ERROR.Enum()
			case types.TargetHealthStateEnumUnused:
				item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
			case types.TargetHealthStateEnumDraining:
				item.Health = sdp.Health_HEALTH_PENDING.Enum()
			case types.TargetHealthStateEnumUnavailable:
				item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
			default:
				item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
			}
		}

		// Check that we have an input and not a nil pointer
		if input == nil {
			return nil, fmt.Errorf("input cannot be nil")
		}

		if input.TargetGroupArn == nil {
			return nil, fmt.Errorf("target group ARN cannot be nil")
		}

		// Make sure there is actually a target in this result, there always
		// should be but safer to check
		if desc.Target == nil {
			continue
		}

		if desc.Target.Id == nil {
			continue
		}

		id := TargetHealthUniqueID{
			TargetGroupArn:   *input.TargetGroupArn,
			Id:               *desc.Target.Id,
			AvailabilityZone: desc.Target.AvailabilityZone,
			Port:             desc.Target.Port,
		}

		item.GetAttributes().Set("UniqueId", id.String())

		// See if the ID is an ARN
		a, err := adapterhelpers.ParseARN(*desc.Target.Id)

		if err == nil {
			switch a.Service {
			case "lambda":
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "lambda-function",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *desc.Target.Id,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Everything is tightly coupled with target health
						In:  true,
						Out: true,
					},
				})
			case "elasticloadbalancing":
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "elbv2-load-balancer",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *desc.Target.Id,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
		} else {
			// In this case it could be an instance ID or an IP. We will check
			// for IP first
			if net.ParseIP(*desc.Target.Id) != nil {
				// This means it's an IP
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  *desc.Target.Id,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			} else {
				// If all else fails it must be an instance ID
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-instance",
						Method: sdp.QueryMethod_GET,
						Query:  *desc.Target.Id,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewELBv2TargetHealthAdapter(client *elbv2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*elbv2.DescribeTargetHealthInput, *elbv2.DescribeTargetHealthOutput, *elbv2.Client, *elbv2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*elbv2.DescribeTargetHealthInput, *elbv2.DescribeTargetHealthOutput, *elbv2.Client, *elbv2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "elbv2-target-health",
		AdapterMetadata: targetHealthAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *elbv2.Client, input *elbv2.DescribeTargetHealthInput) (*elbv2.DescribeTargetHealthOutput, error) {
			return client.DescribeTargetHealth(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*elbv2.DescribeTargetHealthInput, error) {
			id, err := ToTargetHealthUniqueID(query)

			if err != nil {
				return nil, err
			}

			return &elbv2.DescribeTargetHealthInput{
				TargetGroupArn: &id.TargetGroupArn,
				Targets: []types.TargetDescription{
					{
						Id:               &id.Id,
						AvailabilityZone: id.AvailabilityZone,
						Port:             id.Port,
					},
				},
			}, nil
		},
		InputMapperList: func(scope string) (*elbv2.DescribeTargetHealthInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for elbv2-target-health, use search",
			}
		},
		InputMapperSearch: func(ctx context.Context, client *elbv2.Client, scope, query string) (*elbv2.DescribeTargetHealthInput, error) {
			// Search by target group ARN
			return &elbv2.DescribeTargetHealthInput{
				TargetGroupArn: &query,
			}, nil
		},
		OutputMapper: targetHealthOutputMapper,
	}
}

var targetHealthAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "elbv2-target-health",
	DescriptiveName: "ELB Target Health",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get target health by unique ID ({TargetGroupArn}|{Id}|{AvailabilityZone}|{Port})",
		SearchDescription: "Search for target health by target group ARN",
	},
	PotentialLinks: []string{"ec2-instance", "lambda-function", "ip", "elbv2-load-balancer"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
})
