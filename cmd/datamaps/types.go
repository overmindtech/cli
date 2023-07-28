package datamaps

import (
	"github.com/overmindtech/sdp-go"
)

//go:generate go run ../../extractmaps.go aws-source
//go:generate go run ../../extractmaps.go k8s-source

type TfMapData struct {
	// The overmind type name
	Type string

	// The method that the query should use
	Method sdp.QueryMethod

	// The field within the resource that should be queried for
	QueryField string

	// The scope for the query. This can be either `*`, `global` or a string
	// that includes interpolations in Terraform format i.e.
	// ${outputs.overmind_kubernetes_cluster_name}.${values.metadata.namespace}
	Scope string
}
