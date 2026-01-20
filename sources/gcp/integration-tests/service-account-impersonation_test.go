package integrationtests

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	credentials "cloud.google.com/go/iam/credentials/apiv1"
	credentialspb "cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"github.com/google/uuid"
	"github.com/googleapis/gax-go/v2/apierror"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

// Test state structure to hold service account information
type testState struct {
	projectID                   string
	ourServiceAccountID         string
	ourServiceAccountEmail      string
	ourServiceAccountKey        []byte
	ourServiceAccountKeyID      string
	customerServiceAccountID    string
	customerServiceAccountEmail string
	customerServiceAccountKey   []byte
	customerServiceAccountKeyID string
}

func TestServiceAccountImpersonationIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	t.Parallel()

	// Initialize Cloud Resource Manager service
	crmService, err := cloudresourcemanager.NewService(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Cloud Resource Manager service: %v", err)
	}

	state := &testState{
		projectID: projectID,
	}

	// Initialize IAM service using Application Default Credentials
	iamService, err := iam.NewService(t.Context())
	if err != nil {
		t.Fatalf("Failed to create IAM service: %v", err)
	}

	// Create UUIDs for service account names
	ourSAUUID := uuid.New().String()
	customerSAUUID := uuid.New().String()

	// Generate service account IDs (max 30 chars, must be alphanumeric and lowercase)
	// Remove hyphens and take first part of UUID
	state.ourServiceAccountID = fmt.Sprintf("ovm-test-our-sa-%s", strings.ReplaceAll(ourSAUUID[:8], "-", ""))
	state.customerServiceAccountID = fmt.Sprintf("ovm-test-cust-%s", strings.ReplaceAll(customerSAUUID[:8], "-", ""))

	// since this test needs to keep state between tests, we wrap it in a Run function
	t.Run("Run", func(t *testing.T) {
		setupTest(t, t.Context(), iamService, crmService, state)

		t.Cleanup(func() {
			teardownTest(t, t.Context(), iamService, crmService, state)
		})

		t.Run("Test1_OurServiceAccountDirectAuth", func(t *testing.T) {
			testOurServiceAccountDirectAuth(t, t.Context(), state)
		})

		t.Run("Test2_CustomerServiceAccountDirectAuth", func(t *testing.T) {
			testCustomerServiceAccountDirectAuth(t, t.Context(), state)
		})

		t.Run("Test3_Impersonation", func(t *testing.T) {
			testImpersonation(t, t.Context(), state)
		})

	})
}

func setupTest(t *testing.T, ctx context.Context, iamService *iam.Service, crmService *cloudresourcemanager.Service, state *testState) {
	// Create "Our Service Account"
	t.Logf("Creating 'Our Service Account': %s", state.ourServiceAccountID)
	ourSA, err := createServiceAccount(ctx, iamService, state.projectID, state.ourServiceAccountID, "Our Service Account for impersonation test")
	if err != nil {
		t.Fatalf("Failed to create 'Our Service Account': %v", err)
	}
	state.ourServiceAccountEmail = ourSA.Email
	t.Logf("Created 'Our Service Account': %s", state.ourServiceAccountEmail)

	// Create "Customer Service Account"
	t.Logf("Creating 'Customer Service Account': %s", state.customerServiceAccountID)
	customerSA, err := createServiceAccount(ctx, iamService, state.projectID, state.customerServiceAccountID, "Customer Service Account for impersonation test")
	if err != nil {
		t.Fatalf("Failed to create 'Customer Service Account': %v", err)
	}
	state.customerServiceAccountEmail = customerSA.Email
	t.Logf("Created 'Customer Service Account': %s", state.customerServiceAccountEmail)

	// Verify service accounts are created
	t.Log("Verifying service accounts are created...")
	maxAttempts := 30
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ourSAVerified, err := verifyServiceAccountExists(ctx, iamService, state.projectID, state.ourServiceAccountEmail)
		if err != nil {
			t.Logf("Attempt %d/%d: Error verifying 'Our Service Account': %v", attempt, maxAttempts, err)
		}
		customerSAVerified, err := verifyServiceAccountExists(ctx, iamService, state.projectID, state.customerServiceAccountEmail)
		if err != nil {
			t.Logf("Attempt %d/%d: Error verifying 'Customer Service Account': %v", attempt, maxAttempts, err)
		}

		if ourSAVerified && customerSAVerified {
			t.Logf("✓ Service accounts verified after %d attempt(s)", attempt)
			break
		} else {
			t.Logf("Attempt %d/%d: Service accounts not yet available, waiting...", attempt, maxAttempts)
		}

		if attempt < maxAttempts {
			time.Sleep(1 * time.Second)
		} else {
			t.Fatalf("Service account verification failed after %d attempts. The service accounts may not have been created correctly.", maxAttempts)
		}
	}

	// Grant "Our Service Account" permission to impersonate "Customer Service Account"
	t.Logf("Granting impersonation permission to 'Our Service Account'")
	err = grantServiceAccountTokenCreator(ctx, iamService, state.projectID, state.customerServiceAccountEmail, state.ourServiceAccountEmail)
	if err != nil {
		t.Fatalf("Failed to grant serviceAccountTokenCreator role: %v", err)
	}

	// Verify IAM policy binding is effective
	t.Log("Verifying IAM policy binding for serviceAccountTokenCreator role...")
	maxAttempts = 30
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		verified, err := verifyServiceAccountTokenCreatorBinding(ctx, iamService, state.projectID, state.customerServiceAccountEmail, state.ourServiceAccountEmail)
		if err != nil {
			t.Logf("Attempt %d/%d: Error verifying IAM policy: %v", attempt, maxAttempts, err)
		} else if verified {
			t.Logf("✓ IAM policy binding verified after %d attempt(s)", attempt)
			break
		} else {
			t.Logf("Attempt %d/%d: IAM policy binding not yet effective, waiting...", attempt, maxAttempts)
		}

		if attempt < maxAttempts {
			time.Sleep(1 * time.Second)
		} else {
			t.Fatalf("IAM policy binding verification failed after %d attempts. The role may not have been granted correctly.", maxAttempts)
		}
	}

	// Grant "Customer Service Account" permission to list Compute Engine instances
	t.Logf("Granting roles/compute.viewer to 'Customer Service Account' at project level")
	err = grantProjectIAMRole(ctx, crmService, state.projectID, state.customerServiceAccountEmail, "roles/compute.viewer")
	if err != nil {
		t.Fatalf("Failed to grant roles/compute.viewer role: %v", err)
	}

	// Create service account keys for authentication
	t.Log("Creating service account keys...")

	// Create key for "Our Service Account"
	ourKey, err := createServiceAccountKey(ctx, iamService, state.projectID, state.ourServiceAccountEmail)
	if err != nil {
		t.Fatalf("Failed to create key for 'Our Service Account': %v", err)
	}
	state.ourServiceAccountKey = []byte(ourKey.PrivateKeyData)
	state.ourServiceAccountKeyID = extractKeyID(ourKey.Name)
	t.Logf("Created key for 'Our Service Account': %s", state.ourServiceAccountKeyID)

	// Create key for "Customer Service Account"
	customerKey, err := createServiceAccountKey(ctx, iamService, state.projectID, state.customerServiceAccountEmail)
	if err != nil {
		t.Fatalf("Failed to create key for 'Customer Service Account': %v", err)
	}
	state.customerServiceAccountKey = []byte(customerKey.PrivateKeyData)
	state.customerServiceAccountKeyID = extractKeyID(customerKey.Name)
	t.Logf("Created key for 'Customer Service Account': %s", state.customerServiceAccountKeyID)

	// Verify permission is actually effective by attempting GenerateAccessToken
	// This is different from just checking the IAM policy exists - it verifies enforcement
	t.Log("Verifying permission is actually effective by attempting GenerateAccessToken...")
	keyData, err := base64.StdEncoding.DecodeString(string(state.ourServiceAccountKey))
	if err != nil {
		t.Fatalf("Failed to decode service account key for verification: %v", err)
	}

	maxAttempts = 60 // Allow more time for enforcement
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Create credentials from "Our Service Account" key
		testCreds, err := google.CredentialsFromJSON(ctx, keyData, iam.CloudPlatformScope)
		if err != nil {
			t.Fatalf("Failed to create credentials for verification: %v", err)
		}

		// Create IAM Credentials client
		testClient, err := credentials.NewIamCredentialsClient(ctx, option.WithTokenSource(testCreds.TokenSource))
		if err != nil {
			t.Fatalf("Failed to create IAM Credentials client for verification: %v", err)
		}

		// Attempt to generate a token to verify the permission is actually effective
		testReq := &credentialspb.GenerateAccessTokenRequest{
			Name:  fmt.Sprintf("projects/-/serviceAccounts/%s", state.customerServiceAccountEmail),
			Scope: []string{"https://www.googleapis.com/auth/cloud-platform"},
		}
		_, err = testClient.GenerateAccessToken(ctx, testReq)
		testClient.Close()

		if err == nil {
			t.Logf("✓ Permission is actually effective after %d attempt(s)", attempt)
			break
		}

		if attempt < maxAttempts {
			t.Logf("Attempt %d/%d: Permission not yet effective, error: %v, waiting...", attempt, maxAttempts, err)
			time.Sleep(2 * time.Second)
		} else {
			t.Fatalf("Permission verification failed after %d attempts. The permission may not be enforced yet. Last error: %v", maxAttempts, err)
		}
	}
}

func testOurServiceAccountDirectAuth(t *testing.T, ctx context.Context, state *testState) {
	t.Log("Test 1: Authenticating as 'Our Service Account' directly")

	// Decode the service account key
	keyData, err := base64.StdEncoding.DecodeString(string(state.ourServiceAccountKey))
	if err != nil {
		t.Fatalf("Failed to decode service account key: %v", err)
	}

	// Create credentials from the key
	creds, err := google.CredentialsFromJSON(ctx, keyData, compute.DefaultAuthScopes()...)
	if err != nil {
		t.Logf("Key data: %s", string(keyData))
		t.Fatalf("Failed to create credentials from key: %v", err)
	}

	// Create Compute Engine client using these credentials
	client, err := compute.NewInstancesRESTClient(ctx, option.WithTokenSource(creds.TokenSource))
	if err != nil {
		t.Fatalf("Failed to create Compute client: %v", err)
	}
	defer client.Close()

	// Attempt to list instances - this should fail with permission denied
	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		zone = "us-central1-a" // Default zone
	}

	req := &computepb.ListInstancesRequest{
		Project: state.projectID,
		Zone:    zone,
	}

	it := client.List(ctx, req)
	_, err = it.Next()

	// We expect a permission error
	if err == nil {
		t.Fatal("Expected permission denied error, but listing succeeded")
	}

	// Check if it's a permission error
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) {
		if apiErr.HTTPCode() == http.StatusForbidden || apiErr.GRPCStatus().Code().String() == "PermissionDenied" {
			t.Logf("✓ Correctly received permission denied error: %v", err)
			return
		}
		t.Fatalf("Expected permission denied error, got: %v", err)
	}

	// Also check for googleapi.Error
	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		if gErr.Code == http.StatusForbidden {
			t.Logf("✓ Correctly received permission denied error: %v", err)
			return
		}
		t.Fatalf("Expected permission denied error, got: %v", err)
	}

	t.Fatalf("Expected permission denied error, got unexpected error: %v", err)
}

func testCustomerServiceAccountDirectAuth(t *testing.T, ctx context.Context, state *testState) {
	t.Log("Test 2: Authenticating as 'Customer Service Account' directly")

	// Decode the service account key
	keyData, err := base64.StdEncoding.DecodeString(string(state.customerServiceAccountKey))
	if err != nil {
		t.Fatalf("Failed to decode service account key: %v", err)
	}

	// Create credentials from the key
	creds, err := google.CredentialsFromJSON(ctx, keyData, compute.DefaultAuthScopes()...)
	if err != nil {
		t.Fatalf("Failed to create credentials from key: %v", err)
	}

	// Create Compute Engine client using these credentials
	client, err := compute.NewInstancesRESTClient(ctx, option.WithTokenSource(creds.TokenSource))
	if err != nil {
		t.Fatalf("Failed to create Compute client: %v", err)
	}
	defer client.Close()

	// Attempt to list instances - this should succeed
	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		zone = "us-central1-a" // Default zone
	}

	req := &computepb.ListInstancesRequest{
		Project: state.projectID,
		Zone:    zone,
	}

	it := client.List(ctx, req)
	_, err = it.Next()
	if err != nil {
		t.Fatalf("Expected to successfully list instances, but got error: %v", err)
	}

	t.Log("✓ Successfully listed instances as 'Customer Service Account'")
}

func testImpersonation(t *testing.T, ctx context.Context, state *testState) {
	t.Log("Test 3: Authenticating as 'Our Service Account' and impersonating 'Customer Service Account'")

	// Decode the "Our Service Account" key
	keyData, err := base64.StdEncoding.DecodeString(string(state.ourServiceAccountKey))
	if err != nil {
		t.Fatalf("Failed to decode service account key: %v", err)
	}

	// Create credentials from "Our Service Account" key
	creds, err := google.CredentialsFromJSON(ctx, keyData, iam.CloudPlatformScope)
	if err != nil {
		t.Fatalf("Failed to create credentials from key: %v", err)
	}

	// Create IAM Credentials client using "Our Service Account" credentials
	iamCredsClient, err := credentials.NewIamCredentialsClient(ctx, option.WithTokenSource(creds.TokenSource))
	if err != nil {
		t.Fatalf("Failed to create IAM Credentials client: %v", err)
	}
	defer iamCredsClient.Close()

	// Generate access token for "Customer Service Account" for impersonating it
	generateTokenReq := &credentialspb.GenerateAccessTokenRequest{
		Name:  fmt.Sprintf("projects/-/serviceAccounts/%s", state.customerServiceAccountEmail),
		Scope: compute.DefaultAuthScopes(),
	}

	tokenResp, err := iamCredsClient.GenerateAccessToken(ctx, generateTokenReq)
	if err != nil {
		t.Fatalf("Failed to generate access token for impersonated service account: %v", err)
	}

	// Create Compute Engine client using the impersonated token
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: tokenResp.GetAccessToken(),
	})

	client, err := compute.NewInstancesRESTClient(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		t.Fatalf("Failed to create Compute client: %v", err)
	}
	defer client.Close()

	// Attempt to list instances - this should succeed
	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		zone = "us-central1-a" // Default zone
	}

	req := &computepb.ListInstancesRequest{
		Project: state.projectID,
		Zone:    zone,
	}

	it := client.List(ctx, req)
	_, err = it.Next()
	if err != nil {
		t.Fatalf("Expected to successfully list instances via impersonation, but got error: %v", err)
	}

	t.Log("✓ Successfully listed instances via impersonation")
}

func teardownTest(t *testing.T, ctx context.Context, iamService *iam.Service, crmService *cloudresourcemanager.Service, state *testState) {
	// Delete service account keys first (required before deleting service accounts)
	if state.ourServiceAccountKeyID != "" {
		t.Logf("Deleting key for 'Our Service Account': %s", state.ourServiceAccountKeyID)
		keyResource := fmt.Sprintf("projects/%s/serviceAccounts/%s/keys/%s",
			state.projectID, state.ourServiceAccountEmail, state.ourServiceAccountKeyID)
		_, err := iamService.Projects.ServiceAccounts.Keys.Delete(keyResource).Do()
		if err != nil {
			var gErr *googleapi.Error
			if errors.As(err, &gErr) && gErr.Code == http.StatusNotFound {
				t.Log("Key already deleted or not found")
			} else {
				t.Logf("Failed to delete key (non-fatal): %v", err)
			}
		}
	}

	if state.customerServiceAccountKeyID != "" {
		t.Logf("Deleting key for 'Customer Service Account': %s", state.customerServiceAccountKeyID)
		keyResource := fmt.Sprintf("projects/%s/serviceAccounts/%s/keys/%s",
			state.projectID, state.customerServiceAccountEmail, state.customerServiceAccountKeyID)
		_, err := iamService.Projects.ServiceAccounts.Keys.Delete(keyResource).Do()
		if err != nil {
			var gErr *googleapi.Error
			if errors.As(err, &gErr) && gErr.Code == http.StatusNotFound {
				t.Log("Key already deleted or not found")
			} else {
				t.Logf("Failed to delete key (non-fatal): %v", err)
			}
		}
	}

	// Delete service accounts
	if state.customerServiceAccountEmail != "" {
		t.Logf("Deleting 'Customer Service Account': %s", state.customerServiceAccountEmail)
		saResource := fmt.Sprintf("projects/%s/serviceAccounts/%s", state.projectID, state.customerServiceAccountEmail)
		_, err := iamService.Projects.ServiceAccounts.Delete(saResource).Do()
		if err != nil {
			var gErr *googleapi.Error
			if errors.As(err, &gErr) && (gErr.Code == http.StatusNotFound || gErr.Code == http.StatusForbidden) {
				t.Log("Service account already deleted or not found")
			} else {
				t.Logf("Failed to delete service account (non-fatal): %v", err)
			}
		}
	}

	if state.ourServiceAccountEmail != "" {
		t.Logf("Deleting 'Our Service Account': %s", state.ourServiceAccountEmail)
		saResource := fmt.Sprintf("projects/%s/serviceAccounts/%s", state.projectID, state.ourServiceAccountEmail)
		_, err := iamService.Projects.ServiceAccounts.Delete(saResource).Do()
		if err != nil {
			var gErr *googleapi.Error
			if errors.As(err, &gErr) && (gErr.Code == http.StatusNotFound || gErr.Code == http.StatusForbidden) {
				t.Log("Service account already deleted or not found")
			} else {
				t.Logf("Failed to delete service account (non-fatal): %v", err)
			}
		}
	}
}

// Helper functions

func createServiceAccount(ctx context.Context, iamService *iam.Service, projectID, accountID, displayName string) (*iam.ServiceAccount, error) {
	projectResource := fmt.Sprintf("projects/%s", projectID)

	req := &iam.CreateServiceAccountRequest{
		AccountId: accountID,
		ServiceAccount: &iam.ServiceAccount{
			DisplayName: displayName,
			Description: fmt.Sprintf("Test service account created for integration testing: %s", accountID),
		},
	}

	sa, err := iamService.Projects.ServiceAccounts.Create(projectResource, req).Do()
	if err != nil {
		var gErr *googleapi.Error
		if errors.As(err, &gErr) && gErr.Code == http.StatusConflict {
			// Service account already exists, try to get it
			saEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", accountID, projectID)
			saResource := fmt.Sprintf("projects/%s/serviceAccounts/%s", projectID, saEmail)
			return iamService.Projects.ServiceAccounts.Get(saResource).Do()
		}
		return nil, fmt.Errorf("failed to create service account: %w", err)
	}

	return sa, nil
}

func grantServiceAccountTokenCreator(ctx context.Context, iamService *iam.Service, projectID, targetSAEmail, impersonatorSAEmail string) error {
	saResource := fmt.Sprintf("projects/%s/serviceAccounts/%s", projectID, targetSAEmail)

	// Get current IAM policy
	policy, err := iamService.Projects.ServiceAccounts.GetIamPolicy(saResource).Do()
	if err != nil {
		return fmt.Errorf("failed to get IAM policy: %w", err)
	}

	if policy == nil {
		policy = &iam.Policy{}
	}
	if policy.Bindings == nil {
		policy.Bindings = make([]*iam.Binding, 0)
	}

	// Find or create the serviceAccountTokenCreator binding
	role := "roles/iam.serviceAccountTokenCreator"
	member := fmt.Sprintf("serviceAccount:%s", impersonatorSAEmail)

	roleFound := false
	for i, binding := range policy.Bindings {
		if binding.Role == role {
			// Check if member already exists
			memberFound := false
			for _, m := range binding.Members {
				if m == member {
					memberFound = true
					break
				}
			}
			if !memberFound {
				policy.Bindings[i].Members = append(policy.Bindings[i].Members, member)
			}
			roleFound = true
			break
		}
	}

	if !roleFound {
		policy.Bindings = append(policy.Bindings, &iam.Binding{
			Role:    role,
			Members: []string{member},
		})
	}

	// Set the updated policy
	_, err = iamService.Projects.ServiceAccounts.SetIamPolicy(saResource, &iam.SetIamPolicyRequest{
		Policy: policy,
	}).Do()

	return err
}

// verifyServiceAccountExists verifies that a service account exists.
// Returns (true, nil) if the service account exists, (false, nil) if not found, or (false, error) on error.
func verifyServiceAccountExists(ctx context.Context, iamService *iam.Service, projectID, saEmail string) (bool, error) {
	saResource := fmt.Sprintf("projects/%s/serviceAccounts/%s", projectID, saEmail)

	_, err := iamService.Projects.ServiceAccounts.Get(saResource).Do()
	if err != nil {
		var gErr *googleapi.Error
		if errors.As(err, &gErr) && gErr.Code == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to get service account: %w", err)
	}

	return true, nil
}

// verifyServiceAccountTokenCreatorBinding verifies that the impersonator service account
// has the serviceAccountTokenCreator role on the target service account.
// Returns (true, nil) if verified, (false, nil) if not yet effective, or (false, error) on error.
func verifyServiceAccountTokenCreatorBinding(ctx context.Context, iamService *iam.Service, projectID, targetSAEmail, impersonatorSAEmail string) (bool, error) {
	saResource := fmt.Sprintf("projects/%s/serviceAccounts/%s", projectID, targetSAEmail)

	// Get current IAM policy
	policy, err := iamService.Projects.ServiceAccounts.GetIamPolicy(saResource).Do()
	if err != nil {
		return false, fmt.Errorf("failed to get IAM policy: %w", err)
	}

	if policy == nil || policy.Bindings == nil {
		return false, nil
	}

	role := "roles/iam.serviceAccountTokenCreator"
	member := fmt.Sprintf("serviceAccount:%s", impersonatorSAEmail)

	// Check if the binding exists
	for _, binding := range policy.Bindings {
		if binding.Role == role {
			// Check if the impersonator service account is in the members list
			for _, m := range binding.Members {
				if m == member {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func createServiceAccountKey(ctx context.Context, iamService *iam.Service, projectID, saEmail string) (*iam.ServiceAccountKey, error) {
	saResource := fmt.Sprintf("projects/%s/serviceAccounts/%s", projectID, saEmail)

	req := &iam.CreateServiceAccountKeyRequest{}

	key, err := iamService.Projects.ServiceAccounts.Keys.Create(saResource, req).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create service account key: %w", err)
	}

	return key, nil
}

func extractKeyID(keyName string) string {
	// Key name format: projects/{project}/serviceAccounts/{email}/keys/{keyId}
	parts := strings.Split(keyName, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func grantProjectIAMRole(ctx context.Context, crmService *cloudresourcemanager.Service, projectID, saEmail, role string) error {
	member := fmt.Sprintf("serviceAccount:%s", saEmail)

	// Get current IAM policy
	policy, err := crmService.Projects.GetIamPolicy(projectID, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if err != nil {
		return fmt.Errorf("failed to get IAM policy: %w", err)
	}

	if policy == nil {
		policy = &cloudresourcemanager.Policy{}
	}
	if policy.Bindings == nil {
		policy.Bindings = make([]*cloudresourcemanager.Binding, 0)
	}

	// Find or create the binding for the role
	roleFound := false
	for i, binding := range policy.Bindings {
		if binding.Role == role {
			// Check if member already exists
			memberFound := false
			for _, m := range binding.Members {
				if m == member {
					memberFound = true
					break
				}
			}
			if !memberFound {
				policy.Bindings[i].Members = append(policy.Bindings[i].Members, member)
			}
			roleFound = true
			break
		}
	}

	if !roleFound {
		policy.Bindings = append(policy.Bindings, &cloudresourcemanager.Binding{
			Role:    role,
			Members: []string{member},
		})
	}

	// Set the updated policy
	_, err = crmService.Projects.SetIamPolicy(projectID, &cloudresourcemanager.SetIamPolicyRequest{
		Policy: policy,
	}).Do()

	return err
}
