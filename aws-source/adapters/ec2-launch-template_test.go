package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestLaunchTemplateInputMapperGet(t *testing.T) {
	input, err := launchTemplateInputMapperGet("foo", "bar")

	if err != nil {
		t.Error(err)
	}

	if len(input.LaunchTemplateIds) != 1 {
		t.Fatalf("expected 1 LaunchTemplate ID, got %v", len(input.LaunchTemplateIds))
	}

	if input.LaunchTemplateIds[0] != "bar" {
		t.Errorf("expected LaunchTemplate ID to be bar, got %v", input.LaunchTemplateIds[0])
	}
}

func TestLaunchTemplateInputMapperList(t *testing.T) {
	input, err := launchTemplateInputMapperList("foo")

	if err != nil {
		t.Error(err)
	}

	if len(input.Filters) != 0 || len(input.LaunchTemplateIds) != 0 {
		t.Errorf("non-empty input: %v", input)
	}
}

func TestLaunchTemplateOutputMapper(t *testing.T) {
	output := &ec2.DescribeLaunchTemplatesOutput{
		LaunchTemplates: []types.LaunchTemplate{
			{
				CreateTime:           adapterhelpers.PtrTime(time.Now()),
				CreatedBy:            adapterhelpers.PtrString("me"),
				DefaultVersionNumber: adapterhelpers.PtrInt64(1),
				LatestVersionNumber:  adapterhelpers.PtrInt64(10),
				LaunchTemplateId:     adapterhelpers.PtrString("id"),
				LaunchTemplateName:   adapterhelpers.PtrString("hello"),
				Tags:                 []types.Tag{},
			},
		},
	}

	items, err := launchTemplateOutputMapper(context.Background(), nil, "foo", nil, output)

	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", len(items))
	}

}

func TestNewEC2LaunchTemplateAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2LaunchTemplateAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
