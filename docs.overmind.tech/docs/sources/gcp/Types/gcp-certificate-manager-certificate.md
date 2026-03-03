---
title: GCP Certificate Manager Certificate
sidebar_label: gcp-certificate-manager-certificate
---

A **GCP Certificate Manager Certificate** represents an SSL/TLS certificate that is stored and managed by Google Cloud Certificate Manager. Certificates configured here can be Google-managed (automatically provisioned and renewed by Google) or self-managed (imported by the user) and can be attached to load balancers, Cloud CDN, or other Google Cloud resources to provide encrypted connections. Managing certificates through Certificate Manager centralises lifecycle operations such as issuance, rotation and revocation, reducing operational overhead and the risk of serving expired certificates. For full details, see the official documentation: https://cloud.google.com/certificate-manager/docs

**Terrafrom Mappings:**

- `google_certificate_manager_certificate.id`

## Supported Methods

- `GET`: Get GCP Certificate Manager Certificate by "gcp-certificate-manager-certificate-location|gcp-certificate-manager-certificate-name"
- ~~`LIST`~~
- `SEARCH`: Search for GCP Certificate Manager Certificate by "gcp-certificate-manager-certificate-location"
