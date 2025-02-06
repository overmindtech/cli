package adapters

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

var ErrNoQuery = errors.New("no query found")

// SuggestQueries Suggests a linked item query based on the namespace and
// dimensions of a metric. For metrics with many dimensions, it will use the
// most specific dimension since many metrics have overlapping dimensions that
// get more and more specific
//
// The full list of services that provide cloudwatch metrics can be found here:
// https://github.com/awsdocs/amazon-cloudwatch-user-guide/blob/master/doc_source/aws-services-cloudwatch-metrics.md
//
// The below list is not exhaustive and improvements are welcome
func SuggestedQuery(namespace string, scope string, dimensions []types.Dimension) (*sdp.LinkedItemQuery, error) {
	var query *sdp.Query
	var err error

	bp := &sdp.BlastPropagation{
		// These links are the metrics that feed the alarms. If the thing that
		// we're measuring changes, we definitely want the alarm to be in the
		// blast radius. But an alarm on its own doesn't affect these things
		In:  false,
		Out: true,
	}

	accountID, _, err := adapterhelpers.ParseScope(scope)

	if err != nil {
		return nil, err
	}

	switch namespace {
	case "AWS/Route53":
		if d := getDimension("HostedZoneId", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "route53-hosted-zone",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}

		if d := getDimension("HealthCheckId", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "route53-health-check",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}
	case "AWS/Lambda":
		if d := getDimension("FunctionName", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "lambda-function",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}
	case "AWS/DynamoDB":
		if d := getDimension("TableName", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "dynamodb-table",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}
	case "AWS/ECS":
		if d := getDimension("ServiceName", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "ecs-service",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}

			break
		}

		if d := getDimension("ClusterName", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "ecs-cluster",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}

			break
		}
	case "AWS/ELB":
		if d := getDimension("LoadBalancerName", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "elb-load-balancer",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}
	case "AWS/EC2":
		if d := getDimension("InstanceId", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "ec2-instance",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}
		if d := getDimension("AutoScalingGroupName", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "autoscaling-auto-scaling-group",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}
		if d := getDimension("ImageId", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "ec2-image",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}
	case "AWS/RDS":
		if d := getDimension("DBInstanceIdentifier", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "rds-db-instance",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}

			break
		}

		if d := getDimension("DBClusterIdentifier", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "rds-db-cluster",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}

			break
		}
	case "AWS/EBS":
		if d := getDimension("VolumeId", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "ec2-volume",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}
	case "AWS/ApplicationELB", "AWS/NetworkELB":
		if d := getDimension("TargetGroup", dimensions); d != nil {
			sections := strings.Split(*d.Value, "/")

			if len(sections) == 3 {
				query = &sdp.Query{
					Type:   "elbv2-target-group",
					Method: sdp.QueryMethod_GET,
					Query:  sections[1],
					Scope:  scope,
				}

				break
			}
		}
		if d := getDimension("LoadBalancer", dimensions); d != nil {
			sections := strings.Split(*d.Value, "/")

			if len(sections) == 3 {
				query = &sdp.Query{
					Type:   "elbv2-load-balancer",
					Method: sdp.QueryMethod_GET,
					Query:  sections[1],
					Scope:  scope,
				}

				break
			}
		}
	case "AWS/Backup":
		if d := getDimension("BackupVaultName", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "backup-backup-vault",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}
	case "AWS/S3":
		if d := getDimension("BucketName", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "s3-bucket",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  adapterhelpers.FormatScope(accountID, ""),
			}
		}
	case "AWS/NATGateway":
		if d := getDimension("NatGatewayId", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "ec2-nat-gateway",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}
	case "AWS/CertificateManager":
		if d := getDimension("CertificateArn", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "acm-certificate",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}
	case "AWS/EFS":
		if d := getDimension("FileSystemId", dimensions); d != nil {
			query = &sdp.Query{
				Type:   "efs-file-system",
				Method: sdp.QueryMethod_GET,
				Query:  *d.Value,
				Scope:  scope,
			}
		}
	}

	if query == nil {
		err = ErrNoQuery
	}

	return &sdp.LinkedItemQuery{
		Query:            query,
		BlastPropagation: bp,
	}, err
}

func getDimension(name string, dimensions []types.Dimension) *types.Dimension {
	for _, dimension := range dimensions {
		if *dimension.Name == name {
			return &dimension
		}
	}

	return nil
}
