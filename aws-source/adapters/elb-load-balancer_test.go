package adapters

import (
	"context"
	"testing"
	"time"

	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"

	"github.com/overmindtech/cli/go/sdp-go"
)

type mockElbClient struct{}

func (m mockElbClient) DescribeTags(ctx context.Context, params *elb.DescribeTagsInput, optFns ...func(*elb.Options)) (*elb.DescribeTagsOutput, error) {
	return &elb.DescribeTagsOutput{
		TagDescriptions: []types.TagDescription{
			{
				LoadBalancerName: new("a8c3c8851f0df43fda89797c8e941a91"),
				Tags: []types.Tag{
					{
						Key:   new("foo"),
						Value: new("bar"),
					},
				},
			},
		},
	}, nil
}

func (m mockElbClient) DescribeLoadBalancers(ctx context.Context, params *elb.DescribeLoadBalancersInput, optFns ...func(*elb.Options)) (*elb.DescribeLoadBalancersOutput, error) {
	return nil, nil
}

func TestELBv2LoadBalancerOutputMapper(t *testing.T) {
	output := &elb.DescribeLoadBalancersOutput{
		LoadBalancerDescriptions: []types.LoadBalancerDescription{
			{
				LoadBalancerName:          new("a8c3c8851f0df43fda89797c8e941a91"),
				DNSName:                   new("a8c3c8851f0df43fda89797c8e941a91-182843316.eu-west-2.elb.amazonaws.com"), // link
				CanonicalHostedZoneName:   new("a8c3c8851f0df43fda89797c8e941a91-182843316.eu-west-2.elb.amazonaws.com"), // link
				CanonicalHostedZoneNameID: new("ZHURV8PSTC4K8"),                                                          // link
				ListenerDescriptions: []types.ListenerDescription{
					{
						Listener: &types.Listener{
							Protocol:         new("TCP"),
							LoadBalancerPort: 7687,
							InstanceProtocol: new("TCP"),
							InstancePort:     new(int32(30133)),
						},
						PolicyNames: []string{},
					},
					{
						Listener: &types.Listener{
							Protocol:         new("TCP"),
							LoadBalancerPort: 7473,
							InstanceProtocol: new("TCP"),
							InstancePort:     new(int32(31459)),
						},
						PolicyNames: []string{},
					},
					{
						Listener: &types.Listener{
							Protocol:         new("TCP"),
							LoadBalancerPort: 7474,
							InstanceProtocol: new("TCP"),
							InstancePort:     new(int32(30761)),
						},
						PolicyNames: []string{},
					},
				},
				Policies: &types.Policies{
					AppCookieStickinessPolicies: []types.AppCookieStickinessPolicy{
						{
							CookieName: new("foo"),
							PolicyName: new("policy"),
						},
					},
					LBCookieStickinessPolicies: []types.LBCookieStickinessPolicy{
						{
							CookieExpirationPeriod: new(int64(10)),
							PolicyName:             new("name"),
						},
					},
					OtherPolicies: []string{},
				},
				BackendServerDescriptions: []types.BackendServerDescription{
					{
						InstancePort: new(int32(443)),
						PolicyNames:  []string{},
					},
				},
				AvailabilityZones: []string{ // link
					"euwest-2b",
					"euwest-2a",
					"euwest-2c",
				},
				Subnets: []string{ // link
					"subnet0960234bbc4edca03",
					"subnet09d5f6fa75b0b4569",
					"subnet0e234bef35fc4a9e1",
				},
				VPCId: new("vpc-0c72199250cd479ea"), // link
				Instances: []types.Instance{
					{
						InstanceId: new("i-0337802d908b4a81e"), // link *2 to ec2-instance and health
					},
				},
				HealthCheck: &types.HealthCheck{
					Target:             new("HTTP:31151/healthz"),
					Interval:           new(int32(10)),
					Timeout:            new(int32(5)),
					UnhealthyThreshold: new(int32(6)),
					HealthyThreshold:   new(int32(2)),
				},
				SourceSecurityGroup: &types.SourceSecurityGroup{
					OwnerAlias: new("944651592624"),
					GroupName:  new("k8s-elb-a8c3c8851f0df43fda89797c8e941a91"), // link
				},
				SecurityGroups: []string{
					"sg097e3cfdfc6d53b77", // link
				},
				CreatedTime: new(time.Now()),
				Scheme:      new("internet-facing"),
			},
		},
	}

	items, err := elbLoadBalancerOutputMapper(context.Background(), mockElbClient{}, "foo", nil, output)

	if err != nil {
		t.Error(err)
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Error(err)
		}
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", len(items))
	}

	item := items[0]

	if item.GetTags()["foo"] != "bar" {
		t.Errorf("expected tag foo to be bar, got %v", item.GetTags()["foo"])
	}

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := QueryTests{
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "a8c3c8851f0df43fda89797c8e941a91-182843316.eu-west-2.elb.amazonaws.com",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "a8c3c8851f0df43fda89797c8e941a91-182843316.eu-west-2.elb.amazonaws.com",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "route53-hosted-zone",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "ZHURV8PSTC4K8",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "subnet0960234bbc4edca03",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "subnet09d5f6fa75b0b4569",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "subnet0e234bef35fc4a9e1",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-vpc",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpc-0c72199250cd479ea",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-instance",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "i-0337802d908b4a81e",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "elb-instance-health",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "a8c3c8851f0df43fda89797c8e941a91/i-0337802d908b4a81e",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-security-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "k8s-elb-a8c3c8851f0df43fda89797c8e941a91",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-security-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "sg097e3cfdfc6d53b77",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}
