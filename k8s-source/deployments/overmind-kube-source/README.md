# K8s Source Helm Chart

## Developing

Installing into a local cluster:

```shell
helm install k8s-source deployments/overmind-kube-source --set source.natsJWT=REPLACEME,source.natsNKeySeed=REPLACEME
```

Removing the chart:

```shell
helm uninstall k8s-source
```

## Releasing

These charts are released automatically using [helm-chart-releaser](https://github.com/marketplace/actions/helm-chart-releaser). Chart version has to match the tags in the repo. Since the action only checks for changes since the last tag, these are the right hoops to jump through:

* Edit `Chart.yaml`, updating `version` to the new version
* Commit and push to `main`
* wait for the release to happen (`overmind-kube-source-$version` tag shows up, discord notification)
* tag the same commit with `v$version` and push to github to have the corresponding docker image built
