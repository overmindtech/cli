package shared

import (
	"github.com/overmindtech/cli/sources/shared"
)

const GCP shared.Source = "gcp"

// APIs
const (
	Compute   shared.API = "compute"
	Container shared.API = "container"
)

// Resources
const (
	Instance             shared.Resource = "instance"
	Cluster              shared.Resource = "cluster"
	Disk                 shared.Resource = "disk"
	Network              shared.Resource = "network"
	Subnetwork           shared.Resource = "subnetwork"
	Address              shared.Resource = "address"
	ForwardingRule       shared.Resource = "forwarding-rule"
	BackendService       shared.Resource = "backend-service"
	Autoscaler           shared.Resource = "autoscaler"
	InstanceGroupManager shared.Resource = "instance-group-manager"
)
