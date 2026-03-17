---
title: GCP Configuration
sidebar_position: 1
---

# GCP Configuration

## Overview

Overmind's GCP infrastructure discovery provides comprehensive visibility into your Google Cloud Platform resources through secure, read-only access using Google Cloud's native IAM system.

Overmind supports two authentication methods:

1. **Direct Access** (Default) - Grant permissions directly to the Overmind service account
2. **Service Account Impersonation** (Optional) - Create your own service account with permissions, then allow Overmind to impersonate it

Both methods provide the same functionality and security. Choose the method that fits your organization's security policies.

### Authentication Methods Comparison

**Direct Access:**

- Simplest setup - grant roles directly to Overmind's service account
- Best for quick setup and straightforward security requirements

**Service Account Impersonation:**

- Enhanced control - you create and manage your own service account
- Better for organizations requiring all service accounts to be internally managed
- Provides dual identity in audit logs (both Overmind's SA and your SA)
- Learn more: [GCP Service Account Impersonation](https://cloud.google.com/iam/docs/service-account-impersonation)

### Why Service Account-Based Access?

Each customer receives a unique Overmind service account with minimal, read-only permissions. All access is logged through Google Cloud's audit system, giving you complete control with no shared credentials. This aligns with [Google Cloud's security best practices](https://cloud.google.com/security/best-practices).

## Prerequisites

Before beginning setup, ensure you have:

- **GCP Resource Access**: Appropriate IAM admin permissions at the organization, folder, or project level to grant IAM roles (and create service accounts for impersonation)
- **Required Tools**: One of the following:
  - [Google Cloud CLI (`gcloud`)](https://cloud.google.com/sdk/docs/install) installed and authenticated
  - Terraform with the Google Cloud Provider configured
- **Parent Resource**: The parent resource ID where Overmind will discover resources. This can be:
  - An organization: `organizations/123456789`
  - A folder: `folders/987654321`
  - A project: `projects/my-project-id`
- **Regional Scope**: List of GCP regions where your resources are located (mandatory for source configuration)

### Authentication Setup

Ensure your local environment is authenticated with Google Cloud:

```bash
# Authenticate with Google Cloud
gcloud auth login

# Set your default project (if using a project as parent)
gcloud config set project YOUR_PROJECT_ID

# Verify authentication
gcloud auth list
```

For Terraform users, configure [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/application-default-credentials):

```bash
gcloud auth application-default login
```

## Quick Start

### Step 1: Create Your Overmind GCP Source

1. Navigate to **Settings** > **Sources** > **Add Source** > **GCP** in the Overmind application
2. Configure your source:
   - **Parent ID**: The parent resource to discover from. Format:
     - Organization: `organizations/123456789`
     - Folder: `folders/987654321`
     - Project: `projects/my-project-id`
   - **Name**: A descriptive name for this source (optional)
   - **Regions**: Select the regions where your resources are located (mandatory)
   - **Impersonation** (optional): Toggle on to use service account impersonation
     - Enter the email of the service account you'll create (e.g., `overmind-reader@your-project.iam.gserviceaccount.com`)
     - Use any unique name for your service account
3. Click **Create Source**

You'll be redirected to the source details page showing:

- The Overmind service account email (e.g., `C-xxxxx@ovm-production.iam.gserviceaccount.com`)
- Configuration instructions customized for your setup
- Whether impersonation is enabled

### Step 2: Grant Permissions

The source details page provides customized scripts for your setup. These scripts automatically apply IAM permissions at the level you specified (organization, folder, or project). Permissions granted at a parent level are inherited by all child resources.

Choose your preferred method:

#### Option A: Cloud Shell (Easiest)

Click the **"Open in Google Cloud Shell"** button shown on the source details page. This provides you with the scripts and guidance needed to complete the setup. Follow the instructions in Cloud Shell to run the appropriate setup script for your configuration.

#### Option B: Manual Script

Copy and run the bash script shown on the source details page. The script automatically detects whether you're using an organization, folder, or project parent and applies the correct `gcloud` commands. The script varies based on whether impersonation is enabled:

**For Direct Access:**

- Grants read-only roles directly to the Overmind service account at your specified parent level
- For project-level parents, also creates a custom role for additional permissions

**For Impersonation:**

- Grants read-only roles to your service account at your specified parent level (you must create the service account manually first)
- For project-level parents, also creates a custom role for additional permissions
- Grants Overmind's service account permission to impersonate yours (`roles/iam.serviceAccountTokenCreator`)

#### Option C: Terraform

Copy the Terraform configuration shown on the source details page and apply it:

```bash
terraform init
terraform plan
terraform apply
```

### Step 3: Verify Source Status

1. Navigate to **Settings** > **Sources** in the Overmind application
2. Verify your GCP source shows as **Healthy**

## Required Permissions

Overmind requires read-only IAM roles for infrastructure discovery. See the [Required GCP Roles Reference](#required-gcp-roles-reference) for the complete list.

### Permission Flow

Permissions can be applied at any level of the GCP resource hierarchy and are inherited by child resources:

**Direct Access:**

```text
Your GCP Organization/Folder/Project
  └─ Overmind Service Account
      └─ Granted: Viewer roles (+ custom role for project-level)
          └─ Inherited by all child folders and projects
```

**Service Account Impersonation:**

```text
Your GCP Organization/Folder/Project
  ├─ Your Service Account
  │   └─ Granted: Viewer roles (+ custom role for project-level)
  │       └─ Inherited by all child folders and projects
  └─ Overmind Service Account
      └─ Granted: roles/iam.serviceAccountTokenCreator on Your Service Account
```

## Switching Between Authentication Methods

### Enable Impersonation

1. Create a service account in your GCP project (if you haven't already)
2. Grant it the required read-only roles and impersonation permission (use the scripts from the source details page - they handle both)
3. Edit your source in Overmind: enable **Impersonation** and enter your service account email
4. (Optional) Remove direct permissions from Overmind's service account

### Disable Impersonation

1. Edit your source in Overmind: disable **Impersonation** (this updates the scripts on the source details page)
2. Grant the required read-only roles directly to Overmind's service account (use the updated scripts from the source details page)
3. (Optional) Remove the impersonation permission and delete your service account

## Validation

### Verify IAM Permissions

**Using Google Cloud Console:**

1. Navigate to [IAM & Admin > IAM](https://console.cloud.google.com/iam-admin/iam)
2. Select your organization, folder, or project
3. Search for the service account (Overmind's or yours, depending on setup)
4. Verify all required roles are listed

**Using Google Cloud CLI:**

For direct access at organization level:

```bash
gcloud organizations get-iam-policy YOUR_ORG_ID \
  --flatten="bindings[].members" \
  --format="table(bindings.role)" \
  --filter="bindings.members:serviceAccount:OVERMIND_SA_EMAIL"
```

For direct access at folder level:

```bash
gcloud resource-manager folders get-iam-policy YOUR_FOLDER_ID \
  --flatten="bindings[].members" \
  --format="table(bindings.role)" \
  --filter="bindings.members:serviceAccount:OVERMIND_SA_EMAIL"
```

For direct access at project level:

```bash
gcloud projects get-iam-policy YOUR_PROJECT_ID \
  --flatten="bindings[].members" \
  --format="table(bindings.role)" \
  --filter="bindings.members:serviceAccount:OVERMIND_SA_EMAIL"
```

For impersonation (verify Overmind can impersonate your SA):

```bash
gcloud iam service-accounts get-iam-policy YOUR_SA_EMAIL \
  --project=YOUR_PROJECT_ID \
  --flatten="bindings[].members" \
  --format="table(bindings.role,bindings.members)" \
  --filter="bindings.members:serviceAccount:OVERMIND_SA_EMAIL"
```

### Test Source Discovery

1. Navigate to **Explore** in the Overmind application
2. Run a query: GCP sources are prefixed with `gcp-`
   - To list all VMs: `gcp-compute-instance` > `LIST`
3. Verify resources are being discovered

### Validate Regional Coverage

Review the **Regions** configuration in your source settings and verify discovered resources match your expected regional distribution.

## Troubleshooting

### Common Issues

**"Insufficient Permissions" Error**

Verify all required roles are assigned at the appropriate level:

```bash
# For organization-level access
gcloud organizations get-iam-policy YOUR_ORG_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:SA_EMAIL"

# For folder-level access
gcloud resource-manager folders get-iam-policy YOUR_FOLDER_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:SA_EMAIL"

# For project-level access
gcloud projects get-iam-policy YOUR_PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:SA_EMAIL"
```

Re-run the setup script or check for organization-level policies restricting service account access.

**No Resources Discovered**

1. Verify regional configuration matches where your resources exist
2. For project-level parents, check that required GCP APIs are enabled:

   ```bash
   gcloud services list --enabled --project=YOUR_PROJECT_ID
   ```

3. For organization or folder-level parents, verify that you have the necessary permissions to list projects and that child projects have the required APIs enabled
4. Some resources may require additional permissions at different levels of the hierarchy

**Service Account Impersonation Fails**

1. Verify the impersonation permission is granted:

   ```bash
   gcloud iam service-accounts get-iam-policy YOUR_SA_EMAIL --project=YOUR_PROJECT_ID
   ```

   You should see Overmind's service account with `roles/iam.serviceAccountTokenCreator`.

2. Verify your service account exists and isn't disabled:

   ```bash
   gcloud iam service-accounts describe YOUR_SA_EMAIL --project=YOUR_PROJECT_ID
   ```

3. Ensure the service account email in Overmind matches exactly

4. Wait for propagation: IAM policy changes can take a few minutes to propagate. Wait 2-5 minutes after granting permissions before testing.

5. Check organization policies: Some organization policies may restrict service account impersonation.

**Service Account Not Found**

1. Verify you copied the correct email from the Overmind application
2. Ensure the email format is correct (ends with `.iam.gserviceaccount.com`)
3. For impersonation: verify your service account was created successfully
4. Contact [Overmind support](https://docs.overmind.tech/misc/support) if issues persist

**Terraform Apply Failures**

1. Verify authentication: `gcloud auth application-default print-access-token`
2. Ensure your credentials have necessary IAM permissions
3. For impersonation: ensure you have `iam.serviceAccounts.create` permission

### Getting Help

If you continue to experience issues, contact [Overmind support](https://docs.overmind.tech/misc/support) with:

- Your GCP parent resource (organization/folder/project ID)
- The Overmind service account email
- Your service account email (if using impersonation)
- Whether you're using direct access or impersonation
- The parent level you're configuring (organization, folder, or project)
- Specific error messages and screenshots

## Security Considerations

### Principle of Least Privilege

All roles are read-only and do not allow:

- Resource modification or deletion
- Data access (beyond metadata)
- Configuration changes
- Administrative operations

### Monitoring and Auditing

1. Enable [Cloud Audit Logs](https://cloud.google.com/logging/docs/audit) for your project
2. Monitor service account activity in audit logs
3. Configure alerts for unusual behavior

**Impersonation Audit Benefits:**
With impersonation, audit logs show both Overmind's identity and your service account identity, providing enhanced traceability.

### Permission Management

- **Regular Review**: Periodically review granted permissions
- **Revocation**: Remove access anytime:
  - **Direct access**: Remove IAM bindings
  - **Impersonation**: Remove `serviceAccountTokenCreator` role or disable/delete your service account

## Required Permissions

Overmind requires read-only access to discover and map your GCP infrastructure. The setup scripts provided in the Overmind application automatically grant all necessary permissions.

### What Gets Configured

**Essential role for resource discovery:**

- `roles/browser` - Required for listing projects and navigating the resource hierarchy

**Read-only viewer roles** for GCP services including:

- Compute Engine, GKE, Cloud Run, Cloud Functions, Dataflow
- Cloud SQL, BigQuery, Spanner, Cloud Storage
- IAM, networking, monitoring, and logging resources
- And other GCP services

**A custom role** with additional permissions for:

- BigQuery data transfer configurations
- Spanner database details

**Project-level-only roles** (only applied when using `projects/` parent):

- `roles/iam.roleViewer` - View IAM roles
- `roles/iam.serviceAccountViewer` - View service accounts

> **Note:** Some GCP IAM roles can only be granted at the project level, not at the organization or folder level. When configuring at the organization or folder level, these project-specific roles are automatically excluded. The custom role and project-level IAM roles are only created and assigned when using a project-level parent (e.g., `projects/my-project`).

**For impersonation** (if enabled):

- `roles/iam.serviceAccountTokenCreator` - Allows Overmind to impersonate your service account

All permissions are read-only and do not allow resource modification, deletion, or access to data beyond metadata.

The complete list of roles is included in the setup scripts shown in your source details page. These scripts are automatically updated as Overmind adds support for new GCP services and adapt based on whether you're configuring at the organization, folder, or project level.

## Required GCP Roles Reference

Here are all the predefined GCP roles that Overmind requires, plus the custom role for additional permissions:

### Predefined Roles

| Role                                    | Purpose                                                                                                                                                       |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `roles/browser`                         | **Required:** List projects and navigate resource hierarchy [GCP Docs](https://cloud.google.com/iam/docs/understanding-roles#browser)                         |
| `roles/aiplatform.viewer`               | AI Platform resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/aiplatform#aiplatform.viewer)                                   |
| `roles/artifactregistry.reader`         | Artifact Registry repository discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/artifactregistry#artifactregistry.reader)               |
| `roles/bigquery.metadataViewer`         | BigQuery metadata discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/bigquery#bigquery.metadataViewer)                                  |
| `roles/bigquery.user`                   | BigQuery data transfer discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/bigquery#bigquery.user)                                       |
| `roles/bigtable.viewer`                 | Cloud Bigtable resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/bigtable#bigtable.viewer)                                    |
| `roles/cloudbuild.builds.viewer`        | Cloud Build resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/cloudbuild#cloudbuild.builds.viewer)                            |
| `roles/cloudfunctions.viewer`           | Cloud Functions discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/cloudfunctions#cloudfunctions.viewer)                                |
| `roles/cloudkms.viewer`                 | Cloud KMS resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/cloudkms#cloudkms.viewer)                                         |
| `roles/cloudsql.viewer`                 | Cloud SQL instance discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/cloudsql#cloudsql.viewer)                                         |
| `roles/compute.viewer`                  | Compute Engine resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/compute#compute.viewer)                                      |
| `roles/container.viewer`                | GKE cluster and resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/container#container.viewer)                                 |
| `roles/dataform.viewer`                 | Dataform resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/dataform#dataform.viewer)                                          |
| `roles/dataplex.catalogViewer`          | Dataplex catalog resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/dataplex#dataplex.catalogViewer)                           |
| `roles/dataplex.viewer`                 | Dataplex resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/dataplex#dataplex.viewer)                                          |
| `roles/dataflow.viewer`                 | Dataflow job discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/dataflow#dataflow.viewer)                                               |
| `roles/dataproc.viewer`                 | Dataproc cluster discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/dataproc#dataproc.viewer)                                           |
| `roles/dns.reader`                      | Cloud DNS resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/dns#dns.reader)                                                   |
| `roles/essentialcontacts.viewer`        | Essential Contacts discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/essentialcontacts#essentialcontacts.viewer)                       |
| `roles/eventarc.viewer`                 | Eventarc trigger discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/eventarc#eventarc.viewer)                                           |
| `roles/file.viewer`                     | Cloud Filestore discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/file#file.viewer)                                                    |
| `roles/iam.roleViewer`                  | **Project-level only:** IAM role discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/iam#iam.roleViewer)                                 |
| `roles/iam.serviceAccountViewer`        | **Project-level only:** IAM service account discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/iam#iam.serviceAccountViewer)            |
| `roles/logging.viewer`                  | Cloud Logging resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/logging#logging.viewer)                                       |
| `roles/monitoring.viewer`               | Cloud Monitoring resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/monitoring#monitoring.viewer)                              |
| `roles/orgpolicy.policyViewer`          | Organization Policy discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/orgpolicy#orgpolicy.policyViewer)                                |
| `roles/pubsub.viewer`                   | Pub/Sub resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/pubsub#pubsub.viewer)                                               |
| `roles/redis.viewer`                    | Cloud Memorystore Redis discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/redis#redis.viewer)                                          |
| `roles/resourcemanager.tagViewer`       | Resource Manager tag discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/resourcemanager#resourcemanager.tagViewer)                      |
| `roles/run.viewer`                      | Cloud Run resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/run#run.viewer)                                                   |
| `roles/secretmanager.viewer`            | Secret Manager secret discovery (metadata only) [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/secretmanager#secretmanager.viewer)            |
| `roles/securitycentermanagement.viewer` | Security Center management discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/securitycentermanagement#securitycentermanagement.viewer) |
| `roles/servicedirectory.viewer`         | Service Directory resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/servicedirectory#servicedirectory.viewer)                 |
| `roles/serviceusage.serviceUsageViewer` | Service Usage discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/serviceusage#serviceusage.serviceUsageViewer)                          |
| `roles/spanner.viewer`                  | Cloud Spanner resource discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/spanner#spanner.viewer)                                       |
| `roles/storage.bucketViewer`            | Cloud Storage bucket discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/storage#storage.bucketViewer)                                   |
| `roles/storagetransfer.viewer`          | Storage Transfer Service discovery [GCP Docs](https://cloud.google.com/iam/docs/roles-permissions/storagetransfer#storagetransfer.viewer)                     |

### Custom Role

| Role                                             | Purpose                                                                                                                                                                                                                                                                 |
| ------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `projects/{PROJECT_ID}/roles/overmindCustomRole` | Custom role for additional BigQuery and Spanner permissions **Permissions:** `bigquery.transfers.get` - BigQuery transfer configuration discovery, `spanner.databases.get` - Spanner database detail discovery, `spanner.databases.list` - Spanner database enumeration |

All predefined roles provide read-only access and are sourced from Google Cloud's [predefined roles documentation](https://cloud.google.com/iam/docs/understanding-roles#predefined).

**Project-Level Restrictions:** Some roles (`roles/iam.roleViewer` and `roles/iam.serviceAccountViewer`) can only be granted at the project level in GCP. When configuring at the organization or folder level, these roles are automatically excluded. The custom role is also only created and assigned when using a project-level parent (e.g., `projects/my-project`).
