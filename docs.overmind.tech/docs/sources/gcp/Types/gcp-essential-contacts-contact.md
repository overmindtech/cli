---
title: GCP Essential Contacts Contact
sidebar_label: gcp-essential-contacts-contact
---

Google Cloud’s Essential Contacts service allows an organisation to register one or more e-mail addresses that will receive important operational and security notifications about a project, folder, or organisation. A “contact” resource represents a single recipient and records the e-mail address, preferred language and notification categories that the person should receive. More than one contact can be added so that the right teams are informed whenever Google issues mandatory or time-sensitive messages.  
For a full description of the resource and its fields, refer to the official documentation: https://cloud.google.com/resource-manager/docs/managing-notification-contacts

**Terrafrom Mappings:**

- `google_essential_contacts_contact.id`

## Supported Methods

- `GET`: Get a gcp-essential-contacts-contact by its "name"
- `LIST`: List all gcp-essential-contacts-contact
- `SEARCH`: Search for contacts by their ID in the form of "projects/[project_id]/contacts/[contact_id]".
