// Package adapterdata embeds the per-type adapter metadata JSON files so
// other packages can look up category, descriptive name, supported query
// methods, and potential links without duplicating the data.
package adapterdata

import "embed"

// Files contains every adapter JSON file under {provider}/data/*.json.
//
//go:embed */data/*.json
var Files embed.FS
