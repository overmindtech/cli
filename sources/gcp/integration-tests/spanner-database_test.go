package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"github.com/googleapis/gax-go/v2/apierror"
	"google.golang.org/grpc/codes"

	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestSpannerDatabase(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	instanceName := "integration-test-instance"
	databaseName := "integration-test-database"

	ctx := context.Background()

	// Create a new Admin Database instanceClient
	instanceClient, err := instance.NewInstanceAdminClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create Spanner client: %v", err)
	}

	defer instanceClient.Close()

	databaseClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create Spanner database client: %v", err)
	}
	defer databaseClient.Close()

	t.Run("Setup", func(t *testing.T) {
		err := setupSpannerInstance(ctx, instanceClient, projectID, instanceName)
		if err != nil {
			t.Fatalf("Failed to setup Spanner Instance: %v", err)
		}

		err = setupSpannerDatabase(ctx, databaseClient, projectID, instanceName, databaseName)
		if err != nil {
			t.Fatalf("Failed to setup Spanner Database: %v", err)
		}
	})
	t.Run("Run", func(t *testing.T) {
		meta := gcpshared.SDPAssetTypeToAdapterMeta[gcpshared.SpannerDatabase]
		linker := gcpshared.NewLinker()

		gcpHTTPCliWithOtel, err := gcpshared.GCPHTTPClientWithOtel()
		if err != nil {
			t.Fatalf("Failed to create gcp http client with otel")
		}
		adapter, err := dynamic.MakeAdapter(gcpshared.SpannerDatabase, meta, linker, gcpHTTPCliWithOtel, projectID)
		if err != nil {
			t.Fatalf("Failed to make adapter for spanner database")
		}
		query := shared.CompositeLookupKey(instanceName, databaseName)
		sdpItem, err := adapter.Get(ctx, projectID, query, true)
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}

		uniqueAttrKey := sdpItem.GetUniqueAttribute()

		uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
		if err != nil {
			t.Fatalf("Failed to get unique attribute: %v", err)
		}

		if uniqueAttrValue != query {
			t.Fatalf("Expected unique attribute value to be %s, got %s", query, uniqueAttrValue)
		}

		sdpItems, err := adapter.(dynamic.SearchableAdapter).Search(ctx, projectID, instanceName, true)
		if err != nil {
			t.Fatalf("Failed to use spanner database adapter to search: %v", err)
		}
		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one database, got %d", len(sdpItems))
		}
	})
	t.Run("Teardown", func(t *testing.T) {
		err := deleteSpannerDatabase(ctx, databaseClient, projectID, instanceName, databaseName)
		if err != nil {
			t.Fatalf("Failed to teardown Spanner Database: %v", err)
		}

		err = deleteSpannerInstance(ctx, instanceClient, projectID, instanceName)
		if err != nil {
			t.Fatalf("Failed to teardown Spanner Instance: %v", err)
		}
	})
}

func setupSpannerDatabase(ctx context.Context, client *database.DatabaseAdminClient, projectID, instanceName, databaseName string) error {
	// Create the database
	op, err := client.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          "projects/" + projectID + "/instances/" + instanceName,
		CreateStatement: "CREATE DATABASE `" + databaseName + "`",
	})
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.GRPCStatus().Proto().GetCode() == int32(codes.AlreadyExists) {
			log.Printf("Resource already exists in project, skipping creation: %v", err)
			return nil
		}

		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Wait for the operation to complete
	if _, err := op.Wait(ctx); err != nil {
		return err
	}

	return nil
}

func deleteSpannerDatabase(ctx context.Context, client *database.DatabaseAdminClient, projectID, instanceName, databaseName string) error {
	// Delete the database
	err := client.DropDatabase(ctx, &databasepb.DropDatabaseRequest{
		Database: "projects/" + projectID + "/instances/" + instanceName + "/databases/" + databaseName,
	})
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.GRPCStatus().Proto().GetCode() == int32(codes.NotFound) {
			log.Printf("Failed to find resource to delete: %v", err)
			return nil
		}

		return fmt.Errorf("failed to delete resource: %w", err)
	}

	log.Printf("Spanner database %s deleted successfully", databaseName)
	return nil
}
