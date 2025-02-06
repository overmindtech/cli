package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

// ImageInputMapperGet Gets a given image. As opposed to list, get will get
// details of any image given a correct ID, not just images owned by the current
// account
func imageInputMapperGet(scope string, query string) (*ec2.DescribeImagesInput, error) {
	return &ec2.DescribeImagesInput{
		ImageIds: []string{
			query,
		},
	}, nil
}

// ImageInputMapperList Lists images that are owned by the current account, as
// opposed to all available images since this is simply way too much data
func imageInputMapperList(scope string) (*ec2.DescribeImagesInput, error) {
	return &ec2.DescribeImagesInput{
		Owners: []string{
			// Avoid getting every image in existence, just get the ones
			// relevant to this scope i.e. owned by this account in this region
			"self",
		},
	}, nil
}

func imageOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeImagesInput, output *ec2.DescribeImagesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, image := range output.Images {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(image, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-image",
			UniqueAttribute: "ImageId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(image.Tags),
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2ImageAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeImagesInput, *ec2.DescribeImagesOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeImagesInput, *ec2.DescribeImagesOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-image",
		AdapterMetadata: imageAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error) {
			return client.DescribeImages(ctx, input)
		},
		InputMapperGet:  imageInputMapperGet,
		InputMapperList: imageInputMapperList,
		OutputMapper:    imageOutputMapper,
	}
}

var imageAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-image",
	DescriptiveName: "Amazon Machine Image (AMI)",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an AMI by ID",
		ListDescription:   "List all AMIs",
		SearchDescription: "Search AMIs by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_ami.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
})
