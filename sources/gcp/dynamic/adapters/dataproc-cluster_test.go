package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/dataproc/apiv1/dataprocpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestDataprocCluster(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	region := "us-central1"
	zone := region + "-a"
	linker := gcpshared.NewLinker()
	clusterName := "test-cluster"

	cluster := &dataprocpb.Cluster{
		ClusterName: clusterName,
		Config: &dataprocpb.ClusterConfig{
			GceClusterConfig: &dataprocpb.GceClusterConfig{
				NetworkUri:     fmt.Sprintf("projects/%s/global/networks/default", projectID),
				SubnetworkUri:  fmt.Sprintf("projects/%s/regions/%s/subnetworks/default-subnet", projectID, region),
				ServiceAccount: "test-sa@test-project.iam.gserviceaccount.com",
			},
			EncryptionConfig: &dataprocpb.EncryptionConfig{
				GcePdKmsKeyName: "projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/test-key",
			},
			MasterConfig: &dataprocpb.InstanceGroupConfig{
				ImageUri:       fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/images/master-dataproc-image", projectID),
				MachineTypeUri: fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes/n1-standard-4", projectID, zone),
				Accelerators: []*dataprocpb.AcceleratorConfig{
					{
						AcceleratorTypeUri: fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/acceleratorTypes/nvidia-tesla-t4", projectID, zone),
					},
				},
			},
			WorkerConfig: &dataprocpb.InstanceGroupConfig{
				ImageUri:       fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/images/worker-dataproc-image", projectID),
				MachineTypeUri: fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes/n1-standard-8", projectID, zone),
			},
			SecondaryWorkerConfig: &dataprocpb.InstanceGroupConfig{
				ImageUri:       fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/images/secondary-dataproc-image", projectID),
				MachineTypeUri: fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes/n1-standard-2", projectID, zone),
			},
			AutoscalingConfig: &dataprocpb.AutoscalingConfig{
				PolicyUri: fmt.Sprintf("projects/%s/regions/%s/autoscalingPolicies/test-policy", projectID, region),
			},
			TempBucket: "test-temp-bucket",
		},
	}

	clusterName2 := "test-cluster-2"
	cluster2 := &dataprocpb.Cluster{
		ClusterName: clusterName2,
	}

	clusterList := &dataprocpb.ListClustersResponse{
		Clusters: []*dataprocpb.Cluster{cluster, cluster2},
	}

	sdpItemType := gcpshared.DataprocCluster

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://dataproc.googleapis.com/v1/projects/%s/regions/%s/clusters/%s", projectID, region, clusterName): {
			StatusCode: http.StatusOK,
			Body:       cluster,
		},
		fmt.Sprintf("https://dataproc.googleapis.com/v1/projects/%s/regions/%s/clusters/%s", projectID, region, clusterName2): {
			StatusCode: http.StatusOK,
			Body:       cluster2,
		},
		fmt.Sprintf("https://dataproc.googleapis.com/v1/projects/%s/regions/%s/clusters", projectID, region): {
			StatusCode: http.StatusOK,
			Body:       clusterList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), clusterName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != clusterName {
			t.Errorf("Expected unique attribute value '%s', got %s", clusterName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default-subnet",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-sa@test-project.iam.gserviceaccount.com",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "test-ring", "test-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Master config
				{
					ExpectedType:   gcpshared.ComputeImage.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "master-dataproc-image",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Master machine type
				{
					ExpectedType:   gcpshared.ComputeMachineType.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "n1-standard-4",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Master accelerator
				{
					ExpectedType:   gcpshared.ComputeAcceleratorType.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "nvidia-tesla-t4",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Worker config
				{
					ExpectedType:   gcpshared.ComputeImage.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "worker-dataproc-image",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeMachineType.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "n1-standard-8",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Secondary worker config
				{
					ExpectedType:   gcpshared.ComputeImage.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "secondary-dataproc-image",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeMachineType.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "n1-standard-2",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-temp-bucket",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.DataprocAutoscalingPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-policy",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, fmt.Sprintf("%s.%s", projectID, region), true)
		if err != nil {
			t.Fatalf("Failed to list resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://dataproc.googleapis.com/v1/projects/%s/regions/%s/clusters/%s", projectID, region, clusterName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Cluster not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), clusterName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
