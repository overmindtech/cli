package adapters

import "testing"

// Adapters without SupportedQueryMethods are left out of the type list the
// change-analysis investigator validates its queries against, so it cannot
// query them at all.
func TestAdapterMetadataDeclaresQueryMethods(t *testing.T) {
	t.Parallel()

	for _, metadata := range Metadata.AllAdapterMetadata() {
		if metadata.GetSupportedQueryMethods() == nil {
			t.Errorf("adapter %q has no SupportedQueryMethods", metadata.GetType())
		}
	}
}
