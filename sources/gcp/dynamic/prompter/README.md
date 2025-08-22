# GCP Adapter Prompter CLI

This small CLI (prompter.go) generates a structured authoring prompt for creating a new dynamic GCP adapter.

## Run Directly
```bash
# With type definition reference
go run ./sources/gcp/dynamic/prompter \
  -name monitoring-alert-policy \
  -api-ref https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.alertPolicies/get \
  -type-ref https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.alertPolicies#AlertPolicy

# Sometimes the api reference can contain the type definition. In that case we can skip this reference.
go run ./sources/gcp/dynamic/prompter \
  -name compute-global-address \
  -api-ref https://cloud.google.com/compute/docs/reference/rest/v1/globalAddresses/get
```

## Flags
- -name (required): Adapter name (kebab-case) used for the new file and prompt wording.
- -api-ref (required): Official Google reference describing the GET (and LIST/SEARCH) endpoint.
- -type-ref (optional): Official Google reference for the resource type definition.

If required flags are missing the tool prints an error and usage info.

## Sample (truncated) Output
```
You are an expert in Google Cloud Platform (GCP).
The objective of this task is to create a new adapter for GCP monitoring-alert-policy.
...
- API Call structure: https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.alertPolicies/get
- Type Definition https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.alertPolicies#AlertPolicy
...
```

Keep the generated prompt nearby while implementing the adapter in `sources/gcp/dynamic/adapters/<name>.go`.

