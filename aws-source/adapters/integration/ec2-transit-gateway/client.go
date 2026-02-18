package ec2transitgateway

import (
	"context"
	"fmt"

	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func ec2Client(ctx context.Context) (*awsec2.Client, error) {
	testAWSConfig, err := integration.AWSSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS settings: %w", err)
	}
	return awsec2.NewFromConfig(testAWSConfig.Config), nil
}
