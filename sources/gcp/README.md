# Further Information for GCG Adapter Creation

Please refer to the [generic adapter creation documentation](../README.md) to learn about the generic adapter framework.

This document is to highlight the specific implementation details for the GCP adapters.

## Naming Conventions

To construct the name of the adapter, we need to identify the following elements:
- Source: Currently we defined this as `gcp`.
- API: The API name, e.g. `compute`, `storage`, `bigquery`, etc.
- Resource: The resource name, e.g. `instance`, `bucket`, `dataset`, etc.

Let's take the GCP Compute Instance as an example.

We can use the API explorer to get the correct API endpoint documentation: [API Explorer](https://developers.google.com/apis-explorer).

From the Compute API, the service BASE URL is `https://compute.googleapis.com`.
So, the API name is `compute`: The API name is the first part of the URL after the `https://` and before the `googleapis.com`.

Then we can navigate to the section for the [Instances](https://cloud.google.com/compute/docs/reference/rest/v1#rest-resource:-v1.instances).
It is in plural form, but in our adapter we use the singular form `instance`.

We define all these elements as constants.

The API and Resource type definitions are in the [gcp shared models file](./shared/models.go).

Then we define the type itself in the relevant adapter file: `sources/gcp/compute-instance.go`.

```go
var ComputeInstance   = shared.NewItemType(gcpshared.GCP, gcpshared.Compute, gcpshared.Instance)
```

## Scopes

Every adapter has a scope that guarantees that the connected external resource can be identified within that scope uniquely.
For more see [Scope](../../sdp/README.md).

For GCP, we define the following scopes:
- Project: If the connected GCP resource requires the `project_id` for retrieving the resource along with its unique identifier, we define the scope as `project_id`. For example [Compute Network](https://cloud.google.com/compute/docs/reference/rest/v1/networks/get).
- Region: If the connected GCP resource requires the `project_id` and `region` for retrieving the resource along with its unique identifier, we define the scope as `project_id.region`. For example [Compute Subnetwork](https://cloud.google.com/compute/docs/reference/rest/v1/subnetworks/get).
- Zone: If the connected GCP resource requires the `project_id` and `zone` for retrieving the resource along with its unique identifier, we define the scope as `project_id.zone`. For example [Compute Instance](https://cloud.google.com/compute/docs/reference/rest/v1/instances/get).

After deciding which scope to use, we can create the adapter by using the relevant Base struct which will construct the correct scope for us.
```go
// NewComputeInstance creates a new computeInstanceWrapper instance
func NewComputeInstance(client gcpshared.ComputeInstanceClient, projectID, zone string) sources.ListableWrapper {
	return &computeInstanceWrapper{
		client: client,
		ZonalBase: gcpshared.NewZoneBase( // <-- Use the ZoneBase struct
			projectID,
			zone,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			ComputeInstance,
		),
	}
}
```

## Linked Item Queries

### Simple Queries for the Same API

When defining a relation between two adapters, we need to answer the following questions:
- What is the type of the related item?
- What is the method to use to get the related item?: `sdp.QueryMethod_GET`, `sdp.QueryMethod_SEARCH`, `sdp.QueryMethod_LIST`
- What is the query string to pass to the selected method?
- What is the scope of the related item?: `project`, `region`, `zone`
- How is the relation between the two items?: `BlastPropagation`

In the following example, we define a relation between the `ComputeInstance` and `ComputeSubnetwork` adapters.
- We identify the `ComputeSubnetwork` adapter as the related item.
- We use the `sdp.QueryMethod_GET` method to get the related item. Because the attribute `subnetwork_name` can be used to get the `ComputeSubnetwork` resource. If it was an attribute that can be used for searching, we would use the `sdp.QueryMethod_SEARCH` method. By the time we are developing the adapter, the linked adapter may not be present. In that case, we have to research the linked adapter and make the correct judgement.
- We use the `subnetworkName` as the query string to pass to the `GET` method. Because its [SDK documentation](https://cloud.google.com/compute/docs/reference/rest/v1/subnetworks/get) states that we need to pass its `name` to get the resource. 
- We define the scope as `region` via the `gcpshared.RegionalScope(c.ProjectID(), region)` helper function. Because the `ComputeSubnetwork` resource is a regional resource. It requires the `project_id` and `region` along with its `name` to get the resource.
- We define the relation as `BlastPropagation` with `In: true` and `Out: false`. Because the adapter we define is the `ComputeInstance` adapter, we want to propagate the blast radius from the `ComputeInstance` to the `ComputeSubnetwork`. This means that if the `ComputeSubnetwork` is deleted, the `ComputeInstance` will be affected by that (`in:true`). But if the `ComputeInstance` is deleted, the `ComputeSubnetwork` will not be affected (`out:false`). The relation might not be that clear all the time. In this case we should err on to `true` side.
```go
&sdp.LinkedItemQuery{
    Query: &sdp.Query{
        Type:   ComputeSubnetwork.String(),
        Method: sdp.QueryMethod_GET,
        Query:  subnetworkName,
        // This is a regional resource
        Scope: gcpshared.RegionalScope(c.ProjectID(), region),
    },
    BlastPropagation: &sdp.BlastPropagation{
        In:  true,
        Out: false,
    },
}
```

### Composite Queries for Different APIs

When the related item is not in the same API as the adapter, we need to investigate how to get the related item.
In the case of creating a link to a crypto key version, first we need to find the [relevant API](https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions/get?rep_location=global#path-parameters).
It gives us the GET method to use, the `https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}`.

Now, we need to decide how the linked item will look like:
- API: `cloudkms`, first item of the domain after `https://` and before `googleapis.com`.
- Name: `cryptoKeyVersion`, which is the single version of the identifier for the resource.
- Scope: `Project` level. Because the `locations` in the url can be a region or zone, so it will be dynamically required from the query.

Putting together all this information:
- Linked Item Type: `CloudKMSCryptoKeyVersion.String()`: assuming that we defined this type in its own file for future.
- Linked Item Query: What we need to construct the full URL? ProjectID will come from the scope. We need to pass: `location`, `keyring`, `cryptoKey` and `cryptoKeyVersion`. So we need a helper function to extract these information and compose a query by constructing a string simply joining all these variables by our default query separator `|`. We can use the helper function from shared: `shared.CompositeLookupKey(location, keyring, cryptoKey, cryptoKeyVersion)`.
- Linked Item Scope: Project level, because the adapter for this type will have a project level scope.