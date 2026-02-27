---
title: GCP Artifact Registry Docker Image
sidebar_label: gcp-artifact-registry-docker-image
---

A GCP Artifact Registry Docker Image represents a single container image stored within Google Cloud Artifact Registry. Artifact Registry is Google Cloud’s fully-managed, secure, and scalable repository service that allows teams to store, manage and secure their build artefacts, including Docker container images. Each Docker image is identified by its path in the form `projects/{project}/locations/{location}/repositories/{repository}/dockerImages/{image}` and can hold multiple tags and versions. Managing images through Artifact Registry enables fine-grained IAM permissions, vulnerability scanning, and seamless integration with Cloud Build and Cloud Run.  
For more information, see the official documentation: https://cloud.google.com/artifact-registry/docs/docker

**Terrafrom Mappings:**

* `google_artifact_registry_docker_image.name`

## Supported Methods

* `GET`: Get a gcp-artifact-registry-docker-image by its "locations|repositories|dockerImages"
* ~~`LIST`~~
* `SEARCH`: Search for Docker images in Artifact Registry. Use the format "location|repository_id" or "projects/[project]/locations/[location]/repository/[repository_id]/dockerImages/[docker_image]" which is supported for terraform mappings.
