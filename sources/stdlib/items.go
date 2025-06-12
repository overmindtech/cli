package stdlib

import (
	"github.com/overmindtech/cli/sources/shared"
	stdlibshared "github.com/overmindtech/cli/sources/stdlib/shared"
)

var (
	NetworkIP  = shared.NewItemType(stdlibshared.Stdlib, stdlibshared.Network, stdlibshared.IP)
	NetworkDNS = shared.NewItemType(stdlibshared.Stdlib, stdlibshared.Network, stdlibshared.DNS)
)
