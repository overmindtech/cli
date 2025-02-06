package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type platformApplicationClient interface {
	ListPlatformApplications(ctx context.Context, params *sns.ListPlatformApplicationsInput, optFns ...func(*sns.Options)) (*sns.ListPlatformApplicationsOutput, error)
	GetPlatformApplicationAttributes(ctx context.Context, params *sns.GetPlatformApplicationAttributesInput, optFns ...func(*sns.Options)) (*sns.GetPlatformApplicationAttributesOutput, error)
	ListTagsForResource(context.Context, *sns.ListTagsForResourceInput, ...func(*sns.Options)) (*sns.ListTagsForResourceOutput, error)
}

func getPlatformApplicationFunc(ctx context.Context, client platformApplicationClient, scope string, input *sns.GetPlatformApplicationAttributesInput) (*sdp.Item, error) {
	output, err := client.GetPlatformApplicationAttributes(ctx, input)
	if err != nil {
		return nil, err
	}

	if output.Attributes == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "get platform application attributes response was nil",
		}
	}

	attributes, err := adapterhelpers.ToAttributesWithExclude(output.Attributes)
	if err != nil {
		return nil, err
	}

	err = attributes.Set("PlatformApplicationArn", *input.PlatformApplicationArn)
	if err != nil {
		return nil, err
	}

	item := &sdp.Item{
		Type:            "sns-platform-application",
		UniqueAttribute: "PlatformApplicationArn",
		Attributes:      attributes,
		Scope:           scope,
	}

	if resourceTags, err := tagsByResourceARN(ctx, client, *input.PlatformApplicationArn); err == nil {
		item.Tags = tagsToMap(resourceTags)
	}

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "sns-endpoint",
			Method: sdp.QueryMethod_SEARCH,
			Query:  *input.PlatformApplicationArn,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// An unhealthy endpoint won't affect the platform application
			In: false,
			// If platform application is unhealthy, then endpoints won't get notifications
			Out: true,
		},
	})

	return item, nil
}

func NewSNSPlatformApplicationAdapter(client platformApplicationClient, accountID string, region string) *adapterhelpers.AlwaysGetAdapter[*sns.ListPlatformApplicationsInput, *sns.ListPlatformApplicationsOutput, *sns.GetPlatformApplicationAttributesInput, *sns.GetPlatformApplicationAttributesOutput, platformApplicationClient, *sns.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*sns.ListPlatformApplicationsInput, *sns.ListPlatformApplicationsOutput, *sns.GetPlatformApplicationAttributesInput, *sns.GetPlatformApplicationAttributesOutput, platformApplicationClient, *sns.Options]{
		ItemType:        "sns-platform-application",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		ListInput:       &sns.ListPlatformApplicationsInput{},
		AdapterMetadata: platformApplicationAdapterMetadata,
		GetInputMapper: func(scope, query string) *sns.GetPlatformApplicationAttributesInput {
			return &sns.GetPlatformApplicationAttributesInput{
				PlatformApplicationArn: &query,
			}
		},
		ListFuncPaginatorBuilder: func(client platformApplicationClient, input *sns.ListPlatformApplicationsInput) adapterhelpers.Paginator[*sns.ListPlatformApplicationsOutput, *sns.Options] {
			return sns.NewListPlatformApplicationsPaginator(client, input)
		},
		ListFuncOutputMapper: func(output *sns.ListPlatformApplicationsOutput, input *sns.ListPlatformApplicationsInput) ([]*sns.GetPlatformApplicationAttributesInput, error) {
			var inputs []*sns.GetPlatformApplicationAttributesInput
			for _, platformApplication := range output.PlatformApplications {
				inputs = append(inputs, &sns.GetPlatformApplicationAttributesInput{
					PlatformApplicationArn: platformApplication.PlatformApplicationArn,
				})
			}
			return inputs, nil
		},
		GetFunc: getPlatformApplicationFunc,
	}
}

var platformApplicationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "sns-platform-application",
	DescriptiveName: "SNS Platform Application",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an SNS platform application by its ARN",
		ListDescription:   "List all SNS platform applications",
		SearchDescription: "Search SNS platform applications by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_sns_platform_application.id"},
	},
	PotentialLinks: []string{"sns-endpoint"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
})
