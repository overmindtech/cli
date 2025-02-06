package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

type EKSTestClient struct {
	ListClustersOutput                   *eks.ListClustersOutput
	DescribeClusterOutput                *eks.DescribeClusterOutput
	ListAddonsOutput                     *eks.ListAddonsOutput
	DescribeAddonOutput                  *eks.DescribeAddonOutput
	ListFargateProfilesOutput            *eks.ListFargateProfilesOutput
	DescribeFargateProfileOutput         *eks.DescribeFargateProfileOutput
	ListIdentityProviderConfigsOutput    *eks.ListIdentityProviderConfigsOutput
	DescribeIdentityProviderConfigOutput *eks.DescribeIdentityProviderConfigOutput
	ListNodegroupsOutput                 *eks.ListNodegroupsOutput
	DescribeNodegroupOutput              *eks.DescribeNodegroupOutput
}

func (t EKSTestClient) ListClusters(context.Context, *eks.ListClustersInput, ...func(*eks.Options)) (*eks.ListClustersOutput, error) {
	return t.ListClustersOutput, nil
}

func (t EKSTestClient) DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error) {
	return t.DescribeClusterOutput, nil
}

func (t EKSTestClient) ListAddons(context.Context, *eks.ListAddonsInput, ...func(*eks.Options)) (*eks.ListAddonsOutput, error) {
	return t.ListAddonsOutput, nil
}

func (t EKSTestClient) DescribeAddon(ctx context.Context, params *eks.DescribeAddonInput, optFns ...func(*eks.Options)) (*eks.DescribeAddonOutput, error) {
	return t.DescribeAddonOutput, nil
}

func (t EKSTestClient) ListFargateProfiles(ctx context.Context, params *eks.ListFargateProfilesInput, optFns ...func(*eks.Options)) (*eks.ListFargateProfilesOutput, error) {
	return t.ListFargateProfilesOutput, nil
}

func (t EKSTestClient) DescribeFargateProfile(ctx context.Context, params *eks.DescribeFargateProfileInput, optFns ...func(*eks.Options)) (*eks.DescribeFargateProfileOutput, error) {
	return t.DescribeFargateProfileOutput, nil
}

func (t EKSTestClient) ListIdentityProviderConfigs(ctx context.Context, params *eks.ListIdentityProviderConfigsInput, optFns ...func(*eks.Options)) (*eks.ListIdentityProviderConfigsOutput, error) {
	return t.ListIdentityProviderConfigsOutput, nil
}

func (t EKSTestClient) DescribeIdentityProviderConfig(ctx context.Context, params *eks.DescribeIdentityProviderConfigInput, optFns ...func(*eks.Options)) (*eks.DescribeIdentityProviderConfigOutput, error) {
	return t.DescribeIdentityProviderConfigOutput, nil
}

func (t EKSTestClient) ListNodegroups(ctx context.Context, params *eks.ListNodegroupsInput, optFns ...func(*eks.Options)) (*eks.ListNodegroupsOutput, error) {
	return t.ListNodegroupsOutput, nil
}

func (t EKSTestClient) DescribeNodegroup(ctx context.Context, params *eks.DescribeNodegroupInput, optFns ...func(*eks.Options)) (*eks.DescribeNodegroupOutput, error) {
	return t.DescribeNodegroupOutput, nil
}

func eksGetAutoConfig(t *testing.T) (*eks.Client, string, string) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := eks.NewFromConfig(config)

	return client, account, region
}
