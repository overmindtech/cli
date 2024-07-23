package cmd

import (
	"fmt"
	"testing"

	"github.com/overmindtech/cli/tfutils"
)

func TestPrintProviderResult(t *testing.T) {
	results := []tfutils.ProviderResult{
		{
			Provider: &tfutils.AWSProvider{
				Name:      "aws",
				AccessKey: "OOH SECRET",
				SecretKey: "EVEN MORE SECRET",
				Region:    "us-west-2",
			},
			FilePath: "/path/to/aws.tf",
		},
		{
			Error:    fmt.Errorf("failed to parse provider"),
			FilePath: "/path/to/bad.tf",
		},
		{
			Provider: &tfutils.AWSProvider{
				Alias:  "dev",
				Region: "us-west-2",
				SharedConfigFiles: []string{
					"/path/to/credentials",
				},
				AssumeRole: &tfutils.AssumeRole{
					Duration:   "12h",
					ExternalID: "external-id",
					Policy:     "policy",
					RoleARN:    "arn:aws:iam::123456789012:role/role-name",
				},
			},
		},
	}

	for _, result := range results {
		// This doesn't test anything, it's just used to visually confirm the
		// results in the debug window
		str := renderProviderResult(result, 0)
		for _, line := range str {
			fmt.Println(line)
		}
	}
}
