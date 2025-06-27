# Running Integration Tests for GCP

Integration tests are defined in an individual file for each resource.
Test names follow the pattern `Test<API><RESOURCE>Integration`, where `<API>` is the API name and `<RESOURCE>` is the resource name.
For example, `TestComputeInstanceIntegration` tests the Compute API's Instance resource.

Integration tests are using Google Cloud Client Libraries and Google API Client Libraries to interact with GCP resources.
These libraries require setting up the Application Default Credentials (ADC) to authenticate with GCP.
See [here](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) for how to set up the ADC for local development environment.

Each test has `Setup`, `Run`, and `Teardown` methods.
- `Setup` is used to create any resources needed for the test.
- `Run` is where the actual test logic is implemented.
- `Teardown` is used to clean up any resources created during the test.

The `Setup` and `Teardown` methods are idempotent, meaning they can be run multiple times without causing issues. This allows for flexibility in running tests in different orders or multiple times.

We can easily run all `Setup` tests to create resources, then run all `Run` tests to execute the actual tests, and finally run all `Teardown` tests to clean up resources.

From the `sources/gcp/integration-tests` directory:

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