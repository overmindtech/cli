package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/googleapis/gax-go/v2/apierror"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestComputeReservationIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		t.Skip("GCP_ZONE environment variable not set")
	}

	reservationName := "integration-test-reservation"
	machineType := "e2-medium" // Use a common machine type for testing

	ctx := context.Background()

	// Create a new Compute Reservations client
	client, err := compute.NewReservationsRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewReservationsRESTClient: %v", err)
	}
	defer client.Close()

	t.Run("Setup", func(t *testing.T) {
		err := createComputeReservation(ctx, client, projectID, zone, reservationName, machineType)
		if err != nil {
			t.Fatalf("Failed to create compute reservation: %v", err)
		}
	})

	t.Run("ListReservations", func(t *testing.T) {
		log.Printf("Listing reservations in project %s, zone %s", projectID, zone)

		reservationsWrapper := manual.NewComputeReservation(gcpshared.NewComputeReservationClient(client), projectID, zone)
		scope := reservationsWrapper.Scopes()[0]

		reservationsAdapter := sources.WrapperToAdapter(reservationsWrapper)
		sdpItems, err := reservationsAdapter.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list compute reservations: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one compute reservation, got %d", len(sdpItems))
		}

		var found bool
		for _, item := range sdpItems {
			uniqueAttrKey := item.GetUniqueAttribute()
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == reservationName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find reservation %s in the list of compute reservations", reservationName)
		}

		log.Printf("Found %d reservations in project %s, zone %s", len(sdpItems), projectID, zone)
	})

	t.Run("GetReservation", func(t *testing.T) {
		log.Printf("Retrieving reservation %s in project %s, zone %s", reservationName, projectID, zone)

		reservationsWrapper := manual.NewComputeReservation(gcpshared.NewComputeReservationClient(client), projectID, zone)
		scope := reservationsWrapper.Scopes()[0]

		reservationsAdapter := sources.WrapperToAdapter(reservationsWrapper)
		sdpItem, qErr := reservationsAdapter.Get(ctx, scope, reservationName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem == nil {
			t.Fatalf("Expected sdpItem to be non-nil")
		}

		uniqueAttrKey := sdpItem.GetUniqueAttribute()
		uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
		if err != nil {
			t.Fatalf("Failed to get unique attribute: %v", err)
		}

		if uniqueAttrValue != reservationName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", reservationName, uniqueAttrValue)
		}

		log.Printf("Successfully retrieved reservation %s in project %s, zone %s", reservationName, projectID, zone)
	})

	t.Run("Teardown", func(t *testing.T) {
		err := deleteReservation(ctx, client, projectID, zone, reservationName)
		if err != nil {
			t.Fatalf("Failed to delete compute reservation: %v", err)
		}
	})
}

// createComputeReservation creates a GCP Compute Reservation with the given parameters.
func createComputeReservation(ctx context.Context, client *compute.ReservationsClient, projectID, zone, reservationName, machineType string) error {
	reservation := &computepb.Reservation{
		Name: ptr.To(reservationName),
		SpecificReservation: &computepb.AllocationSpecificSKUReservation{
			InstanceProperties: &computepb.AllocationSpecificSKUAllocationReservedInstanceProperties{
				MachineType: ptr.To(machineType),
			},
			Count: ptr.To(int64(1)),
		},
	}

	req := &computepb.InsertReservationRequest{
		Project:             projectID,
		Zone:                zone,
		ReservationResource: reservation,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusConflict {
			log.Printf("Resource already exists in project, skipping creation: %v", err)
			return nil
		}

		return fmt.Errorf("failed to create resource: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for reservation creation operation: %w", err)
	}

	log.Printf("Reservation %s created successfully in project %s, zone %s", reservationName, projectID, zone)
	return nil
}

func deleteReservation(ctx context.Context, client *compute.ReservationsClient, projectID, zone, reservationName string) error {
	req := &computepb.DeleteReservationRequest{
		Project:     projectID,
		Zone:        zone,
		Reservation: reservationName,
	}

	op, err := client.Delete(ctx, req)
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusNotFound {
			log.Printf("Failed to find resource to delete: %v", err)
			return nil
		}

		return fmt.Errorf("failed to delete resource: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for reservation deletion operation: %w", err)
	}

	log.Printf("Compute reservation %s deleted successfully in project %s, zone %s", reservationName, projectID, zone)
	return nil
}
