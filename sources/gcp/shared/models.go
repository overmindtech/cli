package shared

import (
	"github.com/overmindtech/cli/sources/shared"
)

const GCP shared.Source = "gcp"

// APIs
const (
	Compute         shared.API = "compute"
	Container       shared.API = "container"
	NetworkSecurity shared.API = "network-security"
	NetworkServices shared.API = "network-services"
)

// Resources
const (
	Instance                 shared.Resource = "instance"
	Cluster                  shared.Resource = "cluster"
	Disk                     shared.Resource = "disk"
	Network                  shared.Resource = "network"
	NodeGroup                shared.Resource = "node-group"
	NodeTemplate             shared.Resource = "node-template"
	Subnetwork               shared.Resource = "subnetwork"
	Address                  shared.Resource = "address"
	ForwardingRule           shared.Resource = "forwarding-rule"
	BackendService           shared.Resource = "backend-service"
	Autoscaler               shared.Resource = "autoscaler"
	InstanceGroupManager     shared.Resource = "instance-group-manager"
	SecurityPolicy           shared.Resource = "security-policy"
	ClientTlsPolicy          shared.Resource = "client-tls-policy"
	ServiceLbPolicy          shared.Resource = "service-lb-policy"
	ServiceBinding           shared.Resource = "service-binding"
	InstanceTemplate         shared.Resource = "instance-template"
	RegionalInstanceTemplate shared.Resource = "regional-instance-template"
	InstanceGroup            shared.Resource = "instance-group"
	TargetPool               shared.Resource = "target-pool"
	ResourcePolicy           shared.Resource = "resource-policy"
)
