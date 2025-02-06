package ec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func deleteInstance(ctx context.Context, client *ec2.Client, instanceID string) error {
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	}

	_, err := client.TerminateInstances(ctx, input)
	return err
}
