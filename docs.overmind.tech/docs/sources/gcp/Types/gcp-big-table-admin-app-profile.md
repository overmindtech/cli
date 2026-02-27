---
title: GCP Big Table Admin App Profile
sidebar_label: gcp-big-table-admin-app-profile
---

A Bigtable **App Profile** is a logical configuration that tells Google Cloud Bigtable how client traffic for a particular application should be routed to one or more clusters within an instance. It lets you choose between single-cluster routing (for the lowest latency within a specific region) or multi-cluster routing (for higher availability across several regions) and also defines the consistency model that the application will see. Because app profiles govern the path that live data takes, mis-configuration can lead to increased latency, unexpected fail-over behaviour, or cross-region egress costs.  
Official documentation: https://cloud.google.com/bigtable/docs/app-profiles

**Terrafrom Mappings:**

* `google_bigtable_app_profile.id`

## Supported Methods

* `GET`: Get a gcp-big-table-admin-app-profile by its "instances|appProfiles"
* ~~`LIST`~~
* `SEARCH`: Search for BigTable App Profiles in an instance. Use the format "instance" or "projects/[project_id]/instances/[instance_name]/appProfiles/[app_profile_id]" which is supported for terraform mappings.

## Possible Links

### [`gcp-big-table-admin-cluster`](/sources/gcp/Types/gcp-big-table-admin-cluster)

An App Profile points client traffic towards one or more specific clusters. Each routing policy within the profile references the cluster identifiers defined by `gcp-big-table-admin-cluster`. Observing this link lets you see which clusters will receive traffic from the application and assess redundancy or regional placement risks.

### [`gcp-big-table-admin-instance`](/sources/gcp/Types/gcp-big-table-admin-instance)

Every App Profile exists inside a single Bigtable instance. Linking to `gcp-big-table-admin-instance` shows the broader configuration—such as replication settings and all clusters—that frames the context in which the App Profile operates.
