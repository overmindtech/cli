package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestBackupPolicyOutputMapper(t *testing.T) {
	output := &efs.DescribeBackupPolicyOutput{
		BackupPolicy: &types.BackupPolicy{
			Status: types.StatusEnabled,
		},
	}

	items, err := BackupPolicyOutputMapper(context.Background(), nil, "foo", &efs.DescribeBackupPolicyInput{
		FileSystemId: adapterhelpers.PtrString("fs-1234"),
	}, output)

	if err != nil {
		t.Fatal(err)
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Error(err)
		}
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", len(items))
	}
}
