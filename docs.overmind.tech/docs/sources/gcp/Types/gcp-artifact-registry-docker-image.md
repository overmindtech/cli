---
title: GCP Artifact Registry Docker Image
sidebar_label: gcp-artifact-registry-docker-image
---

A GCP Artifact Registry Docker Image resource represents a single immutable image stored in Google Cloud’s Artifact Registry. It contains metadata such as the image digest, tags, size and creation timestamp, and can be queried to understand exactly which layers and versions are about to be deployed. Managing this resource allows you to verify provenance, scan for vulnerabilities and enforce policies before the image ever reaches production.  
For a full description of the REST resource, see Google’s official documentation: https://cloud.google.com/artifact-registry/docs/reference/rest/v1/projects.locations.repositories.dockerImages

**Terrafrom Mappings:**

- `google_artifact_registry_docker_image.name`

## Supported Methods

- `GET`: Get a gcp-artifact-registry-docker-image by its "locations|repositories|dockerImages"
- ~~`LIST`~~
- `SEARCH`: Search for Docker images in Artifact Registry. Use the format "location|repository_id" or "projects/[project]/locations/[location]/repository/[repository_id]/dockerImages/[docker_image]" which is supported for terraform mappings.
