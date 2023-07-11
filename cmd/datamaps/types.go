package datamaps

import "github.com/overmindtech/sdp-go"

//go:generate go run ../../extractmaps.go aws-source

type TfMapData struct {
	Type       string
	Method     sdp.QueryMethod
	QueryField string
	Scope      string
}
