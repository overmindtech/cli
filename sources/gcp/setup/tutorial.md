# GCP Source Setup Tutorial

## Overview

This tutorial will guide you through setting up the necessary permissions for the Overmind service account in your GCP project.

## Set up permissions

Let's set up the required permissions for the Overmind service account.

### Step 1: Run the permissions script

Run the shell command copied from the Overmind Create Source page.
It should look something like this:

```bash
./overmind-gcp-source-setup.sh <project-id> <overmind-service-account-email>
```

<walkthrough-footnote>
This script will set up the necessary IAM permissions for the Overmind service account to access your GCP resources.
</walkthrough-footnote>

### Step 2: Verify the permissions

After the script completes, you can verify that the permissions were set correctly by running the following command. The permission check script will automatically use the environment variables that were set by the setup script:

```bash
./overmind-gcp-source-permission-check.sh
```

This script will check if all the necessary permissions have been correctly assigned to the Overmind service account.

## What's Next

You have successfully set up the necessary permissions for the Overmind service account. You can now:

1. Close this Cloud Shell session
2. Return to the Overmind application to continue your setup process

<walkthrough-conclusion-trophy></walkthrough-conclusion-trophy>
