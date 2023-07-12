# ovm-cli

CLI to interact with the overmind API

```
Usage:
  ovm-cli [command]

Available Commands:
  change-from-tfplan     Creates a new Change from a given terraform plan file
  completion             Generate the autocompletion script for the specified shell
  get-affected-bookmarks Calculates the bookmarks that would be overlapping with a snapshot.
  get-bookmark           Displays the contents of a bookmark.
  get-snapshot           Displays the contents of a snapshot.
  help                   Help about any command
  request                Runs a request against the overmind API

Flags:
      --apikey-url string          The overmind API Keys endpoint (defaults to --url)
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
      --token string               The API token to use for authentication, also read from OVM_TOKEN environment variable
      --url string                 The overmind API endpoint (default "https://api.prod.overmind.tech")

Use "ovm-cli [command] --help" for more information about a command.
```

## Examples

Upload a terraform plan to overmind for Blast Radius Analysis:

```
terraform show -json ./tfplan > ./tfplan.json
ovm-cli change-from-tfplan --title "example change" --tfplan-json ./tfplan.json
```

