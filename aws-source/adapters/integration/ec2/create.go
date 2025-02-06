package ec2

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func createEC2Instance(ctx context.Context, logger *slog.Logger, client *ec2.Client, testID string) error {
	// check if a resource with the same tags already exists
	id, err := findActiveInstanceIDByTags(ctx, client)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating EC2 instance")
		} else {
			return err
		}
	}

	if id != nil {
		logger.InfoContext(ctx, "EC2 instance already exists")
		return nil
	}

	// Search for the latest AMI for Amazon Linux. We can't hardcode this as the
	// AMI for the same image differs per-region
	images, err := client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name: aws.String("name"),
				Values: []string{
					"amzn2-ami-hvm-2.0.*-x86_64-gp2",
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to describe images: %w", err)
	}

	if len(images.Images) == 0 {
		return errors.New("no images found")
	}

	// We need to select a subnet since we can't rely on having a default VPC
	subnets, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{})
	if err != nil {
		return fmt.Errorf("failed to describe subnets: %w", err)
	}

	if len(subnets.Subnets) == 0 {
		return errors.New("no subnets found")
	}

	input := &ec2.RunInstancesInput{
		DryRun: aws.Bool(false),
		// `Subscribe Now` is selected on marketplace UI
		ImageId:      images.Images[0].ImageId,
		SubnetId:     subnets.Subnets[0].SubnetId,
		InstanceType: types.InstanceTypeT3Nano,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				// TODO: Create a convenience function to add shared tags to the resources
				Tags: resourceTags(instanceSrc, testID),
			},
		},
	}

	result, err := client.RunInstances(ctx, input)
	if err != nil {
		return err
	}

	waiter := ec2.NewInstanceRunningWaiter(client)
	err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{*result.Instances[0].InstanceId},
	},
		5*time.Minute)
	if err != nil {
		return err
	}

	return nil
}
