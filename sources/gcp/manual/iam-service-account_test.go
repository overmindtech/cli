package manual_test

import (
	"context"
	"testing"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestIAMServiceAccount(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockIAMServiceAccountClient(ctrl)
	projectID := "test-project-id"

	testUniqueID := "1234567890"
	testEmail := "test-sa@test-project-id.iam.gserviceaccount.com"
	testDisplayName := "Test Service Account"

	t.Run("Get by unique_id", func(t *testing.T) {
		wrapper := manual.NewIAMServiceAccount(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createServiceAccount(testUniqueID, testEmail, testDisplayName, projectID, false), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], testUniqueID, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:             gcpshared.CloudResourceManagerProject.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-project-id",
					ExpectedScope:            "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
				},
				{
					ExpectedType:             gcpshared.IAMServiceAccountKey.String(),
					ExpectedMethod:           sdp.QueryMethod_SEARCH,
					ExpectedQuery:            "test-service-account-id",
					ExpectedScope:            "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: false, Out: true},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get by email", func(t *testing.T) {
		wrapper := manual.NewIAMServiceAccount(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createServiceAccount(testUniqueID, testEmail, testDisplayName, projectID, false), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], testEmail, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}
		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:             gcpshared.CloudResourceManagerProject.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            "test-project-id",
					ExpectedScope:            "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
				},
				{
					ExpectedType:             gcpshared.IAMServiceAccountKey.String(),
					ExpectedMethod:           sdp.QueryMethod_SEARCH,
					ExpectedQuery:            "test-service-account-id",
					ExpectedScope:            "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{In: false, Out: true},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewIAMServiceAccount(mockClient, projectID)
		adapter := sources.WrapperToAdapter(wrapper)

		mockIterator := mocks.NewMockIAMServiceAccountIterator(ctrl)

		mockIterator.EXPECT().Next().Return(createServiceAccount("111", "sa1@test-project-id.iam.gserviceaccount.com", "SA 1", projectID, false), nil)
		mockIterator.EXPECT().Next().Return(createServiceAccount("222", "sa2@test-project-id.iam.gserviceaccount.com", "SA 2", projectID, true), nil)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockIterator)

		sdpItems, err := adapter.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		expectedCount := 2
		actualCount := len(sdpItems)
		if actualCount != expectedCount {
			t.Fatalf("Expected %d items, got: %d", expectedCount, actualCount)
		}

		for _, item := range sdpItems {
			if err := item.Validate(); err != nil {
				t.Fatalf("Expected no validation error, got: %v", err)
			}
		}
	})

}

// createServiceAccount creates a ServiceAccount with the specified fields.
func createServiceAccount(uniqueID, email, displayName, projectID string, disabled bool) *adminpb.ServiceAccount {
	return &adminpb.ServiceAccount{
		UniqueId:    uniqueID,
		Email:       email,
		DisplayName: displayName,
		Disabled:    disabled,
		ProjectId:   projectID,
		Name:        "projects/test-project-id/serviceAccounts/test-service-account-id",
	}
}
