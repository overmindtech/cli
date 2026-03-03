---
title: GCP Essential Contacts Contact
sidebar_label: gcp-essential-contacts-contact
---

A **Google Cloud Essential Contact** represents an email address or Google Group that Google Cloud will use to send important notifications about incidents, security issues, and other critical updates for a project, folder, or organisation. Each contact is stored under a parent resource (e.g. `projects/123456789`, `folders/987654321`, or `organizations/555555555`) and can be categorised by notification types such as `SECURITY`, `TECHNICAL`, or `LEGAL`.  
For further details, refer to the official Google Cloud documentation: https://cloud.google.com/resource-manager/docs/reference/essentialcontacts/rest

**Terrafrom Mappings:**

- `google_essential_contacts_contact.id`

## Supported Methods

- `GET`: Get a gcp-essential-contacts-contact by its "name"
- `LIST`: List all gcp-essential-contacts-contact
- `SEARCH`: Search for contacts by their ID in the form of "projects/[project_id]/contacts/[contact_id]".
