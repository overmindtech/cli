# K8s Source Helm Chart

## Developing

Installing into a local cluster:

```shell
helm install k8s-source deployments/overmind-kube-source \
  --set source.apiKey.value=YOUR_API_KEY \
  --set source.clusterName=my-cluster
```

### Production Configuration Example

For production deployments (single replica with PDB enabled by default):

```shell
helm install k8s-source deployments/overmind-kube-source \
  --set source.apiKey.value=YOUR_API_KEY \
  --set source.clusterName=production-cluster
```

**Note**: The k8s source typically has very low load, so a single replica is usually sufficient. PDB is enabled by default to protect against maintenance operations, and the deployment uses a rolling update strategy with `maxUnavailable: 1` to ensure zero-downtime updates.

Removing the chart:

```shell
helm uninstall k8s-source
```

## Releasing

These charts are automatically released and pushed to Cloudsmith when the monorepo is tagged with a version in the following format `k8s-source/v1.2.3`. This will cause the docker container to be built, tagged with `1.2.3`, pushed, and a new corresponding helm chart released. See `.github/workflows/k8s-source-release.yml` for more details