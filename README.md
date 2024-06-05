<p align="center">
  <picture width="260px" align="center">
      <source media="(prefers-color-scheme: dark)" srcset="https://assets-global.website-files.com/6241e92445c21f9c1245a940/6582c2b96d741b023f1afabf_ov-lite-icon-p-500.png">
      <img alt="Overmind" src="https://assets-global.website-files.com/6241e92445c21f9c1245a940/6582c2b96d741b023f1afabf_ov-lite-icon-p-500.png" width="260px" align="center">
    </picture>
  <h1 align="center">Overmind CLI</h1>

<p align="center">
  <a href="https://discord.com/invite/5UKsqAkPWG" rel="nofollow"><img src="https://img.shields.io/discord/1088753599951151154?label=Discord&logo=discord&logoColor=white" alt="Discord Server"></a>
</p>

<p align="center">
  <a href="https://vimeo.com/903381683">🎥 Watch a demo</a> | <a href="https://overmind.tech/how-it-work">📖 How it works</a> | <a href="https://app.overmind.tech/api/auth/signup">🚀 Sign up</a> | <a href="https://app.overmind.tech/playground">💻 Playground</a> | <a href="https://www.linkedin.com/company/overmindtech/">🙌 Follow us</a>
</p>

## What is Overmind CLI?

![Running 'overmind terraform plan' and viewing in the app](https://uploads-ssl.webflow.com/6241e92445c21f9c1245a940/666039f90a7a42bebcfaf692_overmind_cli_demo%20(1).gif)

Overmind CLI is a powerful tool for real-time impact analysis on Terraform changes. By leveraging Overmind's capabilities, you can identify and mitigate potential risks before they harm your infrastructure, ultimately giving you the insight of a post-mortem without the associated fallout.

## Quick Install

#### Prerequisites
- Terraform environment set up
- Access to all required credentials
- Ability to install and run the Overmind CLI

#### Installation

**Mac**

```bash
brew install overmindtech/overmind/overmind-cli
```

**Windows**

1. Download the zip file for your architecture from the [Releases page](<releases-page-url>).
2. Unpack the zip file.
3. Copy the `overmind` file to your `$PATH` or use it directly with its full path.

**Linux**

1. Download the tar.gz file for your architecture from the [Releases page](<releases-page-url>).
2. Unpack with `tar xvzf FILE.tar.gz`.
3. Copy the `overmind` file to your `$PATH` or use it directly with its full path.

## Getting Started

To see the impact and potential risks of a Terraform code change you've made locally, run `overmind terraform plan` from the root of your Terraform project. This command will inspect your checkout, run `terraform plan`, discover all your existing cloud resources, and create a report of all items that could be impacted by this change. Overmind will also provide an automated assessment of deployment risks. At no point will credentials or sensitive values be uploaded to Overmind systems.

### Example Session
```sh
$ overmind terraform plan

Connected to Ovemrind
Authentication successful, using API key.
Configuring AWS Access
Choose how to access your AWS account (read-only):
> Use the default settings
  Use $AWS_PROFILE (currently: dogfood)
  Use a different profile
  Select a different AWS auth profile
  Configure managed source (opens browser)
### Detect outdated topology cache and populate if necessary
  AWS Source: running
  Stdlib Source: running
Running `terraform plan -out /tmp/overmind-plan3525309685`...
### Terraform plan output
Saved the plan to: /tmp/overmind-plan3525309685

To perform exactly these actions, run the following command to apply:
  terraform apply "/tmp/overmind-plan3525309685"
✅ Planning Changes
✅ Discover and link all resources: cache is hot
✅ Removing secrets
✅ Extracting 3 changing resources: 3 supported 0 unsupported
  aws_s3_bucket (1)
  ...
✅ Uploading planned changes (new)
✅ Calculating Blast Radius
  ✅ Discovering dependencies - done (128 items, 350 edges)
  ✅ Saving
✅ Calculating risks
  ✅ Mappning planned changes to current cloud resources
  ✅ Processing changes (3 planned changes & 0 mapped resources)
  ✅ Analyzing blast radius 
  ✅ Returing enriched risks

Check the blast radius graph while you wait:
https://app.overmind.tech/changes/02938475092387450928374059

### Potential Risks
- **Impact on Target Groups (High 🔥)**: Target groups may be indirectly affected if the security group change causes networking issues.
- **Impact on Load Balancer Traffic (Medium !)**: The restriction of egress traffic to just port 8080 could affect the distribution of traffic to backend services.
- **Misconfiguration of Egress Rules (Low ⁉)**: The security group change to port 8080 poses a risk of blocking other outbound traffic required by applications.

Check the blast radius graph and risks at:
https://app.overmind.tech/changes/1290380-28374-23498987

```

## Applying Changes

When running `overmind terraform apply`, Overmind will replicate the user experience of running `terraform apply`. It will generate a plan file but will not show this to the user. If the user specifies `-file`, Overmind will link the apply to an existing change rather than creating a new one. The yes/no decision will be made after the risks have been calculated.

For users running with `-auto-approve`, Overmind will skip the risk calculation step.

## Managed Sources

Choose how to access your AWS (read-only):
- Run locally using $AWS_PROFILE (currently: dogfood)
- Run locally using the dogfood profile
- Run locally using the prod profile
> Run managed source (opens browser)

```sh
Open: <https://app.overmind.tech/config/sources/add?type=aws>
```
To continue, select:
- I have configured a managed source
- Choose another option

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

## Join the Community

- Join our [Discord](https://discord.com/invite/5UKsqAkPWG)
- Contact us via email at [engineering@overmind.tech](mailto:engineering@overmind.tech)
- Follow us on [LinkedIn](https://www.linkedin.com/company/overmindtech/)

## Additional Resources

- [Documentation](https://docs.overmind.tech)
- [Playground](https://app.overmind.tech/playground)
- [Getting Started Guide](https://docs.overmind.tech)
- [Overmind Blog](https://overmind.tech/blog)

## Reporting Bugs, Requesting Features, or Contributing to Overmind

- Want to report a bug or request a feature? [Open an issue](<issues-url>)
- Interested in contributing to Overmind? Check out our [Contribution Guide](<contribution-guide-url>)

## License

See the [LICENSE](/LICENSE) file for licensing information.

Overmind is made with ❤️ by OvermindTech