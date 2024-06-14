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
  <a href="https://vimeo.com/903381683">🎥 Watch a demo</a> | <a href="https://docs.overmind.tech">📖 Docs</a> | <a href="https://app.overmind.tech/api/auth/signup">🚀 Sign up</a> | <a href="https://app.overmind.tech/playground">💻 Playground</a> | <a href="https://www.linkedin.com/company/overmindtech/">🙌 Follow us</a>
</p>

# What is Overmind?

Overmind is a powerful tool for real-time impact analysis on Terraform changes. Overmind can **identify the blast radius** and **uncover potential risks** with `overmind terrafrom plan` before they harm your infrastructure, allowing anyone to make changes with confidence. We also track the impacts of the changes you make with `overmind teraform apply`, so that you can be sure that your changes haven't had any unexpected downstream impact.

# Quick Start

Install the Overmind CLI using brew:

```
brew install overmindtech/overmind/overmind-cli
```

Run a terraform plan:

```
overmind terraform plan
```

![Running 'overmind terraform plan' and viewing in the app](https://uploads-ssl.webflow.com/6241e92445c21f9c1245a940/666039f90a7a42bebcfaf692_overmind_cli_demo%20(1).gif)

<details>
<summary>Install on other platforms</summary>

## Prerequisites

- Terraform environment set up
- Access to all required credentials
- Ability to install and run the Overmind CLI

## Installation

### MacOS

To install on Mac with homebrew use:

```
brew install overmindtech/overmind/overmind-cli
```

### Windows

Install using [winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/):

```shell
winget install Overmind.OvermindCLI
```

Or manually download the [latest release](https://github.com/overmindtech/cli/releases/latest), extract `overmind.exe`, and add to your `PATH`

### Ubuntu / Debian

Set up the repository automatically:

```shell
curl -1sLf \
  'https://dl.cloudsmith.io/public/overmind/tools/setup.deb.sh' \
  | sudo -E bash
```

Or set it up manually

```shell
apt-get install -y debian-keyring  # debian only
apt-get install -y debian-archive-keyring  # debian only
apt-get install -y apt-transport-https
# For Debian Stretch, Ubuntu 16.04 and later
keyring_location=/usr/share/keyrings/overmind-tools-archive-keyring.gpg
# For Debian Jessie, Ubuntu 15.10 and earlier
keyring_location=/etc/apt/trusted.gpg.d/overmind-tools.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/overmind/tools/gpg.BC5CDEFB4E37A1B3.key' |  gpg --dearmor >> ${keyring_location}
curl -1sLf 'https://dl.cloudsmith.io/public/overmind/tools/config.deb.txt?distro=ubuntu&codename=xenial&component=main' > /etc/apt/sources.list.d/overmind-tools.list
apt-get update
```

Then install the CLI:

```shell
apt-get install overmind-cli
```

### RHEL

Set up the repository automatically:

```shell
curl -1sLf \
  'https://dl.cloudsmith.io/public/overmind/tools/setup.rpm.sh' \
  | sudo -E bash
```

Or set it up manually

```shell
yum install yum-utils pygpgme
rpm --import 'https://dl.cloudsmith.io/public/overmind/tools/gpg.BC5CDEFB4E37A1B3.key'
curl -1sLf 'https://dl.cloudsmith.io/public/overmind/tools/config.rpm.txt?distro=amzn&codename=2023' > /tmp/overmind-tools.repo
yum-config-manager --add-repo '/tmp/overmind-tools.repo'
yum -q makecache -y --disablerepo='*' --enablerepo='overmind-tools'
```

Then install the CLI:

```shell
sudo yum install overmind-cli
```

### Alpine

Set up the repository automatically:

```shell
sudo apk add --no-cache bash
curl -1sLf \
  'https://dl.cloudsmith.io/public/overmind/tools/setup.alpine.sh' \
  | sudo -E bash
```

Or set it up manually

```shell
curl -1sLf 'https://dl.cloudsmith.io/public/overmind/tools/rsa.7B6E65C2058FDB78.key' > /etc/apk/keys/tools@overmind-7B6E65C2058FDB78.rsa.pub
curl -1sLf 'https://dl.cloudsmith.io/public/overmind/tools/config.alpine.txt?distro=alpine&codename=v3.8' >> /etc/apk/repositories
apk update
```

Then install the CLI:

```shell
apk add overmind-cli
```

### Arch

Packages for Arch are available on the [releases page](https://github.com/overmindtech/cli/releases/latest) for manual download and installation.

</details>

## Why Use Overmind?

* **☁️ Cloud Complexity:** Terraform tells you what it's going to change, but not whether this change will break everything. Teams need to understand dependencies to properly understand impact.
* **👨‍🏫 Onboarding & Productivity:** Due to the reliance on "tribal knowledge", expert staff are stuck doing approvals rather than productive work and newer staff take longer to become productive.
* **📋 Change Management Process:** IaC and automation mean that changes spend substantially more time in review and approval steps than the change itself actually takes.
* **🔥 Downtime:** Outages are not caused by simple cause-and-effect relationships. More often than not, downtime is a result of dependencies people didn't know existed.

## How We Solve It?
<table style="width: 100%; table-layout: fixed;">
  <tr>
    <td style="width: 50%; vertical-align: top;">
      <img width="100%" src="https://uploads-ssl.webflow.com/6241e92445c21f9c1245a940/66607bb64e562f2d332dad8b_blast_radius.png" /><br/>
        <b>🔍 Blast Radius: </b>Overmind maps out all potential dependencies and interactions within your infra in realtime. Supports over 120 AWS resources and all Kubernetes.
    </td>
    <td style="width: 50%; vertical-align: top;">
      <img width="100%" src="https://uploads-ssl.webflow.com/6241e92445c21f9c1245a940/66607454e2bf59158c49565a_health%20check%20risk.png" /><br/>
      <b>🚨 Risks: </b>Discover specific risks that would be invisible otherwise. Risks are delivered directly to the pull request. Make deployment decisions within minutes not hours.
    </td>
  </tr>
</table>

## Advanced Use

### Passing Arguments

Overmind's `overmind terraform plan` and `overmind terraform apply` commands mostly just wrap the `terraform` that you already have installed, adding all of Overmind's features on top. This means that no matter how you're using Terraform today, this will still work with Overmind. For example if you're using a more complex command like:

```shell
terraform plan -var-file=production.tfvars -parallelism=20 -auto-approve
```

Then you would add `overmind` to the beginning, and your arguments after a double-dash e.g.

```shell
overmind terraform plan -- -var-file=production.tfvars -parallelism=20 -auto-approve
```

## Join the Community

- Join our [Discord](https://discord.com/invite/5UKsqAkPWG)
- Contact us via email at [sales@overmind.tech](mailto:sales@overmind.tech)
- Follow us on [LinkedIn](https://www.linkedin.com/company/overmindtech/)

## Additional Resources

- [Documentation](https://docs.overmind.tech)
- [Playground](https://app.overmind.tech/playground)
- [Getting Started Guide](https://docs.overmind.tech)
- [Overmind Blog](https://overmind.tech/blog)

## Reporting Bugs

- Want to report a bug or request a feature? [Open an issue](https://github.com/overmindtech/cli/issues/new)

## License

See the [LICENSE](/LICENSE) file for licensing information.

Overmind is made with ❤️ in 🇺🇸🇬🇧🇦🇹🇫🇷🇷🇴