---
title: Azure Configuration
sidebar_position: 1
---

# Azure Configuration

## Overview

Overmind's Azure infrastructure discovery provides visibility into your Microsoft Azure resources through secure, read-only access. Overmind uses an Azure AD App Registration with federated credentials (workload identity) when running the source for you—no client secrets are stored or entered in the UI.

To connect an Azure source, you need a **Name** (friendly label in Overmind), **Subscription ID**, **Tenant ID**, and **Client ID**. Overmind only ever requests read-only access (minimum **Reader** role on the subscription).

## Prerequisites

- **Azure subscription**: An active subscription you want to discover.
- **Azure AD App Registration**: An app registered in Azure AD with at least **Reader** role on the subscription (used for workload identity; no client secret is required in the Overmind UI).
- **Permissions**: Ability to create an App Registration and assign roles in the subscription (e.g. Owner or User Access Administrator).

## Where to get the IDs

You need three values from Azure. All are GUIDs.

### Subscription ID

- **Azure Portal:** In the portal, go to **Cost Management + Billing** → **Subscriptions** (or see [View subscriptions in the Azure portal](https://learn.microsoft.com/en-us/azure/cost-management-billing/manage/view-all-accounts)), select your subscription, and copy **Subscription ID**.
- **Azure CLI:** Run `az account show --query id -o tsv` (after `az login` and, if needed, `az account set --subscription "your-subscription-name-or-id"`).

### Tenant ID

- **Azure Portal:** See [Find your Azure AD tenant ID](https://learn.microsoft.com/en-us/azure/active-directory/fundamentals/active-directory-how-to-find-tenant) — in the portal, go to **Azure Active Directory** → **Overview** and copy **Tenant ID**.
- **Azure CLI:** Run `az account show --query tenantId -o tsv`.

### Client ID (Application ID)

- **Azure Portal:** See [Register an application](https://learn.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app) — in **Azure Active Directory** → **App registrations**, select your app (or create one) and copy **Application (client) ID**.
- **If you create a service principal via CLI:** The **appId** in the command output is your Client ID.

Your app must have at least **Reader** on the subscription. For Overmind’s managed source we use federated credentials (workload identity), so you do **not** need to create or paste a client secret in Overmind.

For detailed setup (e.g. App Registration, role assignment, federated credentials), see [Microsoft’s documentation on registering an application](https://learn.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app) and [Reader role](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles#reader).

## Add an Azure source in Overmind

1. In Overmind, go to **Settings** (profile menu) → **Sources** → **Add source** → **Azure**.
2. Enter a **Name** (e.g. "Production Azure") so you can identify the source in Overmind.
3. Enter **Subscription ID**, **Tenant ID**, and **Client ID** using the values from [Where to get the IDs](#where-to-get-the-ids) above.
4. (Optional) **Regions:** Select specific Azure regions to limit discovery. If you leave this empty, Overmind discovers resources in all regions in the subscription.
5. Click **Create source**.

The source will appear in your Sources list. Once the connection is established, its status will show as healthy and you can use it in Explore and change analysis.

## Check your sources

After you have configured a source, it will appear under [Settings → Sources](https://app.overmind.tech/settings/sources). There you can confirm the source is healthy and view its details (Source UUID, Subscription ID, Tenant ID, Client ID, and Regions).

## Explore your data

Once your Azure source is healthy, go to the [Explore page](https://app.overmind.tech/explore) to browse your Azure resources and their relationships.
