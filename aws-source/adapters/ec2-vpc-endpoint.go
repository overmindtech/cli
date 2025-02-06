package adapters

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/micahhausler/aws-iam-policy/policy"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func vpcEndpointInputMapperGet(scope string, query string) (*ec2.DescribeVpcEndpointsInput, error) {
	return &ec2.DescribeVpcEndpointsInput{
		VpcEndpointIds: []string{
			query,
		},
	}, nil
}

func vpcEndpointInputMapperList(scope string) (*ec2.DescribeVpcEndpointsInput, error) {
	return &ec2.DescribeVpcEndpointsInput{}, nil
}

func vpcEndpointOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeVpcEndpointsInput, output *ec2.DescribeVpcEndpointsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, endpoint := range output.VpcEndpoints {
		var err error
		var attrs *sdp.ItemAttributes

		// A type that we use to override the PolicyDocument with the parsed
		// version
		type endpointParsedPolicy struct {
			types.VpcEndpoint
			PolicyDocument *policy.Policy
		}
		endpointWithPolicy := endpointParsedPolicy{
			VpcEndpoint: endpoint,
		}

		// Parse the policy
		if endpoint.PolicyDocument != nil {
			parsedPolicy, _ := ParsePolicyDocument(*endpoint.PolicyDocument)
			endpointWithPolicy.PolicyDocument = parsedPolicy
		}

		attrs, err = adapterhelpers.ToAttributesWithExclude(endpointWithPolicy, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-vpc-endpoint",
			UniqueAttribute: "VpcEndpointId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(endpoint.Tags),
		}

		// Annoyingly the API doesn't follow its own specification here and
		// returns values in lowercase -_-
		state := strings.ToLower(string(endpoint.State))
		switch state {
		case strings.ToLower(string(types.StatePendingAcceptance)):
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case strings.ToLower(string(types.StatePending)):
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case strings.ToLower(string(types.StateAvailable)):
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case strings.ToLower(string(types.StateDeleting)):
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case strings.ToLower(string(types.StateDeleted)):
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case strings.ToLower(string(types.StateRejected)):
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		case strings.ToLower(string(types.StateFailed)):
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		case strings.ToLower(string(types.StateExpired)):
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		}

		if endpoint.VpcId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *endpoint.VpcId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// We can't affect the VPC overall
					In:  true,
					Out: false,
				},
			})
		}

		if endpointWithPolicy.PolicyDocument != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, LinksFromPolicy(endpointWithPolicy.PolicyDocument)...)
		}

		for _, routeTableID := range endpoint.RouteTableIds {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-route-table",
					Method: sdp.QueryMethod_GET,
					Query:  routeTableID,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// We can't affect the route table overall
					In:  true,
					Out: false,
				},
			})
		}

		for _, subnetID := range endpoint.SubnetIds {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-subnet",
					Method: sdp.QueryMethod_GET,
					Query:  subnetID,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// We can't affect the subnet overall
					In:  true,
					Out: false,
				},
			})
		}

		for _, group := range endpoint.Groups {
			if group.GroupId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-security-group",
						Method: sdp.QueryMethod_GET,
						Query:  *group.GroupId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// We can't affect the security group overall
						In:  true,
						Out: false,
					},
				})
			}
		}

		for _, dnsEntry := range endpoint.DnsEntries {
			if dnsEntry.DnsName != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "dns",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *dnsEntry.DnsName,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// These are tightly linked
						In:  true,
						Out: true,
					},
				})
			}

			if dnsEntry.HostedZoneId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "route53-hosted-zone",
						Method: sdp.QueryMethod_GET,
						Query:  *dnsEntry.HostedZoneId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// We can't affect the hosted zone overall
						In:  true,
						Out: false,
					},
				})
			}
		}

		for _, networkInterfaceID := range endpoint.NetworkInterfaceIds {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-network-interface",
					Method: sdp.QueryMethod_GET,
					Query:  networkInterfaceID,
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

func NewEC2VpcEndpointAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeVpcEndpointsInput, *ec2.DescribeVpcEndpointsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeVpcEndpointsInput, *ec2.DescribeVpcEndpointsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-vpc-endpoint",
		AdapterMetadata: vpcEndpointAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeVpcEndpointsInput) (*ec2.DescribeVpcEndpointsOutput, error) {
			return client.DescribeVpcEndpoints(ctx, input)
		},
		InputMapperGet:  vpcEndpointInputMapperGet,
		InputMapperList: vpcEndpointInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeVpcEndpointsInput) adapterhelpers.Paginator[*ec2.DescribeVpcEndpointsOutput, *ec2.Options] {
			return ec2.NewDescribeVpcEndpointsPaginator(client, params)
		},
		OutputMapper: vpcEndpointOutputMapper,
	}
}

var vpcEndpointAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-vpc-endpoint",
	DescriptiveName: "VPC Endpoint",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a VPC Endpoint by ID",
		ListDescription:   "List all VPC Endpoints",
		SearchDescription: "Search VPC Endpoints by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_vpc_endpoint.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
