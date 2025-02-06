package adapters

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type HealthCheck struct {
	types.HealthCheck
	HealthCheckObservations []types.HealthCheckObservation
}

func healthCheckGetFunc(ctx context.Context, client *route53.Client, scope, query string) (*HealthCheck, error) {
	out, err := client.GetHealthCheck(ctx, &route53.GetHealthCheckInput{
		HealthCheckId: &query,
	})

	if err != nil {
		return nil, err
	}

	status, err := client.GetHealthCheckStatus(ctx, &route53.GetHealthCheckStatusInput{
		HealthCheckId: &query,
	})

	if err != nil {
		return nil, err
	}

	return &HealthCheck{
		HealthCheck:             *out.HealthCheck,
		HealthCheckObservations: status.HealthCheckObservations,
	}, nil
}

func healthCheckListFunc(ctx context.Context, client *route53.Client, scope string) ([]*HealthCheck, error) {
	out, err := client.ListHealthChecks(ctx, &route53.ListHealthChecksInput{})

	if err != nil {
		return nil, err
	}

	healthChecks := make([]*HealthCheck, 0, len(out.HealthChecks))

	for _, healthCheck := range out.HealthChecks {
		status, err := client.GetHealthCheckStatus(ctx, &route53.GetHealthCheckStatusInput{
			HealthCheckId: healthCheck.Id,
		})

		if err != nil {
			return nil, err
		}

		healthChecks = append(healthChecks, &HealthCheck{
			HealthCheck:             healthCheck,
			HealthCheckObservations: status.HealthCheckObservations,
		})
	}

	return healthChecks, nil
}

func healthCheckItemMapper(_, scope string, awsItem *HealthCheck) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "route53-health-check",
		UniqueAttribute: "Id",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link to the cloudwatch metric that tracks this health check
	query, err := ToQueryString(&cloudwatch.DescribeAlarmsForMetricInput{
		Namespace:  aws.String("AWS/Route53"),
		MetricName: aws.String("HealthCheckStatus"),
		Dimensions: []cwtypes.Dimension{
			{
				Name:  aws.String("HealthCheckId"),
				Value: awsItem.Id,
			},
		},
	})

	if err == nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "cloudwatch-alarm",
				Query:  query,
				Method: sdp.QueryMethod_SEARCH,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Tightly coupled
				In:  true,
				Out: true,
			},
		})
	}

	healthy := true

	for _, observation := range awsItem.HealthCheckObservations {
		if observation.StatusReport != nil && observation.StatusReport.Status != nil {
			if strings.HasPrefix(*observation.StatusReport.Status, "Failure") {
				healthy = false
			}
		}
	}

	if healthy {
		item.Health = sdp.Health_HEALTH_OK.Enum()
	} else {
		item.Health = sdp.Health_HEALTH_ERROR.Enum()
	}

	return &item, nil
}

func NewRoute53HealthCheckAdapter(client *route53.Client, accountID string, region string) *adapterhelpers.GetListAdapter[*HealthCheck, *route53.Client, *route53.Options] {
	return &adapterhelpers.GetListAdapter[*HealthCheck, *route53.Client, *route53.Options]{
		ItemType:        "route53-health-check",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		GetFunc:         healthCheckGetFunc,
		ListFunc:        healthCheckListFunc,
		ItemMapper:      healthCheckItemMapper,
		AdapterMetadata: healthCheckAdapterMetadata,
		ListTagsFunc: func(ctx context.Context, hc *HealthCheck, c *route53.Client) (map[string]string, error) {
			if hc.Id == nil {
				return nil, nil
			}

			// Strip the prefix
			id := strings.TrimPrefix(*hc.Id, "/healthcheck/")

			out, err := c.ListTagsForResource(ctx, &route53.ListTagsForResourceInput{
				ResourceId:   &id,
				ResourceType: types.TagResourceTypeHealthcheck,
			})

			if err != nil {
				return nil, err
			}

			return route53TagsToMap(out.ResourceTagSet.Tags), nil
		},
	}
}

var healthCheckAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "route53-health-check",
	DescriptiveName: "Route53 Health Check",
	PotentialLinks:  []string{"cloudwatch-alarm"},
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get health check by ID",
		ListDescription:   "List all health checks",
		SearchDescription: "Search for health checks by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_route53_health_check.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
})
