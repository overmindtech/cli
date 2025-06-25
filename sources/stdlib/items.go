package stdlib

import (
	"github.com/overmindtech/cli/sources/shared"
	stdlibshared "github.com/overmindtech/cli/sources/stdlib/shared"
)

type ItemType struct {
	shared.ItemTypeInstance
}

// String returns the string representation of the ItemType
// This is created for backwards compatibility
// Currently, it returns the resource name only without the source and API
func (i ItemType) String() string {
	return string(i.Resource)
}

var (
	NetworkIP  = ItemType{ItemTypeInstance: shared.NewItemType(stdlibshared.Stdlib, stdlibshared.Network, stdlibshared.IP)}
	NetworkDNS = ItemType{ItemTypeInstance: shared.NewItemType(stdlibshared.Stdlib, stdlibshared.Network, stdlibshared.DNS)}
)
