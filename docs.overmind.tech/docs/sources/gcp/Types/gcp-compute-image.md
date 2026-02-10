---
title: GCP Compute Image
sidebar_label: gcp-compute-image
---

A GCP Compute Image represents a bootable disk image in Google Compute Engine. Images capture the contents of a virtual machine’s root volume (operating system, installed packages, configuration files, etc.) and act as the template from which new persistent disks and VM instances are created. Teams use images to standardise the base operating-system layer across their fleet, speed up instance provisioning, and ensure consistency between environments. Modifying or deleting an image can therefore have an immediate impact on every workload that references it, including instance templates and managed instance groups.  
Official documentation: https://cloud.google.com/compute/docs/images

**Terrafrom Mappings:**

- `google_compute_image.name`

## Supported Methods

- `GET`: Get GCP Compute Image by "gcp-compute-image-name"
- `LIST`: List all GCP Compute Image items
- ~~`SEARCH`~~
