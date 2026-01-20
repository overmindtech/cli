# Running Integration Tests for GCP

Integration tests are defined in an individual file for each resource.
Test names follow the pattern `Test<API><RESOURCE>Integration`, where `<API>` is the API name and `<RESOURCE>` is the resource name.
For example, `TestComputeInstanceIntegration` tests the Compute API's Instance resource.

## Setup your local environment for testing

1. Log in with your Google account here, `https://console.cloud.google.com/`
2. Use your brex credit card information to create a project and a billing account to use for integration tests.
3. You can see the other overmind projects, it will be under projects -> all.
4. Login to gcloud `gcloud auth login` on the terminal.
5. Enable the tested APIs:

    ```bash
    gcloud services enable \
        compute.googleapis.com \
        bigquery.googleapis.com \
        spanner.googleapis.com \
        cloudresourcemanager.googleapis.com \
        iam.googleapis.com \
        iamcredentials.googleapis.com \
        --project=integration-tests-484908
    ```

    > **Note:** `integration-tests-484908` is the project id of the shared project used for integration tests.

6. To run the **integration tests in debug mode** you need to set the following environment variables. `~/.config/Cursor/User/settings.json`

    ```json
    {
        "window.commandCenter": true,
        "workbench.activityBar.orientation": "vertical",
        "go.testEnvVars": {
            "RUN_GCP_INTEGRATION_TESTS": "true",
            "GCP_PROJECT_ID": "integration-tests-484908",
        }
    }
    ```

    > **Note:** `"integration-tests-484908"` is a shared project that is used for integration tests. Communicate on discord when you're using it, to avoid conflicts.
   Or you can run them in the CLI by using:

   ```bash
    export RUN_GCP_INTEGRATION_TESTS=true
    # For GCP
    export GCP_PROJECT_ID="integration-tests-484908" # use your own project id here
    export GCP_ZONE="us-central1-c"                 # not all tests need a zone
    export GCP_REGION="us-central1"                 # not all tests need a region
   ```

7. Integration tests are using Google Cloud Client Libraries and Google API Client Libraries to interact with GCP resources. These libraries require setting up the Application Default Credentials (ADC) to authenticate with GCP. See the [official documentation](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) for how to set up the ADC for your local development environment.
    Login `gcloud auth application-default login`
8. **optional** You may need to set the quota project `gcloud auth application-default set-quota-project integration-tests-484908`.
9. You can now run integration tests.

Each test has `Setup`, `Run`, and `Teardown` methods.

- `Setup` is used to create any resources needed for the test.
- `Run` is where the actual test logic is implemented.
- `Teardown` is used to clean up any resources created during the test.

The `Setup` and `Teardown` methods are idempotent, meaning they can be run multiple times without causing issues. This allows for flexibility in running tests in different orders or multiple times.

We can easily run all `Setup` tests to create resources, then run all `Run` tests to execute the actual tests, and finally run all `Teardown` tests to clean up resources.

From the `sources/gcp` directory:

For building up the infra for the Compute API resources.

```bash
go test ./integration-tests -run "TestCompute.*/Setup" -count 1
```

For running the actual tests for the Compute API resources.

```bash
go test ./integration-tests -run "TestCompute.*/Run" -count 1
```

For tearing down the infra for the Compute API resources.

```bash
go test ./integration-tests -run "TestCompute.*/Teardown" -count 1
```

> **Note:** `-count 1` is used to ensure that the tests are run and no cached results are used.

> **Note:** that the TestServiceAccountImpersonationIntegration tests do not have separate Setup, Run, and Teardown methods, as it requires state to be shared between the tests.
