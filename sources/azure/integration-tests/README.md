# Running Integration Tests for Azure

Integration tests are defined in an individual file for each resource.
Test names follow the pattern `Test<API><RESOURCE>Integration`, where `<API>` is the API name and `<RESOURCE>` is the resource name.
For example, `TestComputeVirtualMachineIntegration` tests the Compute API's VirtualMachine resource.

## Setup your local environment for testing

1. Create your Azure account here, `https://portal.azure.com/`
2. Use your brex credit card information
3. You can see the other overmind subscriptions, they will be under Subscriptions in the Azure portal.
4. Login to Azure CLI `az login` on the terminal.
5. To run the **integration tests in debug mode** you need to set the following environment variables. `~/.config/Cursor/User/settings.json`

    ```json
    {
        "window.commandCenter": true,
        "workbench.activityBar.orientation": "vertical",
        "go.testEnvVars": {
            "RUN_AZURE_INTEGRATION_TESTS": "true",
            "AZURE_SUBSCRIPTION_ID": "your-subscription-id",
            "AZURE_TENANT_ID": "your-tenant-id",
            "AZURE_CLIENT_ID": "your-client-id",
            "AZURE_INTEGRATION_TEST_RUN_ID": "local-dev-1"
        }
    }
    ```

    > **Note:** Replace the placeholder values with your own Azure subscription ID, tenant ID, and client ID.
   Or you can run them in the CLI by using:

   ```bash
    export RUN_AZURE_INTEGRATION_TESTS=true
    # For Azure
    export AZURE_SUBSCRIPTION_ID="your-subscription-id"  # your Azure subscription ID
    export AZURE_TENANT_ID="your-tenant-id"              # your Azure AD tenant ID
    export AZURE_CLIENT_ID="your-client-id"             # your Azure application/client ID
    export AZURE_REGIONS="eastus,westus2"                # optional: comma-separated list of regions
    export AZURE_INTEGRATION_TEST_RUN_ID="local-dev-1"   # optional: isolate this run's resource group
    # For SQL Database integration tests
    export AZURE_SQL_SERVER_ADMIN_LOGIN="sqladmin"       # SQL server administrator login
    export AZURE_SQL_SERVER_ADMIN_PASSWORD="your-secure-password"  # SQL server administrator password
    # For PostgreSQL Flexible Server integration tests
    export AZURE_POSTGRESQL_SERVER_ADMIN_LOGIN="pgadmin"       # PostgreSQL Flexible Server administrator login
    export AZURE_POSTGRESQL_SERVER_ADMIN_PASSWORD="your-secure-password"  # PostgreSQL Flexible Server administrator password
   ```

6. Integration tests are using Azure SDK for Go to interact with Azure resources. For local development, you can authenticate using:
   - **Azure CLI**: `az login` (recommended for local development)
   - **Service Principal**: Set `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`, and `AZURE_SUBSCRIPTION_ID` environment variables
   - **Managed Identity**: When running in Azure (automatically detected)
   - **Workload Identity Federation**: When running in Kubernetes/EKS (automatically detected via federated credentials)

   See the [official documentation](https://learn.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication) for more authentication options.
7. **optional** You may need to set the default subscription `az account set --subscription "your-subscription-id"`. Use your own subscription ID here.
8. You can now run integration tests.

Each test has `Setup`, `Run`, and `Teardown` methods.

- `Setup` is used to create any resources needed for the test.
- `Run` is where the actual test logic is implemented.
- `Teardown` is used to clean up any resources created during the test.

The `Setup` and `Teardown` methods are idempotent, meaning they can be run multiple times without causing issues. This allows for flexibility in running tests in different orders or multiple times.

We can easily run all `Setup` tests to create resources, then run all `Run` tests to execute the actual tests, and finally run all `Teardown` tests to clean up resources.

**Run after Setup:** `Run` subtests skip with a clear message when `Setup` did not complete successfully (for example Setup was skipped, failed, or you ran only `Run` without a prior successful Setup). That avoids noisy failures that are not adapter bugs.

### Skips, quotas, and slow Azure operations

Some tests intentionally call `t.Skip` for Azure conditions that are external to adapter correctness, for example:

- Batch account quota exhaustion (`SubscriptionQuotaExceeded`)
- **Gallery application version** (`compute-gallery-application-version_test.go`): requires env vars `AZURE_TEST_GALLERY_NAME`, `AZURE_TEST_GALLERY_APPLICATION_NAME`, and `AZURE_TEST_GALLERY_APPLICATION_VERSION` pointing at an existing gallery application version; if the version is missing (`404`), the test skips after preflight
- **Role assignments** (`authorization-role-assignment_test.go`): may wait for RBAC eventual consistency before asserting adapter behaviour

VM/VMSS/role-assignment ghost `409 Conflict` states are now handled with "auto-remediate then fail": tests attempt cleanup and a retry, and fail loudly if the resource is still unrecoverable.

Also note that PostgreSQL Flexible Server creation and Key Vault purge/recreate can take many minutes. If a run times out, increase `go test -timeout` (for example `-timeout 30m`) before assuming the test is stuck.

From the `sources/azure/integration-tests` directory:

For building up the infra for the Compute API resources.

```bash
go test ./integration-tests -run "TestCompute.*/Setup" -v
```

For running the actual tests for the Compute API resources.

```bash
go test ./integration-tests -run "TestCompute.*/Run" -v
```

For tearing down the infra for the Compute API resources.

```bash
go test ./integration-tests -run "TestCompute.*/Teardown" -v
```

## Running Integration Tests via Cloud Agents

Cursor Cloud Agents can run Azure integration tests autonomously when configured with the correct credentials.

### Prerequisites

1. **1Password vault**: Azure credentials are stored in the "cursor" 1Password vault under the item "Azure Integration Tests"
2. **Cursor Cloud Agent secret**: Configure only `OP_SERVICE_ACCOUNT_TOKEN` in `https://cursor.com/dashboard/cloud-agents`
3. **Repo env files**: `op.azure-cloud-agent.secret` and `op.azure-cloud-agent.env` exist with required `op://...` references

### How it works

When a Cloud Agent picks up a Linear issue to create an Azure adapter:

1. Cursor injects `OP_SERVICE_ACCOUNT_TOKEN` into the Cloud Agent environment
2. `inject-secrets` reads `op://...` references from env files using the 1Password SDK
3. `inject-secrets` writes resolved values to a local env file
4. The shell sources that file before test execution
5. The `DefaultAzureCredential` chain picks up `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, and `AZURE_TENANT_ID` from environment
6. Integration tests use `AZURE_SUBSCRIPTION_ID` and `RUN_AZURE_INTEGRATION_TESTS=true`

To inject credentials manually (e.g. for debugging), run:

```bash
go run build/inject-secrets/main.go \
  --no-ping \
  --secret-file .github/env/op.azure-cloud-agent.secret \
  --env-file .github/env/op.azure-cloud-agent.env \
  --output-file .env.azure-cloud-agent

set -a
source .env.azure-cloud-agent
set +a
```

### Security

- The service principal has **read-write access** scoped to the integration test subscription only
- Cloud Agent dashboard stores only the bootstrap token (`OP_SERVICE_ACCOUNT_TOKEN`)
- Azure credentials remain in 1Password and are resolved only at runtime via `inject-secrets`
- By default test resources are created in `overmind-integration-tests`; set `AZURE_INTEGRATION_TEST_RUN_ID` to isolate parallel runs into per-run resource groups (for example `overmind-integration-tests-agent-42`)
- Teardown steps clean up created resources after each test run
