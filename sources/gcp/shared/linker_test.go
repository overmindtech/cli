package shared

import (
	"context"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

func Test_isIPAddress(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{
			name: "valid IPv4",
			s:    "192.168.1.1",
			want: true,
		},
		{
			name: "valid IPv6",
			s:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			want: true,
		},
		{
			name: "invalid IP - random string",
			s:    "not.an.ip",
			want: false,
		},
		{
			name: "empty string",
			s:    "",
			want: false,
		},
		{
			name: "hostname",
			s:    "example.com",
			want: false,
		},
		{
			name: "IPv4 with port",
			s:    "127.0.0.1:80",
			want: false,
		},
		{
			name: "IPv6 with brackets",
			s:    "[2001:db8::1]",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIPAddress(tt.s)
			if got != tt.want {
				t.Errorf("isIPAddress(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func Test_isDNSName(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{
			name: "valid DNS name",
			s:    "example.com",
			want: true,
		},
		{
			name: "valid DNS name with subdomain",
			s:    "sub.example.com",
			want: true,
		},
		{
			name: "valid DNS name with hyphen",
			s:    "my-site.example.com",
			want: true,
		},
		{
			name: "valid DNS name with numbers",
			s:    "123.example.com",
			want: true,
		},
		{
			name: "single label (no dot)",
			s:    "localhost",
			want: false,
		},
		{
			name: "contains underscore (invalid)",
			s:    "foo_bar.example.com",
			want: false,
		},
		{
			name: "contains space (invalid)",
			s:    "foo bar.example.com",
			want: false,
		},
		{
			name: "empty string",
			s:    "",
			want: false,
		},
		{
			name: "valid IPv4 address",
			s:    "192.168.1.1",
			want: false,
		},
		{
			name: "valid IPv6 address",
			s:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			want: false,
		},
		{
			name: "IPv4 with port",
			s:    "127.0.0.1:80",
			want: false,
		},
		{
			name: "DNS name with trailing dot - will be normalized",
			s:    "example.com.",
			want: true,
		},
		{
			name: "DNS name with multiple dots",
			s:    "a.b.c.d.e.f.g.com",
			want: true,
		},
		{
			name: "DNS name with only dots",
			s:    "...",
			want: false,
		},
		{
			name: "bracketed IPv6",
			s:    "[2001:db8::1]",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDNSName(tt.s)
			if got != tt.want {
				t.Errorf("isDNSName(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestLinker_Link(t *testing.T) {
	type args struct {
		fromSDPItemType       shared.ItemType
		toItemGCPResourceName string
		toSDPItemType         shared.ItemType
	}
	tests := []struct {
		name       string
		args       args
		expectLink bool
	}{
		{
			name: "Link to a manual adapter",
			args: args{
				fromSDPItemType:       ComputeInstance,
				toItemGCPResourceName: "https://www.googleapis.com/compute/v1/projects/project-test/zones/us-central1-c/disks/integration-test-instance",
				toSDPItemType:         ComputeDisk,
			},
			expectLink: true,
		},
		{
			name: "Should not link to itself",
			args: args{
				fromSDPItemType:       ComputeInstance,
				toItemGCPResourceName: "https://www.googleapis.com/compute/v1/projects/project-test/zones/us-central1-c/instances/integration-test-instance",
				toSDPItemType:         ComputeInstance,
			},
		},
		{
			name: "Link to a dynamic adapter",
			args: args{
				fromSDPItemType:       ComputeInstance,
				toItemGCPResourceName: "projects/my-project/global/networks/my-network",
				toSDPItemType:         ComputeNetwork, // We don't have a manual adapter for this, so it should use the dynamic adapter
			},
			expectLink: true,
		},
		{
			name: "Missing blast propagation for the From type",
			args: args{
				fromSDPItemType:       shared.NewItemType("foo", "bar", "baz"),
				toItemGCPResourceName: "https://www.googleapis.com/compute/v1/projects/project-test/zones/us-central1-c/disks/test-disk",
				toSDPItemType:         ComputeDisk,
			},
			expectLink: false,
		},
	}
	projectID := "test-project-id"
	l := NewLinker()
	for _, tt := range tests {
		fromSDPItem := &sdp.Item{}
		t.Run(tt.name, func(t *testing.T) {
			l.Link(context.TODO(), projectID, fromSDPItem, tt.args.fromSDPItemType, tt.args.toItemGCPResourceName, tt.args.toSDPItemType)

			if tt.expectLink && len(fromSDPItem.GetLinkedItemQueries()) == 0 {
				t.Fatalf("Linker.Link() did not return any linked items, expected at least one")
			}

			if !tt.expectLink && len(fromSDPItem.GetLinkedItemQueries()) > 0 {
				t.Fatalf("Linker.Link() returned linked items, expected none")
			}

			if !tt.expectLink {
				return
			}

			linkedItemQuery := fromSDPItem.GetLinkedItemQueries()[0]
			if linkedItemQuery.GetQuery() != nil && linkedItemQuery.GetQuery().GetType() != tt.args.toSDPItemType.String() {
				t.Errorf("Linker.Link() returned linked item with type %s, expected %s", linkedItemQuery.GetQuery().GetType(), tt.args.toSDPItemType.String())
			}
		})
	}
}

func TestLinker_AutoLink(t *testing.T) {
	type args struct {
		fromSDPItemType       shared.ItemType
		toItemGCPResourceName string
		toSDPItemType         shared.ItemType
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Auto link from ComputeInstance to ComputeDisk via manual adapters",
			args: args{
				fromSDPItemType:       ComputeInstance,
				toItemGCPResourceName: "projects/project-test/zones/us-central1-c/disks/test-disk",
				toSDPItemType:         ComputeDisk,
			},
		},
		{
			name: "Auto link from ComputeInstance to ComputeNetwork via dynamic adapters",
			args: args{
				fromSDPItemType:       ComputeInstance,
				toItemGCPResourceName: "https://compute.googleapis.com/compute/v1/projects/my-project/global/networks/my-network",
				toSDPItemType:         ComputeNetwork,
			},
		},
	}
	projectID := "project-test"
	l := NewLinker()
	for _, tt := range tests {
		fromSDPItem := &sdp.Item{}
		t.Run(tt.name, func(t *testing.T) {
			l.AutoLink(context.TODO(), projectID, fromSDPItem, tt.args.fromSDPItemType, tt.args.toItemGCPResourceName)

			if len(fromSDPItem.GetLinkedItemQueries()) == 0 {
				t.Fatalf("Linker.AutoLink() did not return any linked items, expected at least one")
			}

			linkedItemQuery := fromSDPItem.GetLinkedItemQueries()[0]
			if linkedItemQuery.GetQuery() != nil && linkedItemQuery.GetQuery().GetType() != tt.args.toSDPItemType.String() {
				t.Errorf("Linker.Link() returned linked item with type %s, expected %s", linkedItemQuery.GetQuery().GetType(), tt.args.toSDPItemType.String())
			}
		})
	}
}

func Test_determineScope(t *testing.T) {
	type args struct {
		ctx                   context.Context
		projectID             string
		scope                 Scope
		toItemGCPResourceName string
		parts                 []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Project scope",
			args: args{
				ctx:                   context.TODO(),
				projectID:             "my-project",
				scope:                 ScopeProject,
				toItemGCPResourceName: "projects/my-project/global/networks/my-network",
				parts:                 []string{"projects", "my-project", "global", "networks", "my-network"},
			},
			want: "my-project",
		},
		{
			name: "Regional scope",
			args: args{
				ctx:                   context.TODO(),
				projectID:             "my-project",
				scope:                 ScopeRegional,
				toItemGCPResourceName: "projects/my-project/regions/us-central1/networks/my-network",
				parts:                 []string{"projects", "my-project", "regions", "us-central1", "networks", "my-network"},
			},
			want: "my-project.us-central1",
		},
		{
			name: "Zonal scope",
			args: args{
				ctx:                   context.TODO(),
				projectID:             "my-project",
				scope:                 ScopeZonal,
				toItemGCPResourceName: "projects/my-project/zones/us-central1-c/instances/my-instance",
				parts:                 []string{"projects", "my-project", "zones", "us-central1-c", "instances", "my-instance"},
			},
			want: "my-project.us-central1-c",
		},
		{
			name: "Regional scope, invalid parts length",
			args: args{
				ctx:                   context.TODO(),
				projectID:             "my-project",
				scope:                 ScopeRegional,
				toItemGCPResourceName: "projects/my-project",
				parts:                 []string{"projects", "my-project"},
			},
			want: "",
		},
		{
			name: "Zonal scope, invalid parts length",
			args: args{
				ctx:                   context.TODO(),
				projectID:             "my-project",
				scope:                 ScopeZonal,
				toItemGCPResourceName: "projects/my-project",
				parts:                 []string{"projects", "my-project"},
			},
			want: "",
		},
		{
			name: "Unknown scope",
			args: args{
				ctx:                   context.TODO(),
				projectID:             "my-project",
				scope:                 Scope("unknown"),
				toItemGCPResourceName: "projects/my-project/zones/us-central1-c/instances/my-instance",
				parts:                 []string{"projects", "my-project", "zones", "us-central1-c", "instances", "my-instance"},
			},
			want: "",
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := determineScope(tt.args.ctx, tt.args.projectID, tt.args.scope, nil, tt.args.toItemGCPResourceName, tt.args.parts); got != tt.want {
				t.Errorf("determineScope() = %v, want %v", got, tt.want)
			}
		})
	}
}
