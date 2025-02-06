package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func clusterGetFunc(ctx context.Context, client EKSClient, scope string, input *eks.DescribeClusterInput) (*sdp.Item, error) {
	output, err := client.DescribeCluster(ctx, input)

	if err != nil {
		return nil, err
	}

	if output.Cluster == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "cluster response was nil",
		}
	}

	cluster := output.Cluster

	attributes, err := adapterhelpers.ToAttributesWithExclude(cluster, "clientRequestToken")

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "eks-cluster",
		UniqueAttribute: "Name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            cluster.Tags,
		LinkedItemQueries: []*sdp.LinkedItemQuery{
			{
				Query: &sdp.Query{
					Type:   "eks-addon",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *cluster.Name,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// These are tightly linked
					In:  true,
					Out: true,
				},
			},
			{
				Query: &sdp.Query{
					Type:   "eks-fargate-profile",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *cluster.Name,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// These are tightly linked
					In:  true,
					Out: true,
				},
			},
			{
				Query: &sdp.Query{
					Type:   "eks-nodegroup",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *cluster.Name,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// These are tightly linked
					In:  true,
					Out: true,
				},
			},
		},
	}

	switch cluster.Status {
	case types.ClusterStatusCreating:
		item.Health = sdp.Health_HEALTH_PENDING.Enum()
	case types.ClusterStatusActive:
		item.Health = sdp.Health_HEALTH_OK.Enum()
	case types.ClusterStatusDeleting:
		item.Health = sdp.Health_HEALTH_WARNING.Enum()
	case types.ClusterStatusFailed:
		item.Health = sdp.Health_HEALTH_ERROR.Enum()
	case types.ClusterStatusUpdating:
		item.Health = sdp.Health_HEALTH_PENDING.Enum()
	case types.ClusterStatusPending:
		item.Health = sdp.Health_HEALTH_PENDING.Enum()
	}

	var a *adapterhelpers.ARN

	if cluster.ConnectorConfig != nil {
		if cluster.ConnectorConfig.RoleArn != nil {
			if a, err = adapterhelpers.ParseARN(*cluster.ConnectorConfig.RoleArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "iam-role",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *cluster.ConnectorConfig.RoleArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// The role can affect the cluster
						In: true,
						// The cluster can't affect the role
						Out: false,
					},
				})
			}
		}
	}

	for _, conf := range cluster.EncryptionConfig {
		if conf.Provider != nil {
			if conf.Provider.KeyArn != nil {
				if a, err = adapterhelpers.ParseARN(*conf.Provider.KeyArn); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "kms-key",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *conf.Provider.KeyArn,
							Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
						},
						BlastPropagation: &sdp.BlastPropagation{
							// The key can affect the cluster
							In: true,
							// The cluster can't affect the key
							Out: false,
						},
					})
				}
			}
		}
	}

	if cluster.Endpoint != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "http",
				Method: sdp.QueryMethod_GET,
				Query:  *cluster.Endpoint,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// HTTP should be linked bidirectionally
				In:  true,
				Out: true,
			},
		})
	}

	if cluster.ResourcesVpcConfig != nil {
		if cluster.ResourcesVpcConfig.ClusterSecurityGroupId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-security-group",
					Method: sdp.QueryMethod_GET,
					Query:  *cluster.ResourcesVpcConfig.ClusterSecurityGroupId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// The SG can affect the cluster
					In: true,
					// The cluster can't affect the SG
					Out: false,
				},
			})
		}

		for _, id := range cluster.ResourcesVpcConfig.SecurityGroupIds {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-security-group",
					Method: sdp.QueryMethod_GET,
					Query:  id,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// The SG can affect the cluster
					In: true,
					// The cluster can't affect the SG
					Out: false,
				},
			})
		}

		for _, id := range cluster.ResourcesVpcConfig.SubnetIds {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-subnet",
					Method: sdp.QueryMethod_GET,
					Query:  id,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// The subnet can affect the cluster
					In: true,
					// The cluster can't affect the subnet
					Out: false,
				},
			})
		}

		if cluster.ResourcesVpcConfig.VpcId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *cluster.ResourcesVpcConfig.VpcId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// The VPC can affect the cluster
					In: true,
					// The cluster can't affect the VPC
					Out: false,
				},
			})
		}
	}

	if cluster.RoleArn != nil {
		if a, err = adapterhelpers.ParseARN(*cluster.RoleArn); err == nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "iam-role",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *cluster.RoleArn,
					Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// The role can affect the cluster
					In: true,
					// The cluster can't affect the role
					Out: false,
				},
			})
		}
	}

	return &item, nil

}

func NewEKSClusterAdapter(client EKSClient, accountID string, region string) *adapterhelpers.AlwaysGetAdapter[*eks.ListClustersInput, *eks.ListClustersOutput, *eks.DescribeClusterInput, *eks.DescribeClusterOutput, EKSClient, *eks.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*eks.ListClustersInput, *eks.ListClustersOutput, *eks.DescribeClusterInput, *eks.DescribeClusterOutput, EKSClient, *eks.Options]{
		ItemType:        "eks-cluster",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: eksClusterAdapterMetadata,
		ListInput:       &eks.ListClustersInput{},
		GetInputMapper: func(scope, query string) *eks.DescribeClusterInput {
			return &eks.DescribeClusterInput{
				Name: &query,
			}
		},
		ListFuncPaginatorBuilder: func(client EKSClient, input *eks.ListClustersInput) adapterhelpers.Paginator[*eks.ListClustersOutput, *eks.Options] {
			return eks.NewListClustersPaginator(client, input)
		},
		ListFuncOutputMapper: func(output *eks.ListClustersOutput, _ *eks.ListClustersInput) ([]*eks.DescribeClusterInput, error) {
			inputs := make([]*eks.DescribeClusterInput, 0, len(output.Clusters))

			for i := range output.Clusters {
				inputs = append(inputs, &eks.DescribeClusterInput{
					Name: &output.Clusters[i],
				})
			}

			return inputs, nil
		},
		GetFunc: clusterGetFunc,
	}
}

var eksClusterAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "eks-cluster",
	DescriptiveName: "EKS Cluster",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a cluster by name",
		ListDescription:   "List all clusters",
		SearchDescription: "Search for clusters by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_eks_cluster.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
})
