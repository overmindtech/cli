package adapters

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/eks"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func fargateProfileGetFunc(ctx context.Context, client EKSClient, scope string, input *eks.DescribeFargateProfileInput) (*sdp.Item, error) {
	out, err := client.DescribeFargateProfile(ctx, input)

	if err != nil {
		return nil, err
	}

	if out.FargateProfile == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "fargate profile was nil",
		}
	}

	attributes, err := adapterhelpers.ToAttributesWithExclude(out.FargateProfile)

	if err != nil {
		return nil, err
	}

	// The uniqueAttributeValue for this is a custom field:
	// {clusterName}:{FargateProfileName}
	attributes.Set("UniqueName", (*out.FargateProfile.ClusterName + ":" + *out.FargateProfile.FargateProfileName))

	item := sdp.Item{
		Type:            "eks-fargate-profile",
		UniqueAttribute: "UniqueName",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            out.FargateProfile.Tags,
	}

	if out.FargateProfile.PodExecutionRoleArn != nil {
		if a, err := adapterhelpers.ParseARN(*out.FargateProfile.PodExecutionRoleArn); err == nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "iam-role",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *out.FargateProfile.PodExecutionRoleArn,
					Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// The execution role will affect the fargate profile
					In: true,
					// The fargate profile can't affect the execution role
					Out: false,
				},
			})
		}
	}

	for _, subnet := range out.FargateProfile.Subnets {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ec2-subnet",
				Method: sdp.QueryMethod_GET,
				Query:  subnet,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// The subnet will affect the fargate profile
				In: true,
				// The fargate profile can't affect the subnet
				Out: false,
			},
		})
	}

	return &item, nil
}

func NewEKSFargateProfileAdapter(client EKSClient, accountID string, region string) *adapterhelpers.AlwaysGetAdapter[*eks.ListFargateProfilesInput, *eks.ListFargateProfilesOutput, *eks.DescribeFargateProfileInput, *eks.DescribeFargateProfileOutput, EKSClient, *eks.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*eks.ListFargateProfilesInput, *eks.ListFargateProfilesOutput, *eks.DescribeFargateProfileInput, *eks.DescribeFargateProfileOutput, EKSClient, *eks.Options]{
		ItemType:         "eks-fargate-profile",
		Client:           client,
		AccountID:        accountID,
		Region:           region,
		DisableList:      true,
		AlwaysSearchARNs: true,
		AdapterMetadata:  fargateProfileAdapterMetadata,
		SearchInputMapper: func(scope, query string) (*eks.ListFargateProfilesInput, error) {
			return &eks.ListFargateProfilesInput{
				ClusterName: &query,
			}, nil
		},
		GetInputMapper: func(scope, query string) *eks.DescribeFargateProfileInput {
			// The uniqueAttributeValue for this is a custom field:
			// {clusterName}/{FargateProfileName}
			fields := strings.Split(query, ":")

			var clusterName string
			var FargateProfileName string

			if len(fields) == 2 {
				clusterName = fields[0]
				FargateProfileName = fields[1]
			}

			return &eks.DescribeFargateProfileInput{
				FargateProfileName: &FargateProfileName,
				ClusterName:        &clusterName,
			}
		},
		ListFuncPaginatorBuilder: func(client EKSClient, input *eks.ListFargateProfilesInput) adapterhelpers.Paginator[*eks.ListFargateProfilesOutput, *eks.Options] {
			return eks.NewListFargateProfilesPaginator(client, input)
		},
		ListFuncOutputMapper: func(output *eks.ListFargateProfilesOutput, input *eks.ListFargateProfilesInput) ([]*eks.DescribeFargateProfileInput, error) {
			inputs := make([]*eks.DescribeFargateProfileInput, 0, len(output.FargateProfileNames))

			for i := range output.FargateProfileNames {
				inputs = append(inputs, &eks.DescribeFargateProfileInput{
					ClusterName:        input.ClusterName,
					FargateProfileName: &output.FargateProfileNames[i],
				})
			}

			return inputs, nil
		},
		GetFunc: fargateProfileGetFunc,
	}
}

var fargateProfileAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "eks-fargate-profile",
	DescriptiveName: "Fargate Profile",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get a fargate profile by unique name ({clusterName}:{FargateProfileName})",
		SearchDescription: "Search for fargate profiles by cluster name",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_eks_fargate_profile.id",
			TerraformMethod:   sdp.QueryMethod_GET,
		},
	},
	PotentialLinks: []string{"iam-role", "ec2-subnet"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
})
