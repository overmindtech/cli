package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/rds"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func dBClusterOutputMapper(ctx context.Context, client rdsClient, scope string, _ *rds.DescribeDBClustersInput, output *rds.DescribeDBClustersOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, cluster := range output.DBClusters {
		var tags map[string]string

		// Get tags for the cluster
		tagsOut, err := client.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
			ResourceName: cluster.DBClusterArn,
		})

		if err == nil {
			tags = rdsTagsToMap(tagsOut.TagList)
		} else {
			tags = HandleTagsError(ctx, err)
		}

		attributes, err := ToAttributesWithExclude(cluster)

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "rds-db-cluster",
			UniqueAttribute: "DBClusterIdentifier",
			Attributes:      attributes,
			Scope:           scope,
			Tags:            tags,
		}

		var a *ARN

		if cluster.DBSubnetGroup != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "rds-db-subnet-group",
					Method: sdp.QueryMethod_GET,
					Query:  *cluster.DBSubnetGroup,
					Scope:  scope,
				},
			})
		}

		for _, endpoint := range []*string{cluster.Endpoint, cluster.ReaderEndpoint} {
			if endpoint != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "dns",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *endpoint,
						Scope:  "global",
					},
				})
			}
		}

		for _, replica := range cluster.ReadReplicaIdentifiers {
			if a, err = ParseARN(replica); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "rds-db-cluster",
						Method: sdp.QueryMethod_SEARCH,
						Query:  replica,
						Scope:  FormatScope(a.AccountID, a.Region),
					},
				})
			}
		}

		for _, member := range cluster.DBClusterMembers {
			if member.DBInstanceIdentifier != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "rds-db-instance",
						Method: sdp.QueryMethod_GET,
						Query:  *member.DBInstanceIdentifier,
						Scope:  scope,
					},
				})
			}
		}

		for _, sg := range cluster.VpcSecurityGroups {
			if sg.VpcSecurityGroupId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-security-group",
						Method: sdp.QueryMethod_GET,
						Query:  *sg.VpcSecurityGroupId,
						Scope:  scope,
					},
				})
			}
		}

		if cluster.HostedZoneId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "route53-hosted-zone",
					Method: sdp.QueryMethod_GET,
					Query:  *cluster.HostedZoneId,
					Scope:  scope,
				},
			})
		}

		if cluster.KmsKeyId != nil {
			if a, err = ParseARN(*cluster.KmsKeyId); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "kms-key",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *cluster.KmsKeyId,
						Scope:  FormatScope(a.AccountID, a.Region),
					},
				})
			}
		}

		if cluster.ActivityStreamKinesisStreamName != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "kinesis-stream",
					Method: sdp.QueryMethod_GET,
					Query:  *cluster.ActivityStreamKinesisStreamName,
					Scope:  scope,
				},
			})
		}

		for _, endpoint := range cluster.CustomEndpoints {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  endpoint,
					Scope:  "global",
				},
			})
		}

		for _, optionGroup := range cluster.DBClusterOptionGroupMemberships {
			if optionGroup.DBClusterOptionGroupName != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "rds-option-group",
						Method: sdp.QueryMethod_GET,
						Query:  *optionGroup.DBClusterOptionGroupName,
						Scope:  scope,
					},
				})
			}
		}

		if cluster.MasterUserSecret != nil {
			if cluster.MasterUserSecret.KmsKeyId != nil {
				if a, err = ParseARN(*cluster.MasterUserSecret.KmsKeyId); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "kms-key",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *cluster.MasterUserSecret.KmsKeyId,
							Scope:  FormatScope(a.AccountID, a.Region),
						},
					})
				}
			}

			if cluster.MasterUserSecret.SecretArn != nil {
				if a, err = ParseARN(*cluster.MasterUserSecret.SecretArn); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "secretsmanager-secret",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *cluster.MasterUserSecret.SecretArn,
							Scope:  FormatScope(a.AccountID, a.Region),
						},
					})
				}
			}
		}

		if cluster.MonitoringRoleArn != nil {
			if a, err = ParseARN(*cluster.MonitoringRoleArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "iam-role",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *cluster.MonitoringRoleArn,
						Scope:  FormatScope(a.AccountID, a.Region),
					},
				})
			}
		}

		if cluster.PerformanceInsightsKMSKeyId != nil {
			// This is an ARN
			if a, err = ParseARN(*cluster.PerformanceInsightsKMSKeyId); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "kms-key",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *cluster.PerformanceInsightsKMSKeyId,
						Scope:  FormatScope(a.AccountID, a.Region),
					},
				})
			}
		}

		if cluster.ReplicationSourceIdentifier != nil {
			if a, err = ParseARN(*cluster.ReplicationSourceIdentifier); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "rds-db-cluster",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *cluster.ReplicationSourceIdentifier,
						Scope:  FormatScope(a.AccountID, a.Region),
					},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewRDSDBClusterAdapter(client rdsClient, accountID string, region string, cache sdpcache.Cache) *DescribeOnlyAdapter[*rds.DescribeDBClustersInput, *rds.DescribeDBClustersOutput, rdsClient, *rds.Options] {
	return &DescribeOnlyAdapter[*rds.DescribeDBClustersInput, *rds.DescribeDBClustersOutput, rdsClient, *rds.Options]{
		ItemType:        "rds-db-cluster",
		Region:          region,
		AccountID:       accountID,
		Client:          client,
		AdapterMetadata: dbClusterAdapterMetadata,
		cache:        cache,
		PaginatorBuilder: func(client rdsClient, params *rds.DescribeDBClustersInput) Paginator[*rds.DescribeDBClustersOutput, *rds.Options] {
			return rds.NewDescribeDBClustersPaginator(client, params)
		},
		DescribeFunc: func(ctx context.Context, client rdsClient, input *rds.DescribeDBClustersInput) (*rds.DescribeDBClustersOutput, error) {
			return client.DescribeDBClusters(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*rds.DescribeDBClustersInput, error) {
			return &rds.DescribeDBClustersInput{
				DBClusterIdentifier: &query,
			}, nil
		},
		InputMapperList: func(scope string) (*rds.DescribeDBClustersInput, error) {
			return &rds.DescribeDBClustersInput{}, nil
		},
		OutputMapper: dBClusterOutputMapper,
	}
}

var dbClusterAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "rds-db-cluster",
	DescriptiveName: "RDS Cluster",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a parameter group by name",
		ListDescription:   "List all RDS parameter groups",
		SearchDescription: "Search for a parameter group by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_rds_cluster.cluster_identifier"},
	},
	PotentialLinks: []string{"rds-db-subnet-group", "dns", "rds-db-cluster", "ec2-security-group", "route53-hosted-zone", "kms-key", "kinesis-stream", "rds-option-group", "secretsmanager-secret", "iam-role"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
})
