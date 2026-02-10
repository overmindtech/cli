---
title: GCP Big Table Admin App Profile
sidebar_label: gcp-big-table-admin-app-profile
---

A Bigtable **App Profile** is a logical wrapper that tells Cloud Bigtable _how_ an application’s traffic should be routed, which clusters it can use, and what fail-over behaviour to apply. By creating multiple app profiles you can isolate workloads, direct different applications to specific clusters, or enable multi-cluster routing for higher availability.  
For an in-depth explanation see the official documentation: https://cloud.google.com/bigtable/docs/app-profiles

**Terrafrom Mappings:**

- `google_bigtable_app_profile.id`

## Supported Methods

- `GET`: Get a gcp-big-table-admin-app-profile by its "instances|appProfiles"
- ~~`LIST`~~
- `SEARCH`: Search for BigTable App Profiles in an instance. Use the format "instance" or "projects/[project_id]/instances/[instance_name]/appProfiles/[app_profile_id]" which is supported for terraform mappings.

## Possible Links

### [`gcp-big-table-admin-cluster`](/sources/gcp/Types/gcp-big-table-admin-cluster)

Every app profile specifies one or more clusters that client traffic may reach. Therefore an App Profile is directly linked to the Bigtable Cluster(s) it can route requests to.

### [`gcp-big-table-admin-instance`](/sources/gcp/Types/gcp-big-table-admin-instance)

An App Profile always belongs to exactly one Bigtable Instance; it cannot exist outside that instance’s administrative scope.
