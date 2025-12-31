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
            "AZURE_CLIENT_ID": "your-client-id"
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
    # For SQL Database integration tests
    export AZURE_SQL_SERVER_ADMIN_LOGIN="sqladmin"       # SQL server administrator login
    export AZURE_SQL_SERVER_ADMIN_PASSWORD="your-secure-password"  # SQL server administrator password
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