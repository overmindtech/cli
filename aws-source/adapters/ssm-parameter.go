package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/sourcegraph/conc/iter"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ssmClient interface {
	DescribeParameters(context.Context, *ssm.DescribeParametersInput, ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error)
	ListTagsForResource(ctx context.Context, params *ssm.ListTagsForResourceInput, optFns ...func(*ssm.Options)) (*ssm.ListTagsForResourceOutput, error)
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

func ssmParameterInputMapperSearch(ctx context.Context, client ssmClient, scope, query string) (*ssm.DescribeParametersInput, error) {
	// According to the docs here:
	// https://docs.aws.amazon.com/systems-manager/latest/userguide/sysman-paramstore-access.html
	// it's common to use wildcards in SSM parameter ARNS in policies, an
	// example might look like this:
	//
	// {
	//   "Sid": "ParameterStoreActions",
	//   "Effect": "Allow",
	//   "Action": [
	//     "ssm:GetParametersByPath"
	//   ],
	//   "Resource": [
	//     "arn:aws:ssm:us-east-1:1234567890:parameter/prod/service/example-service",
	//     "arn:aws:ssm:us-east-1:1234567890:parameter/prod/*/service/example-service"
	//   ]
	// }
	//
	// This means that we can't just use a simple "Equals" filter, we need to be
	// smarter than that. When we're filtering by name, we can use "Equals",
	// "BeginsWith" and "Contains". The other issue is that in the above
	// example, the user is allowed to run "GetParametersByPath" which allows
	// them request recursive results. This will mean that there is an implicit
	// asterisk (*) at the end of the path, whereas if the "Action" was
	// "GetParameter" then the user would have to specify the exact path. They'd
	// still be able to use IAM wildcards, but the path would need to be
	// complete
	//
	// I think to make this really accurate we would need to take this into
	// account, however maybe to begin with we can at least start by trying to
	// replicate the asterisk behaviour both at the end and inside the path.
	//
	// I was thinking that we should re-implement the IAM wildcard parsing logic
	// from the docs:
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_resource.html#reference_policies_elements_resource_wildcards
	// however I don't know if this will be worth doing as it'll only be able to
	// be applied *after* we have queried the data

	// Parse the ARN
	parsedArn, err := adapterhelpers.ParseARN(query)
	if err != nil {
		return nil, fmt.Errorf("invalid ARN format: %w", err)
	}

	// For SSM parameters, the resource part starts with "parameter/"
	if !strings.HasPrefix(parsedArn.Resource, "parameter/") {
		return nil, fmt.Errorf("invalid SSM parameter ARN: resource must start with 'parameter/'")
	}

	// Extract the parameter name (everything after "parameter/")
	parameterPath := strings.TrimPrefix(parsedArn.Resource, "parameter/")

	// Handle wildcards in the path
	if strings.Contains(parameterPath, "*") || strings.Contains(parameterPath, "?") {
		// Se need to be smart about this in order to make efficient queries.
		// The options we have are "BeginsWith" and "Contains" so I think we
		// should pick the longest substring we can, then query based on that.
		// We will need to split on all the possible wildcards (* and ?), then
		// work out the longest segment, then use that in a "Contains" query

		// Split on both * and ? to get all segments
		segments := strings.FieldsFunc(parameterPath, func(r rune) bool {
			return r == '*' || r == '?'
		})

		// Find the longest segment
		longestSegment := ""
		for _, segment := range segments {
			if len(segment) > len(longestSegment) {
				longestSegment = segment
			}
		}

		// If we have no valid segments after splitting (e.g. "***")
		if longestSegment == "" {
			// If it's all wildcards then search for everything
			return &ssm.DescribeParametersInput{}, nil
		}

		// Use Contains with the longest segment for most efficient filtering
		return &ssm.DescribeParametersInput{
			ParameterFilters: []types.ParameterStringFilter{
				{
					Key:    aws.String("Name"),
					Option: aws.String("Contains"),
					Values: []string{longestSegment},
				},
			},
		}, nil
	}

	// If no wildcards, do an exact match
	return &ssm.DescribeParametersInput{
		ParameterFilters: []types.ParameterStringFilter{
			{
				Key:    aws.String("Name"),
				Option: aws.String("Equals"),
				Values: []string{parameterPath},
			},
		},
	}, nil
}

func ssmParameterPostSearchFilter(ctx context.Context, query string, items []*sdp.Item) ([]*sdp.Item, error) {
	arn, err := adapterhelpers.ParseARN(query)
	if err != nil {
		return nil, fmt.Errorf("invalid ARN format: %w", err)
	}

	// Filter out any items that don't match the ARN wildcard format
	filteredItems := make([]*sdp.Item, 0)
	for _, item := range items {
		itemArn, err := item.GetAttributes().Get("ARN")
		if err != nil {
			return nil, fmt.Errorf("missing ARN attribute: %w for item: %v", err, item.GloballyUniqueName())
		}

		if arn.IAMWildcardMatches(fmt.Sprint(itemArn)) {
			filteredItems = append(filteredItems, item)
		}
	}

	return filteredItems, nil
}

func ssmParameterOutputMapper(ctx context.Context, client ssmClient, scope string, input *ssm.DescribeParametersInput, output *ssm.DescribeParametersOutput) ([]*sdp.Item, error) {
	items, err := iter.MapErr(output.Parameters, func(parameter *types.ParameterMetadata) (*sdp.Item, error) {
		attrs, err := adapterhelpers.ToAttributesWithExclude(parameter)

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "ssm-parameter",
			UniqueAttribute: "Name",
			Attributes:      attrs,
			Scope:           scope,
		}

		// Next thing we want to is try to add tags to this item by running ListTagsForResource
		var tags map[string]string
		tagsOut, err := client.ListTagsForResource(ctx, &ssm.ListTagsForResourceInput{
			ResourceId:   parameter.Name,
			ResourceType: types.ResourceTypeForTaggingParameter,
		})
		if err != nil {
			// If we can't get the tags we don't want to do anything drastic
			// since it's not a critical error
			tags = adapterhelpers.HandleTagsError(ctx, err)
		} else {
			tags = make(map[string]string)
			for _, tag := range tagsOut.TagList {
				if tag.Key != nil && tag.Value != nil {
					tags[*tag.Key] = *tag.Value
				}
			}
		}
		item.Tags = tags

		// Now we need to try to get the actual value and link from it. However
		// we don't want to see any secrets so we'll skip those
		if parameter.Type != types.ParameterTypeSecureString {
			request := &ssm.GetParameterInput{
				Name:           parameter.Name,
				WithDecryption: adapterhelpers.PtrBool(false), // let's be double sure we don't get any secrets
			}
			paramResp, err := client.GetParameter(ctx, request)
			if err != nil {
				// Attach an event in the span
				span := trace.SpanFromContext(ctx)

				span.AddEvent("Error getting parameter value", trace.WithAttributes(
					attribute.String("error", err.Error()),
					attribute.String("parameter_name", *parameter.Name),
					attribute.String("item", item.GloballyUniqueName()),
				))
				return nil, err
			}

			if paramResp.Parameter != nil && paramResp.Parameter.Value != nil {
				// Add the value to the item
				item.GetAttributes().Set("Value", *paramResp.Parameter.Value)

				// Extract links from the value
				newLinks, err := sdp.ExtractLinksFrom(*paramResp.Parameter.Value)
				if err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, newLinks...)
				}
			}
		}

		return &item, nil
	})

	return items, err
}

func NewSSMParameterAdapter(client ssmClient, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ssm.DescribeParametersInput, *ssm.DescribeParametersOutput, ssmClient, *ssm.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ssm.DescribeParametersInput, *ssm.DescribeParametersOutput, ssmClient, *ssm.Options]{
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		ItemType:        "ssm-parameter",
		AdapterMetadata: ssmParameterAdapterMetadata,
		InputMapperGet: func(scope, query string) (*ssm.DescribeParametersInput, error) {
			return &ssm.DescribeParametersInput{
				ParameterFilters: []types.ParameterStringFilter{
					{
						Key:    adapterhelpers.PtrString("Name"),
						Option: adapterhelpers.PtrString("Equals"),
						Values: []string{query},
					},
				},
			}, nil
		},
		InputMapperList: func(scope string) (*ssm.DescribeParametersInput, error) {
			return &ssm.DescribeParametersInput{}, nil
		},
		OutputMapper:      ssmParameterOutputMapper,
		InputMapperSearch: ssmParameterInputMapperSearch,
		PostSearchFilter:  ssmParameterPostSearchFilter,
		PaginatorBuilder: func(client ssmClient, params *ssm.DescribeParametersInput) adapterhelpers.Paginator[*ssm.DescribeParametersOutput, *ssm.Options] {
			return ssm.NewDescribeParametersPaginator(client, params, func(dppo *ssm.DescribeParametersPaginatorOptions) {
				dppo.Limit = 50
			})
		},
		DescribeFunc: func(ctx context.Context, client ssmClient, input *ssm.DescribeParametersInput) (*ssm.DescribeParametersOutput, error) {
			return client.DescribeParameters(ctx, input)
		},
	}
}

var ssmParameterAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ssm-parameter",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	DescriptiveName: "SSM Parameter",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		GetDescription:    "Get an SSM parameter by name",
		List:              true,
		ListDescription:   "List all SSM parameters",
		Search:            true,
		SearchDescription: "Search for SSM parameters by ARN. This supports ARNs from IAM policies that contain wildcards",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "aws_ssm_parameter.name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "aws_ssm_parameter.arn",
		},
	},
	PotentialLinks: []string{
		"ip",
		"http",
		"dns",
	},
})
