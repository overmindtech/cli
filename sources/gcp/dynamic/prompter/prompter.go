package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// This executable produces an adapter authoring prompt by filling a template with
// user-provided parameters.
// Usage:
//   go run ./sources/gcp/dynamic/prompter -name monitoring-alert-policy -api https://... -type https://...
// -type is optional.

const baseTemplate = `
You are an expert in Google Cloud Platform (GCP).
The objective of this task is to create a new adapter for GCP {{NAME}}.

You should create a new file for it inside the "sources/gcp/dynamic/adapters" folder.
This file should be named "{{NAME}}.go".
This file should contain all the necessary code to implement the adapter for {{NAME}}: meta, category, item type, blast propagation, terraform mapping, etc.
You can inspire from similar files in the "sources/gcp/dynamic/adapters" directory.
You can also inspire from the "SDPAssetTypeToAdapterMeta" map inside the "sources/gcp/shared/adapter-meta.go" file, where there are various adapter meta declarations, if there is any relevant.
But, you should not add anything new to the "SDPAssetTypeToAdapterMeta" map, as it is a legacy place for adapter meta declarations and we are moving away from it.
Also, you can check the "sources/gcp/shared/terraform-mappings.go" and "sources/gcp/shared/blast-propagations.go" for inspiration, but you should not add anything new to these files, as they are also legacy places for terraform mappings and blast propagations, and we are moving away from them.
We are collecting all the adapter meta declarations in the "sources/gcp/dynamic/adapters" directory, so we can have a single place for all the adapters.

You should use the following official Google Cloud References:
- API Call structure: {{API_REF}}
{{TYPE_LINE}}

The order of the task should be:
- Define the scope of the adapter: project, region or zone. If the url contains "locations" path parameter, then it is most likely a project level adapter with "locations" key being part of the unique attributes.
- Every adapter must have a get endpoint function.
- If the official API supports listing, then the adapter can support either list or search. We should go for list if listing all resources within its scope does not require passing any other query parameter. We should go for search if we have to pass an additional parameter such as "location" to be able to query multiple resources. A shortcut maybe that if unique attributes field has more than one item, then it is most likely a search.
- For terraform mappings, we should find the official terraform reference. If the terraform resource has an item that can be used in the get function, then this is our mapping. If the terraform resource has an id starting with "projects/...", then we should use search. If adapter does not support search, we should create one.
- The blast propagation is very important. We should find the attributes and use dot notation to address them correctly. We should only create a blast propagation if the attribute value can be used to link to an existing or a potential another adapter. When defining the blast propagation we should ask the simple question of: "What happens to my resource if the other/linked resource is deleted/updated [IN], and the other way around for the [OUT] property. If the there is no adapter for the linked item, we should just define its type in the relevant files (sources/gcp/shared/item-types.go and sources/gcp/shared/models.go), so that we can reference it in the blast propagation. For the non-existing adapter name we should follow the naming conventions: gcp-<api>-<resource>, e.g. gcp-compute-subnetwork.
- If resource has an attribute such as status or state, we should add this todo note along with the attribute name: "TODO: https://linear.app/overmind/issue/ENG-631"
- Sometimes the resource type does not have the "name" attribute, but it has another attribute that can be used as the name. In this case we should use the "NameSelector" field to define which attribute to use as the name. An example can be found in the "sources/gcp/dynamic/adapters/dataproc-cluster.go".`

func main() {
	name := flag.String("name", "", "(required) adapter name, e.g. monitoring-alert-policy")
	api := flag.String("api-ref", "", "(required) GCP reference for API Call structure")
	typeRef := flag.String("type-ref", "", "(optional) GCP reference for Type Definition")
	flag.Parse()

	missing := []string{}
	if *name == "" {
		missing = append(missing, "-name")
	}
	if *api == "" {
		missing = append(missing, "-api-ref")
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Missing required flags: %s\n", strings.Join(missing, ", "))
		flag.Usage()
		os.Exit(2)
	}

	output := baseTemplate
	output = strings.ReplaceAll(output, "{{NAME}}", *name)
	output = strings.ReplaceAll(output, "{{API_REF}}", *api)
	if *typeRef != "" {
		output = strings.ReplaceAll(output, "{{TYPE_LINE}}", "- Type Definition "+*typeRef+"\n")
	} else {
		output = strings.ReplaceAll(output, "{{TYPE_LINE}}", "")
	}

	fmt.Println(output)
}
