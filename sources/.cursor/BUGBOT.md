# Source-Specific Rules

## New Adapters

When reviewing newly created adapters, it is extremely important to ensure that all of the `LinkedItemQueries` that could be added have been added. The way this is done is by looking at the method in which we translate from the cloud provider's data type to an `sdp.Item`. You should look at the definition of the cloud provider's type. This will almost always be a struct with fields, which are quite often other nested structs. What you should do is go through every field in that struct and its children, and see whether it is likely that those fields reference other cloud resources that we could potentially create a link to. Doesn't matter whether or not we have created the adapter for that type of cloud resource yet. We should always create as many links as possible. If it is another cloud resource that we are likely to also create an adapter for at some point.

There are also a couple of generic types that we should always create links for if the attributes are there. These are:

* `ip`: Any attribute that would contain an IP address should create a LinkedItemQueries for an `ip` type. This should always use the scope of global, the method of GET and a query of the IP address itself
* `dns`: any attribute that contains a DNS name should create a LinkedItemQueries for a DNS type.  The type should be `dns`, scope `global`, method SEARCH with the query being the DNS name itself

## IAMPermissions and PredefinedRole

Every adapter must implement both `IAMPermissions()` and `PredefinedRole()`:

* `IAMPermissions()` must return at least one permission string following the pattern `Microsoft.{Provider}/{resourcePath}/read`. The resource path must match the ARM resource type for the resource being adapted. For child resources, include the full path (e.g., `Microsoft.Batch/batchAccounts/applications/versions/read`, not just `Microsoft.Batch/batchAccounts/read`). The method should have a comment linking to the relevant Azure RBAC resource provider operations page.
* `PredefinedRole()` must return a non-empty string naming a valid Azure built-in role. If the service area has a specific reader role (e.g., `"Azure Batch Account Reader"` for Batch, `"Storage Blob Data Reader"` for Storage), use that. Otherwise, `"Reader"` is acceptable as the most restrictive general role.

Flag any adapter missing either method, returning empty values, or using an incorrect resource provider path.

## Azure ARM Get/List options

For Azure adapters, only pass `*Options` fields (for example `$expand`) that the REST API for that resource and API version documents. Unsupported or mistyped query parameters can surface as `400 Bad Request` from malformed URLs. When in doubt, prefer `nil` options or cross-check the official REST reference for the operation.

## PotentialLinks Completeness

`PotentialLinks()` must include every resource type that appears in any `LinkedItemQuery` returned by the adapter's conversion function. If the adapter creates linked item queries for IP addresses, `PotentialLinks()` must include `stdlib.NetworkIP`. If it creates queries for DNS names, `PotentialLinks()` must include `stdlib.NetworkDNS`. Missing entries in `PotentialLinks()` break the Overmind dependency graph — linked items won't be discovered even though the queries exist in the adapter's output.
