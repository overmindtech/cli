package adapters

import (
	"encoding/json"
	"io/fs"

	adapterdata "github.com/overmindtech/cli/docs.overmind.tech/docs/sources"
	"github.com/overmindtech/cli/go/sdp-go"
	log "github.com/sirupsen/logrus"
)

type catalogQueryMethods struct {
	Get               bool   `json:"get"`
	GetDescription    string `json:"getDescription"`
	List              bool   `json:"list"`
	ListDescription   string `json:"listDescription"`
	Search            bool   `json:"search"`
	SearchDescription string `json:"searchDescription"`
}

type catalogEntry struct {
	Type                  string              `json:"type"`
	Category              int32               `json:"category"`
	DescriptiveName       string              `json:"descriptiveName"`
	PotentialLinks        []string            `json:"potentialLinks"`
	SupportedQueryMethods catalogQueryMethods `json:"supportedQueryMethods"`
}

var adapterCatalog map[string]*catalogEntry

func init() {
	adapterCatalog = make(map[string]*catalogEntry)

	err := fs.WalkDir(adapterdata.Files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		data, readErr := adapterdata.Files.ReadFile(path)
		if readErr != nil {
			log.WithError(readErr).WithField("path", path).Warn("Failed to read adapter data file")
			return nil
		}

		var entry catalogEntry
		if jsonErr := json.Unmarshal(data, &entry); jsonErr != nil {
			log.WithError(jsonErr).WithField("path", path).Warn("Failed to parse adapter data file")
			return nil
		}

		if entry.Type != "" {
			adapterCatalog[entry.Type] = &entry
		}
		return nil
	})
	if err != nil {
		log.WithError(err).Error("Failed to walk embedded adapter data")
	}
}

// lookupAdapterMetadata returns AdapterMetadata for the given type by looking
// up the embedded catalog. Falls back to sensible defaults when the type is not
// in the catalog.
func lookupAdapterMetadata(itemType string, scopes []string) *sdp.AdapterMetadata {
	entry, ok := adapterCatalog[itemType]
	if !ok {
		return &sdp.AdapterMetadata{
			Type:            itemType,
			DescriptiveName: itemType,
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:    true,
				List:   true,
				Search: true,
			},
			Category: sdp.AdapterCategory_ADAPTER_CATEGORY_OTHER,
		}
	}

	potentialLinks := make([]string, len(entry.PotentialLinks))
	copy(potentialLinks, entry.PotentialLinks)

	return &sdp.AdapterMetadata{
		Type:            itemType,
		DescriptiveName: entry.DescriptiveName,
		SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
			Get:               entry.SupportedQueryMethods.Get,
			GetDescription:    entry.SupportedQueryMethods.GetDescription,
			List:              entry.SupportedQueryMethods.List,
			ListDescription:   entry.SupportedQueryMethods.ListDescription,
			Search:            entry.SupportedQueryMethods.Search,
			SearchDescription: entry.SupportedQueryMethods.SearchDescription,
		},
		PotentialLinks: potentialLinks,
		Category:       sdp.AdapterCategory(entry.Category),
	}
}
