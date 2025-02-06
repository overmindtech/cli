package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/rds"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
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
			tags = adapterhelpers.HandleTagsError(ctx, err)
		}

		attributes, err := adapterhelpers.ToAttributesWithExclude(cluster)

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

		var a *adapterhelpers.ARN

		if cluster.DBSubnetGroup != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "rds-db-subnet-group",
					Method: sdp.QueryMethod_GET,
					Query:  *cluster.DBSubnetGroup,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Tightly coupled
					In:  true,
					Out: false,
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
					BlastPropagation: &sdp.BlastPropagation{
						// DNS always linked
						In:  true,
						Out: true,
					},
				})
			}
		}

		for _, replica := range cluster.ReadReplicaIdentifiers {
			if a, err = adapterhelpers.ParseARN(replica); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "rds-db-cluster",
						Method: sdp.QueryMethod_SEARCH,
						Query:  replica,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Tightly coupled
						In:  true,
						Out: true,
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
					BlastPropagation: &sdp.BlastPropagation{
						// Tightly coupled
						In:  true,
						Out: true,
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
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the security group can affect the cluster
						In: true,
						// The cluster won't affect the security group
						Out: false,
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
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the hosted zone can affect the cluster
					In: true,
					// The cluster won't affect the hosted zone
					Out: false,
				},
			})
		}

		if cluster.KmsKeyId != nil {
			if a, err = adapterhelpers.ParseARN(*cluster.KmsKeyId); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "kms-key",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *cluster.KmsKeyId,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the KMS key can affect the cluster
						In: true,
						// The cluster won't affect the KMS key
						Out: false,
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
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the Kinesis stream can affect the cluster
					In: true,
					// Changes to the cluster can affect the Kinesis stream
					Out: true,
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
				BlastPropagation: &sdp.BlastPropagation{
					// DNS always linked
					In:  true,
					Out: true,
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
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the option group can affect the cluster
						In: true,
						// Changes to the cluster won't affect the option group
						Out: false,
					},
				})
			}
		}

		if cluster.MasterUserSecret != nil {
			if cluster.MasterUserSecret.KmsKeyId != nil {
				if a, err = adapterhelpers.ParseARN(*cluster.MasterUserSecret.KmsKeyId); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "kms-key",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *cluster.MasterUserSecret.KmsKeyId,
							Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changes to the KMS key can affect the cluster
							In: true,
							// The cluster won't affect the KMS key
							Out: false,
						},
					})
				}
			}

			if cluster.MasterUserSecret.SecretArn != nil {
				if a, err = adapterhelpers.ParseARN(*cluster.MasterUserSecret.SecretArn); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "secretsmanager-secret",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *cluster.MasterUserSecret.SecretArn,
							Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changes to the secret can affect the cluster
							In: true,
							// The cluster won't affect the secret
							Out: false,
						},
					})
				}
			}
		}

		if cluster.MonitoringRoleArn != nil {
			if a, err = adapterhelpers.ParseARN(*cluster.MonitoringRoleArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "iam-role",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *cluster.MonitoringRoleArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the IAM role can affect the cluster
						In: true,
						// The cluster won't affect the IAM role
						Out: false,
					},
				})
			}
		}

		if cluster.PerformanceInsightsKMSKeyId != nil {
			// This is an ARN
			if a, err = adapterhelpers.ParseARN(*cluster.PerformanceInsightsKMSKeyId); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "kms-key",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *cluster.PerformanceInsightsKMSKeyId,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the KMS key can affect the cluster
						In: true,
						// The cluster won't affect the KMS key
						Out: false,
					},
				})
			}
		}

		if cluster.ReplicationSourceIdentifier != nil {
			if a, err = adapterhelpers.ParseARN(*cluster.ReplicationSourceIdentifier); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "rds-db-cluster",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *cluster.ReplicationSourceIdentifier,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Tightly coupled
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

func NewRDSDBClusterAdapter(client rdsClient, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*rds.DescribeDBClustersInput, *rds.DescribeDBClustersOutput, rdsClient, *rds.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*rds.DescribeDBClustersInput, *rds.DescribeDBClustersOutput, rdsClient, *rds.Options]{
		ItemType:        "rds-db-cluster",
		Region:          region,
		AccountID:       accountID,
		Client:          client,
		AdapterMetadata: dbClusterAdapterMetadata,
		PaginatorBuilder: func(client rdsClient, params *rds.DescribeDBClustersInput) adapterhelpers.Paginator[*rds.DescribeDBClustersOutput, *rds.Options] {
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
