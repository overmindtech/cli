package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Essential Contacts Contact adapter for essential contacts
var _ = registerableAdapter{
	sdpType: gcpshared.EssentialContactsContact,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OTHER,
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference: https://cloud.google.com/resource-manager/docs/reference/essentialcontacts/rest/v1/projects.contacts/get
		// GET https://essentialcontacts.googleapis.com/v1/projects/*/contacts/*
		// IAM permissions: essentialcontacts.contacts.get
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://essentialcontacts.googleapis.com/v1/projects/%s/contacts/%s"),
		// Reference: https://cloud.google.com/resource-manager/docs/reference/essentialcontacts/rest/v1/projects.contacts/list
		// GET https://essentialcontacts.googleapis.com/v1/projects/*/contacts
		// IAM permissions: essentialcontacts.contacts.list
		ListEndpointFunc: gcpshared.ProjectLevelListFunc("https://essentialcontacts.googleapis.com/v1/projects/%s/contacts"),
		// This is a special case where we have to define the SEARCH method for only to support Terraform Mapping.
		// We only validate the adapter initiation constraint: whether the project ID is provided or not.
		// We return a nil EndpointFunc without any error, because in the runtime we will use the
		// GET endpoint for retrieving the item for Terraform Query.
		SearchEndpointFunc: func(adapterInitParams ...string) (gcpshared.EndpointFunc, error) {
			if len(adapterInitParams) != 1 || adapterInitParams[0] == "" {
				return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
			}

			return nil, nil
		},
		SearchDescription:   "Search for contacts by their ID in the form of \"projects/[project_id]/contacts/[contact_id]\".",
		UniqueAttributeKeys: []string{"contacts"},
		// HEALTH: https://cloud.google.com/resource-manager/docs/reference/essentialcontacts/rest/v1/folders.contacts#validationstate
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"essentialcontacts.contacts.get", "essentialcontacts.contacts.list"},
		PredefinedRole: "roles/essentialcontacts.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// There is no links for this item type.
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/essential_contacts_contact#email",
		Description: "id => {resourceType}/{resource_id}/contacts/{contact_id}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_essential_contacts_contact.id",
			},
		},
	},
}.Register()
