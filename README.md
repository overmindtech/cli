# ovm-cli

CLI to interact with the overmind API

```
Usage:
  ovm-cli [command]

Available Commands:
  completion             Generate the autocompletion script for the specified shell
  get-affected-bookmarks Calculates the bookmarks that would be overlapping with a snapshot.
  get-bookmark           Displays the contents of a bookmark.
  get-snapshot           Displays the contents of a snapshot.
  help                   Help about any command
  request                Runs a request against the overmind API
  submit-plan            Creates a new Change from a given terraform plan file

Flags:
      --api-key string             The API key to use for authentication, also read from OVM_API_KEY environment variable
      --api-key-url string         The overmind API Keys endpoint (defaults to --url)
      --auth0-client-id string     OAuth Client ID to use when connecting with auth (default "j3LylZtIosVPZtouKI8WuVHmE6Lluva1")
      --auth0-domain string        Auth0 domain to connect to (default "om-prod.eu.auth0.com")
      --config string              config file (default is redacted.yaml)
      --gateway-url string         The overmind Gateway endpoint (defaults to /api/gateway on --url)
  -h, --help                       help for ovm-cli
      --honeycomb-api-key string   If specified, configures opentelemetry libraries to submit traces to honeycomb
      --json-log                   Set to true to emit logs as json for easier parsing
      --log string                 Set the log level. Valid values: panic, fatal, error, warn, info, debug, trace (default "info")
      --run-mode string            Set the run mode for this service, 'release', 'debug' or 'test'. Defaults to 'release'. (default "release")
      --sentry-dsn string          If specified, configures sentry libraries to capture errors
      --stdout-trace-dump          Dump all otel traces to stdout for debugging
      --url string                 The overmind API endpoint (default "https://api.prod.overmind.tech")
  -v, --version                    version for ovm-cli

Use "ovm-cli [command] --help" for more information about a command.
```

## Examples

Upload a terraform plan to overmind for Blast Radius Analysis:

```
terraform show -json ./tfplan > ./tfplan.json
ovm-cli submit-plan --title "example change" --plan-json ./tfplan.json
```

## Terraform ➡ Overmind Mapping

In order to calculate the blast radius from a Terraform plan, we use mappings provided by the sources to map a Terraform resource change to an Overmind item. In many cases this is simple, however in some instances, the plan doesn't have enough information for us to determine which resource the change is referring to. A good example is a Terraform environment that manages 2x Kubernetes deployments in 2x clusters which both have the same name.

By default we'll add both deployments to the blast radius since we can't tell them apart. However to improve the results, you can add the `overmind_mappings` output to your plan:

```hcl
output "overmind_mappings" {
  value = {
    # The key here should be the name of the provider. Resources that use this
    # provider will be mapped to a cluster with the below name. If you had
    # another provider with an alias such as "prod" the name would be
    # "kubernetes.prod"
    kubernetes = {
      cluster_name = var.terraform_env_name
    }
  }
}
```

Valid mapping values are:

* `cluster_name`: The name of the cluster that was provided to the kubernetes source using the `source.clusterName` option