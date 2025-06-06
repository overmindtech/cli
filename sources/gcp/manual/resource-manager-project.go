package manual

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var CloudResourceManagerProject = shared.NewItemType(gcpshared.GCP, gcpshared.CloudResourceManager, gcpshared.Project)
