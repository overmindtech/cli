# Overmind CLI

## Installation

### MacOS

To install on Mac with homebrew use:

```
brew install overmindtech/overmind/overmind-cli
```

### Linux

Releases are available on the [releases page](https://github.com/overmindtech/cli/releases/latest)

### Windows

Releases are available on the [releases page](https://github.com/overmindtech/cli/releases/latest)

## Details

CLI to interact with the Overmind API

```
Usage:
  overmind [command]

Infrastructure as Code:
  terraform   Run Terrafrom with Overmind's change tracking - COMING SOON

Overmind API:
  bookmarks   Interact with the bookarks that were created in the Explore view
  changes     Create, update and delete changes in Overmind
  invites     Manage invites for your team to Overmind
  request     Runs a request against the overmind API
  snapshots   Create, view and delete snapshots if your infrastructure

Additional Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command

Flags:
  -h, --help         help for overmind
      --log string   Set the log level. Valid values: panic, fatal, error, warn, info, debug, trace (default "info")
  -v, --version      version for overmind

Use "overmind [command] --help" for more information about a command.
```

Set the environment variable `ACCESSIBLE` to `'true'` to enable screenreader mode.

## Examples

Upload a terraform plan to overmind for Blast Radius Analysis:

```
terraform show -json ./tfplan > ./tfplan.json
overmind changes submit-plan --title "example change" ./tfplan1.json ./tfplan2.json ./tfplan3.json
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
