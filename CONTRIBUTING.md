# Releasing a new cli version

- Make sure that main is green before proceeding! Then in the CLI directory, update to latest.

```shell
git fetch --all
git rebase origin/main
```

- Compare the changes from the last release to what is in main. For [example](https://github.com/overmindtech/cli/compare/v1.3.2...main). Following [semver](https://semver.org/) choose your new version. And use it to tag a version, and push it.

```shell
git tag -s v0.0.0
git push origin tag v0.0.0
```

- Github actions will then run, assuming everything goes green. It will create a new [pull request in overmind/homebrew-overmind](https://github.com/overmindtech/homebrew-overmind/pulls).
- **DO NOT MERGE THE PR YET** If this PR is green the PR it will require the 'pr-pull' label to be added. It will trigger another github action / check to run. This will automerge the PR and the release is complete.
