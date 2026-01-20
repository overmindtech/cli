# GCP Source Setup for Overmind

This repository provides tools to set up the necessary GCP permissions for the Overmind service account to inspect your GCP project resources.

## Purpose

When setting up a GCP source in Overmind, you need to grant specific permissions to the Overmind service account. This repository contains a script that automates this process, ensuring the Overmind service account has the proper access to collect information about your GCP resources.

## Permissions Granted

The script grants several read-only IAM roles to the Overmind service account. These permissions allow Overmind to inspect your GCP resources without making any changes to your project.

For the exact permissions being granted, please refer to the [roles file](./overmind-gcp-roles.sh).

These permissions allow Overmind to:
- Inspect your GCP resources and their configurations
- Review IAM permissions and security settings
- Access resource hierarchy information

The permissions are read-only and do not allow Overmind to make any changes to your GCP project.

## Usage

You can run the script directly in Google Cloud Shell by clicking the button below:

[![Open in Cloud Shell](https://gstatic.com/cloudssh/images/open-btn.svg)](https://shell.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/overmindtech/gcp-source-setup.git&cloudshell_open_in_editor=README.md&ephemeral=true&show=terminal&cloudshell_tutorial=tutorial.md)

Alternatively, you can run the script manually within your terminal after cloning the repository:

```bash
./overmind-gcp-source-setup.sh <project-id> <overmind-service-account-email>
```

The script will expect two arguments:
- `<project-id>`: Your GCP project ID where Overmind will inspect resources.
- `<overmind-service-account-email>`: The email address of the Overmind service account that will be granted permissions.

## Complete Guide

For a complete guide on setting up and configuring the GCP source in Overmind, please refer to the official documentation:
[Overmind GCP Source Configuration Guide](https://docs.overmind.tech/sources/gcp/configuration)
