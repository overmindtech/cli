package dynamic

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func Test_externalToSDP(t *testing.T) {
	type args struct {
		projectID      string
		scope          string
		uniqueAttrKeys []string
		resp           map[string]interface{}
		sdpAssetType   shared.ItemType
	}
	tests := []struct {
		name    string
		args    args
		want    *sdp.Item
		wantErr bool
	}{
		{
			name: "ReturnsSDPItemWithCorrectAttributes",
			args: args{
				projectID:      "test-project",
				scope:          "test-scope",
				uniqueAttrKeys: []string{"projects", "locations", "instances"},
				resp: map[string]interface{}{
					"name":   "projects/test-project/locations/us-central1/instances/instance-1",
					"labels": map[string]interface{}{"env": "prod"},
					"foo":    "bar",
				},
				sdpAssetType: gcpshared.ComputeInstance,
			},
			want: &sdp.Item{
				Type:            gcpshared.ComputeInstance.String(),
				UniqueAttribute: "uniqueAttr",
				Scope:           "test-scope",
				Tags:            map[string]string{"env": "prod"},
				Attributes: &sdp.ItemAttributes{
					AttrStruct: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"name":       structpb.NewStringValue("projects/test-project/locations/us-central1/instances/instance-1"),
							"foo":        structpb.NewStringValue("bar"),
							"uniqueAttr": structpb.NewStringValue("test-project|us-central1|instance-1"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ReturnsSDPItemWithCorrectAttributesWhenNameDoesNotHaveUniqueAttrKeys",
			args: args{
				projectID:      "test-project",
				scope:          "test-scope",
				uniqueAttrKeys: []string{"projects", "locations", "instances"},
				resp: map[string]interface{}{
					// There is name, but it does not include uniqueAttrKeys, expected to use the name as is.
					"name":   "instance-1",
					"labels": map[string]interface{}{"env": "prod"},
					"foo":    "bar",
				},
				sdpAssetType: gcpshared.ComputeInstance,
			},
			want: &sdp.Item{
				Type:            gcpshared.ComputeInstance.String(),
				UniqueAttribute: "uniqueAttr",
				Scope:           "test-scope",
				Tags:            map[string]string{"env": "prod"},
				Attributes: &sdp.ItemAttributes{
					AttrStruct: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"name":       structpb.NewStringValue("instance-1"),
							"foo":        structpb.NewStringValue("bar"),
							"uniqueAttr": structpb.NewStringValue("instance-1"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ReturnsErrorWhenNameMissing",
			args: args{
				projectID:      "test-project",
				scope:          "test-scope",
				uniqueAttrKeys: []string{"projects", "locations", "instances"},
				resp: map[string]interface{}{
					"labels": map[string]interface{}{"env": "prod"},
					"foo":    "bar",
				},
				sdpAssetType: gcpshared.ComputeInstance,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ReturnsSDPItemWithEmptyLabels",
			args: args{
				projectID:      "test-project",
				scope:          "test-scope",
				uniqueAttrKeys: []string{"projects", "locations", "instances"},
				resp: map[string]interface{}{
					"name": "projects/test-project/locations/us-central1/instances/instance-2",
					"foo":  "baz",
				},
				sdpAssetType: gcpshared.ComputeInstance,
			},
			want: &sdp.Item{
				Type:            gcpshared.ComputeInstance.String(),
				UniqueAttribute: "uniqueAttr",
				Attributes: &sdp.ItemAttributes{
					AttrStruct: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"name":       structpb.NewStringValue("projects/test-project/locations/us-central1/instances/instance-2"),
							"foo":        structpb.NewStringValue("baz"),
							"uniqueAttr": structpb.NewStringValue("test-project|us-central1|instance-2"),
						},
					},
				},
				Scope: "test-scope",
				Tags:  map[string]string{},
			},
			wantErr: false,
		},
	}
	linker := gcpshared.NewLinker()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := externalToSDP(context.Background(), tt.args.projectID, tt.args.scope, tt.args.uniqueAttrKeys, tt.args.resp, tt.args.sdpAssetType, linker)
			if (err != nil) != tt.wantErr {
				t.Errorf("externalToSDP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			//got.Attributes = createAttr(t, tt.args.resp)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("externalToSDP() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getDescription_ReturnsSelectorWithNameWhenNoUniqueAttrKeys(t *testing.T) {
	got := getDescription(gcpshared.ComputeInstance, "test-scope", []string{})
	want := fmt.Sprintf("Get a %s by its {name} within its scope: test-scope", gcpshared.ComputeInstance)
	if got != want {
		t.Errorf("getDescription() got = %v, want %v", got, want)
	}
}

func Test_getDescription_ReturnsSelectorWithUniqueAttrKeys(t *testing.T) {
	got := getDescription(gcpshared.BigQueryTable, "project-1", []string{"datasets", "tables"})
	want := fmt.Sprintf("Get a %s by its {datasets|tables} within its scope: project-1", gcpshared.BigQueryTable)
	if got != want {
		t.Errorf("getDescription() got = %v, want %v", got, want)
	}
}

func Test_getDescription_ReturnsSelectorWithSingleUniqueAttrKey(t *testing.T) {
	got := getDescription(gcpshared.StorageBucket, "scope-2", []string{"buckets"})
	want := fmt.Sprintf("Get a %s by its {name} within its scope: scope-2", gcpshared.StorageBucket)
	if got != want {
		t.Errorf("getDescription() got = %v, want %v", got, want)
	}
}

func Test_listDescription_ReturnsCorrectDescription(t *testing.T) {
	got := listDescription(gcpshared.ComputeInstance, "test-scope")
	want := "List all gcp-compute-instance within its scope: test-scope"
	if got != want {
		t.Errorf("listDescription() got = %v, want %v", got, want)
	}
}

func Test_listDescription_HandlesEmptyScope(t *testing.T) {
	got := listDescription(gcpshared.BigQueryTable, "test-scope")
	want := "List all gcp-big-query-table within its scope: test-scope"
	if got != want {
		t.Errorf("listDescription() got = %v, want %v", got, want)
	}
}

func Test_searchDescription_ReturnsSelectorWithMultipleKeys(t *testing.T) {
	got := searchDescription(gcpshared.ServiceDirectoryEndpoint, "test-scope", []string{"locations", "namespaces", "services", "endpoints"})
	want := "Search for gcp-service-directory-endpoint by its {locations|namespaces|services} within its scope: test-scope"
	if got != want {
		t.Errorf("searchDescription() got = %v, want %v", got, want)
	}
}

func Test_searchDescription_ReturnsSelectorWithTwoKeys(t *testing.T) {
	got := searchDescription(gcpshared.BigQueryTable, "project-1", []string{"datasets", "tables"})
	want := "Search for gcp-big-query-table by its {datasets} within its scope: project-1"
	if got != want {
		t.Errorf("searchDescription() got = %v, want %v", got, want)
	}
}

func Test_searchDescription_PanicsWithOneKey(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("searchDescription() did not panic with one unique attribute key; expected panic")
		}
	}()
	_ = searchDescription(gcpshared.StorageBucket, "scope-2", []string{"buckets"})
}
