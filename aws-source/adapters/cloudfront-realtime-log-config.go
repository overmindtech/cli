package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func realtimeLogConfigsItemMapper(_, scope string, awsItem *types.RealtimeLogConfig) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "cloudfront-realtime-log-config",
		UniqueAttribute: "Name",
		Attributes:      attributes,
		Scope:           scope,
	}

	for _, endpoint := range awsItem.EndPoints {
		if endpoint.KinesisStreamConfig != nil {
			if endpoint.KinesisStreamConfig.RoleARN != nil {
				if arn, err := adapterhelpers.ParseARN(*endpoint.KinesisStreamConfig.RoleARN); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "iam-role",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *endpoint.KinesisStreamConfig.RoleARN,
							Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changes to the role will affect us
							In: true,
							// We can't affect the role
							Out: false,
						},
					})
				}
			}

			if endpoint.KinesisStreamConfig.StreamARN != nil {
				if arn, err := adapterhelpers.ParseARN(*endpoint.KinesisStreamConfig.StreamARN); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "kinesis-stream",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *endpoint.KinesisStreamConfig.StreamARN,
							Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changes to this will affect the stream
							Out: true,
							// The stream can affect us
							In: true,
						},
					})
				}
			}
		}
	}

	return &item, nil
}

func NewCloudfrontRealtimeLogConfigsAdapter(client *cloudfront.Client, accountID string) *adapterhelpers.GetListAdapter[*types.RealtimeLogConfig, *cloudfront.Client, *cloudfront.Options] {
	return &adapterhelpers.GetListAdapter[*types.RealtimeLogConfig, *cloudfront.Client, *cloudfront.Options]{
		ItemType:        "cloudfront-realtime-log-config",
		Client:          client,
		AccountID:       accountID,
		Region:          "", // Cloudfront resources aren't tied to a region
		AdapterMetadata: realtimeLogConfigsAdapterMetadata,
		GetFunc: func(ctx context.Context, client *cloudfront.Client, scope, query string) (*types.RealtimeLogConfig, error) {
			out, err := client.GetRealtimeLogConfig(ctx, &cloudfront.GetRealtimeLogConfigInput{
				Name: &query,
			})

			if err != nil {
				return nil, err
			}

			return out.RealtimeLogConfig, nil
		},
		ListFunc: func(ctx context.Context, client *cloudfront.Client, scope string) ([]*types.RealtimeLogConfig, error) {
			out, err := client.ListRealtimeLogConfigs(ctx, &cloudfront.ListRealtimeLogConfigsInput{})

			if err != nil {
				return nil, err
			}

			logConfigs := make([]*types.RealtimeLogConfig, 0, len(out.RealtimeLogConfigs.Items))

			for _, logConfig := range out.RealtimeLogConfigs.Items {
				logConfigs = append(logConfigs, &logConfig)
			}

			return logConfigs, nil
		},
		ItemMapper: realtimeLogConfigsItemMapper,
	}
}

var realtimeLogConfigsAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "cloudfront-realtime-log-config",
	DescriptiveName: "CloudFront Realtime Log Config",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get Realtime Log Config by Name",
		ListDescription:   "List Realtime Log Configs",
		SearchDescription: "Search Realtime Log Configs by ARN",
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_cloudfront_realtime_log_config.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
})
