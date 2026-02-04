package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type ecsTestClient struct{}

func ecsGetAutoConfig(t *testing.T) (*ecs.Client, string, string) {
	config, account, region := GetAutoConfig(t)
	client := ecs.NewFromConfig(config)

	return client, account, region
}
