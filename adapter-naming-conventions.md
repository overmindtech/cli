# Adapter Naming Conventions Reference

This file is intended for use by agents that need to select the correct adapter type string. The
wrong name will cause a lookup to silently fail, so use the exact strings listed here.

## Naming Format per Source

| Source | Format | Example |
|---|---|---|
| **aws** | `{service}-{resource}` | `iam-role`, `ec2-instance` |
| **azure** | `azure-{api}-{resource}` | `azure-compute-virtual-machine`, `azure-authorization-role-assignment` |
| **gcp** | `gcp-{api}-{resource}` | `gcp-iam-role`, `gcp-compute-instance` |
| **stdlib** | `{name}` | `dns`, `http`, `certificate` |

AWS is the **only** source with no provider prefix. GCP and Azure always start with their provider
name. When a concept like "iam-role" exists in multiple sources, each source uses its own prefix
and sometimes a completely different term.

---

## Cross-Source: Same Concept, Different Names

This section covers concepts that exist in more than one source and where an agent must pick
the exact right name for the right source.

### IAM / Identity

| Concept | aws | azure | gcp |
|---|---|---|---|
| Role | `iam-role` | `azure-authorization-role-assignment` | `gcp-iam-role` |
| Policy | `iam-policy` | *(no equivalent)* | `gcp-iam-policy` |
| User / Account | `iam-user` | *(Entra ID — not in adapters)* | *(no user concept; use service accounts)* |
| Group | `iam-group` | *(no equivalent)* | *(no equivalent)* |
| Instance / Managed Identity | `iam-instance-profile` | `azure-managedidentity-user-assigned-identity` | `gcp-iam-service-account` |
| Service account key | *(n/a)* | *(n/a)* | `gcp-iam-service-account-key` |

**Key pitfall:** `iam-role` is a valid AWS type. The GCP equivalent is `gcp-iam-role`. Azure has
no `iam-` prefix at all — use `azure-authorization-role-assignment` instead.

### Compute: Virtual Machines

| Concept | aws | azure | gcp |
|---|---|---|---|
| VM / Instance | `ec2-instance` | `azure-compute-virtual-machine` | `gcp-compute-instance` |
| Instance status | `ec2-instance-status` | *(no equivalent)* | *(no equivalent)* |
| VM extension | *(no equivalent)* | `azure-compute-virtual-machine-extension` | *(no equivalent)* |
| VM run command | *(no equivalent)* | `azure-compute-virtual-machine-run-command` | *(no equivalent)* |
| Scale set / Instance group | *(n/a)* | `azure-compute-virtual-machine-scale-set` | `gcp-compute-instance-group`, `gcp-compute-instance-group-manager` |
| Instance template | *(n/a)* | *(n/a)* | `gcp-compute-instance-template`, `gcp-compute-regional-instance-template` |

**Key pitfall:** Azure calls it `virtual-machine`; GCP calls it `instance`. AWS prefixes with
`ec2-` rather than `compute-`.

### Compute: Disks and Snapshots

| Concept | aws | azure | gcp |
|---|---|---|---|
| Block disk / Volume | `ec2-volume` | `azure-compute-disk` | `gcp-compute-disk` |
| Volume status | `ec2-volume-status` | *(no equivalent)* | *(no equivalent)* |
| Snapshot | `ec2-snapshot` | `azure-compute-snapshot` | `gcp-compute-snapshot` |
| Instant snapshot | *(n/a)* | *(n/a)* | `gcp-compute-instant-snapshot` |
| Machine image (full VM backup) | *(n/a)* | *(n/a)* | `gcp-compute-machine-image` |

**Key pitfall:** AWS calls disks `volume` (under `ec2-`); Azure and GCP both use `disk` (under
`compute-`).

### Compute: Images

| Concept | aws | azure | gcp |
|---|---|---|---|
| Machine image / AMI | `ec2-image` | `azure-compute-image` | `gcp-compute-image` |
| Gallery image | *(n/a)* | `azure-compute-gallery-image` | *(n/a)* |
| Shared gallery image | *(n/a)* | `azure-compute-shared-gallery-image` | *(n/a)* |
| Image gallery | *(n/a)* | `azure-compute-gallery` | *(n/a)* |

### Compute: Auto Scaling

| Concept | aws | azure | gcp |
|---|---|---|---|
| Scaling group / Scale set | `autoscaling-auto-scaling-group` | `azure-compute-virtual-machine-scale-set` | `gcp-compute-instance-group-manager` |
| Scaling policy / Autoscaler | `autoscaling-auto-scaling-policy` | *(embedded in scale set)* | `gcp-compute-autoscaler` |

**Key pitfall:** The AWS name `autoscaling-auto-scaling-group` repeats "scaling" twice — this is
intentional. GCP models the fleet and policy as separate resources.

### Networking: Virtual Networks and Subnets

| Concept | aws | azure | gcp |
|---|---|---|---|
| Virtual network / VPC | `ec2-vpc` | `azure-network-virtual-network` | `gcp-compute-network` |
| VPC endpoint | `ec2-vpc-endpoint` | *(private endpoints)* | *(n/a)* |
| VPC peering | `ec2-vpc-peering-connection` | `azure-network-virtual-network-peering` | `gcp-compute-network-peering` |
| Subnet | `ec2-subnet` | `azure-network-subnet` | `gcp-compute-subnetwork` |
| Internet gateway | `ec2-internet-gateway` | *(embedded in VNet)* | *(n/a)* |
| NAT gateway | `ec2-nat-gateway` | `azure-network-nat-gateway` | *(embedded in `gcp-compute-router`)* |
| Route table | `ec2-route-table` | `azure-network-route-table` | `gcp-compute-route` |

**Key pitfall:** GCP calls it `subnetwork` (not `subnet`). Azure and GCP put networking under
different prefixes — `network-` vs `compute-`.

### Networking: IP Addresses

| Concept | aws | azure | gcp |
|---|---|---|---|
| Public/Elastic IP | `ec2-address` | `azure-network-public-ip-address` | `gcp-compute-address` |
| IP prefix | *(n/a)* | `azure-network-public-ip-prefix` | *(n/a)* |
| Global address | *(n/a)* | *(n/a)* | `gcp-compute-global-address` |

**Key pitfall:** AWS uses the opaque name `ec2-address` (Elastic IP). Azure spells it out as
`public-ip-address`. GCP uses `compute-address` (regional) vs `compute-global-address`.

### Networking: Security Groups and Firewalls

| Concept | aws | azure | gcp |
|---|---|---|---|
| Security group / NSG / Firewall | `ec2-security-group` | `azure-network-network-security-group` | `gcp-compute-firewall` |
| Security group rule | `ec2-security-group-rule` | `azure-network-security-rule` | *(embedded in firewall)* |
| Network ACL | `ec2-network-acl` | *(no equivalent)* | `gcp-compute-firewall-policy` |
| App security group | *(n/a)* | `azure-network-application-security-group` | *(n/a)* |
| Security policy (WAF) | *(n/a)* | *(n/a)* | `gcp-compute-security-policy` |

**Key pitfall:** Azure's type name `azure-network-network-security-group` has the word "network"
twice — this is correct. AWS uses `security-group`; GCP uses `firewall`.

### Networking: Load Balancers

| Concept | aws | azure | gcp |
|---|---|---|---|
| Classic load balancer | `elb-load-balancer` | *(n/a)* | *(n/a)* |
| Application / modern LB | `elbv2-load-balancer` | `azure-network-load-balancer` | `gcp-compute-forwarding-rule` |
| LB listener / rule | `elbv2-listener`, `elbv2-rule` | *(embedded)* | *(n/a)* |
| Target group | `elbv2-target-group` | *(n/a)* | `gcp-compute-backend-service` |
| App gateway (L7) | *(n/a)* | `azure-network-application-gateway` | *(n/a)* |

**Key pitfall:** GCP decomposes load balancing into `forwarding-rule` (frontend) +
`backend-service` (backend pool). There is no single `gcp-compute-load-balancer` type.

### Networking: DNS

| Concept | aws | azure | gcp |
|---|---|---|---|
| DNS zone / Hosted zone | `route53-hosted-zone` | `azure-network-zone` | `gcp-dns-managed-zone` |
| DNS record set | `route53-resource-record-set` | `azure-network-dns-record-set` | *(embedded in managed zone)* |
| Health check | `route53-health-check` | *(n/a)* | `gcp-compute-health-check` |
| Private DNS zone | *(n/a)* | `azure-network-private-dns-zone` | *(n/a)* |

**Key pitfall:** GCP uses `gcp-dns-managed-zone` (prefix `dns-`, not `network-`). Azure puts DNS
under the `network-` prefix. AWS uses `route53-` as the service prefix.

### Networking: Network Interface

| Concept | aws | azure | gcp |
|---|---|---|---|
| Network interface / NIC | `ec2-network-interface` | `azure-network-network-interface` | *(embedded in instance)* |

**Key pitfall:** Azure's type `azure-network-network-interface` has "network" twice — correct.

### Key Management

| Concept | aws | azure | gcp |
|---|---|---|---|
| KMS / Key vault (container) | *(n/a — flat hierarchy)* | `azure-keyvault-vault` | `gcp-cloud-kms-key-ring` |
| Encryption key | `kms-key` | `azure-keyvault-key` | `gcp-cloud-kms-crypto-key` |
| Key version | *(n/a)* | *(n/a)* | `gcp-cloud-kms-crypto-key-version` |
| Key alias | `kms-alias` | *(n/a)* | *(n/a)* |
| Key policy | `kms-key-policy` | *(embedded in vault)* | *(n/a)* |
| Key grant | `kms-grant` | *(n/a)* | *(n/a)* |
| Managed HSM | *(n/a)* | `azure-keyvault-managed-hsm` | *(n/a)* |

**Key pitfall:** GCP uses `cloud-kms-` prefix (not just `kms-`). The GCP key object is called
`crypto-key`, not just `key`. Azure bundles keys in a `vault`; GCP uses a `key-ring`.

### Secrets Management

| Concept | aws | azure | gcp |
|---|---|---|---|
| Secret / Parameter | `ssm-parameter` | `azure-keyvault-secret` | `gcp-secret-manager-secret` |
| Secret version | *(n/a)* | *(n/a)* | `gcp-secret-manager-secret-version` |

**Key pitfall:** AWS stores secrets as `ssm-parameter` — it's under SSM (Systems Manager), not a
dedicated secrets service. Azure stores secrets in `keyvault`. GCP has a dedicated
`secret-manager-` prefix.

### Object Storage

| Concept | aws | azure | gcp |
|---|---|---|---|
| Storage bucket / container | `s3-bucket` | `azure-storage-blob-container` | `gcp-storage-bucket` |
| Storage account | *(n/a)* | `azure-storage-account` | *(n/a)* |
| Bucket IAM policy | *(n/a)* | *(n/a)* | `gcp-storage-bucket-iam-policy` |
| File share | `efs-file-system` | `azure-storage-file-share` | `gcp-file-instance` |

**Key pitfall:** AWS uses `s3-bucket` (not `storage-bucket`). Azure separates the account
(`storage-account`) from the container (`storage-blob-container`). GCP uses `storage-bucket`.

### Relational Databases

| Concept | aws | azure | gcp |
|---|---|---|---|
| DB server / instance | `rds-db-instance` | `azure-sql-server` | `gcp-sql-admin-instance` |
| DB cluster | `rds-db-cluster` | *(n/a)* | *(n/a)* |
| Database (logical) | *(embedded)* | `azure-sql-database` | `gcp-sql-admin-database` |
| DB subnet group | `rds-db-subnet-group` | *(n/a)* | *(n/a)* |
| DB parameter group | `rds-db-parameter-group`, `rds-db-cluster-parameter-group` | *(n/a)* | *(n/a)* |
| PostgreSQL server | *(covered by rds-db-instance)* | `azure-dbforpostgresql-flexible-server` | *(covered by sql-admin-instance)* |
| PostgreSQL database | *(covered by rds-db-instance)* | `azure-dbforpostgresql-database` | *(covered by sql-admin-database)* |
| DB user | *(n/a)* | *(n/a)* | `gcp-sql-admin-user` |
| DB backup | *(n/a)* | *(n/a)* | `gcp-sql-admin-backup`, `gcp-sql-admin-backup-run` |

**Key pitfall:** AWS uses `rds-` prefix. GCP uses `sql-admin-` prefix. Azure uses `sql-` for
SQL Server and a completely different ARM provider name `dbforpostgresql-` for PostgreSQL.

### Kubernetes

| Concept | aws | azure | gcp |
|---|---|---|---|
| K8s cluster | `eks-cluster` | *(not in adapters)* | `gcp-container-cluster` |
| Node group / Node pool | `eks-nodegroup` | *(not in adapters)* | `gcp-container-node-pool` |
| Add-on | `eks-addon` | *(not in adapters)* | *(n/a)* |
| Fargate profile | `eks-fargate-profile` | *(not in adapters)* | *(n/a)* |

**Key pitfall:** AWS uses `eks-` (Elastic Kubernetes Service acronym); GCP uses `container-`
(the GKE API product name). There is no `kubernetes-` or `k8s-` prefix in either source.

### Serverless Functions

| Concept | aws | azure | gcp |
|---|---|---|---|
| Function | `lambda-function` | *(not in adapters)* | `gcp-cloud-functions-function` |
| Function layer | `lambda-layer`, `lambda-layer-version` | *(n/a)* | *(n/a)* |
| Event source | `lambda-event-source-mapping` | *(n/a)* | *(n/a)* |

**Key pitfall:** GCP's type `gcp-cloud-functions-function` contains "function" twice — this is
correct. AWS uses `lambda-function`.

### Queues, Topics and Pub/Sub

| Concept | aws | azure | gcp |
|---|---|---|---|
| Message queue | `sqs-queue` | `azure-storage-queue` | *(use pub-sub-subscription)* |
| Pub/sub topic | `sns-topic` | *(n/a)* | `gcp-pub-sub-topic` |
| Subscription | `sns-subscription` | *(n/a)* | `gcp-pub-sub-subscription` |
| Schema | *(n/a)* | *(n/a)* | `gcp-pub-sub-schema` |
| Platform app | `sns-platform-application` | *(n/a)* | *(n/a)* |

**Key pitfall:** Azure models queues as a storage feature: `azure-storage-queue`. GCP uses
`pub-sub-` as the prefix (with hyphens, not camelCase).

### Container Registry

| Concept | aws | azure | gcp |
|---|---|---|---|
| Container repository | *(n/a)* | *(n/a)* | `gcp-artifact-registry-repository` |
| Docker image | *(n/a)* | *(n/a)* | `gcp-artifact-registry-docker-image` |

### Certificates (TLS/SSL)

| Concept | aws | azure | gcp | stdlib |
|---|---|---|---|---|
| TLS certificate | *(via ACM — not in adapters)* | *(n/a)* | `gcp-certificate-manager-certificate` | `certificate` |
| SSL cert (compute) | *(n/a)* | *(n/a)* | `gcp-compute-ssl-certificate` | *(n/a)* |

---

## Complete Adapter List by Source

### aws

```
apigateway-api-key
apigateway-authorizer
apigateway-deployment
apigateway-domain-name
apigateway-integration
apigateway-method
apigateway-method-response
apigateway-model
apigateway-resource
apigateway-rest-api
apigateway-stage
autoscaling-auto-scaling-group
autoscaling-auto-scaling-policy
cloudfront-cache-policy
cloudfront-continuous-deployment-policy
cloudfront-distribution
cloudfront-function
cloudfront-key-group
cloudfront-origin-access-control
cloudfront-origin-request-policy
cloudfront-realtime-log-config
cloudfront-response-headers-policy
cloudfront-streaming-distribution
cloudwatch-alarm
cloudwatch-instance-metric
directconnect-connection
directconnect-customer-metadata
directconnect-direct-connect-gateway
directconnect-direct-connect-gateway-association
directconnect-direct-connect-gateway-association-proposal
directconnect-direct-connect-gateway-attachment
directconnect-hosted-connection
directconnect-interconnect
directconnect-lag
directconnect-location
directconnect-router-configuration
directconnect-virtual-gateway
directconnect-virtual-interface
dynamodb-backup
dynamodb-table
ec2-address
ec2-capacity-reservation
ec2-capacity-reservation-fleet
ec2-egress-only-internet-gateway
ec2-iam-instance-profile-association
ec2-image
ec2-instance
ec2-instance-event-window
ec2-instance-status
ec2-internet-gateway
ec2-key-pair
ec2-launch-template
ec2-launch-template-version
ec2-nat-gateway
ec2-network-acl
ec2-network-interface
ec2-network-interface-permission
ec2-placement-group
ec2-reserved-instance
ec2-route-table
ec2-security-group
ec2-security-group-rule
ec2-snapshot
ec2-subnet
ec2-transit-gateway-route
ec2-transit-gateway-route-table
ec2-transit-gateway-route-table-association
ec2-transit-gateway-route-table-propagation
ec2-volume
ec2-volume-status
ec2-vpc
ec2-vpc-endpoint
ec2-vpc-peering-connection
ecs-capacity-provider
ecs-cluster
ecs-container-instance
ecs-service
ecs-task
ecs-task-definition
efs-access-point
efs-backup-policy
efs-file-system
efs-mount-target
efs-replication-configuration
eks-addon
eks-cluster
eks-fargate-profile
eks-nodegroup
elb-instance-health
elb-load-balancer
elbv2-listener
elbv2-load-balancer
elbv2-rule
elbv2-target-group
elbv2-target-health
iam-group
iam-instance-profile
iam-policy
iam-role
iam-user
kms-alias
kms-custom-key-store
kms-grant
kms-key
kms-key-policy
lambda-event-source-mapping
lambda-function
lambda-layer
lambda-layer-version
network-firewall-firewall
network-firewall-firewall-policy
network-firewall-rule-group
network-firewall-tls-inspection-configuration
networkmanager-connect-attachment
networkmanager-connect-peer
networkmanager-connect-peer-association
networkmanager-connection
networkmanager-core-network
networkmanager-core-network-policy
networkmanager-device
networkmanager-global-network
networkmanager-link
networkmanager-link-association
networkmanager-network-resource-relationship
networkmanager-site
networkmanager-site-to-site-vpn-attachment
networkmanager-transit-gateway-connect-peer-association
networkmanager-transit-gateway-peering
networkmanager-transit-gateway-registration
networkmanager-transit-gateway-route-table-attachment
networkmanager-vpc-attachment
rds-db-cluster
rds-db-cluster-parameter-group
rds-db-instance
rds-db-parameter-group
rds-db-subnet-group
rds-option-group
route53-health-check
route53-hosted-zone
route53-resource-record-set
s3-bucket
sns-data-protection-policy
sns-endpoint
sns-platform-application
sns-subscription
sns-topic
sqs-queue
ssm-parameter
```

### azure

```
azure-authorization-role-assignment
azure-batch-batch-account
azure-batch-batch-application
azure-batch-batch-pool
azure-compute-availability-set
azure-compute-capacity-reservation
azure-compute-capacity-reservation-group
azure-compute-dedicated-host
azure-compute-dedicated-host-group
azure-compute-disk
azure-compute-disk-access
azure-compute-disk-access-private-endpoint-connection
azure-compute-disk-encryption-set
azure-compute-gallery
azure-compute-gallery-application
azure-compute-gallery-application-version
azure-compute-gallery-image
azure-compute-image
azure-compute-proximity-placement-group
azure-compute-shared-gallery-image
azure-compute-snapshot
azure-compute-virtual-machine
azure-compute-virtual-machine-extension
azure-compute-virtual-machine-run-command
azure-compute-virtual-machine-scale-set
azure-dbforpostgresql-database
azure-dbforpostgresql-flexible-server
azure-dbforpostgresql-flexible-server-firewall-rule
azure-dbforpostgresql-flexible-server-private-endpoint-connection
azure-documentdb-database-accounts
azure-documentdb-private-endpoint-connection
azure-elasticsan-elastic-san
azure-elasticsan-elastic-san-volume-snapshot
azure-elasticsan-volume-group
azure-keyvault-key
azure-keyvault-managed-hsm
azure-keyvault-managed-hsm-private-endpoint-connection
azure-keyvault-secret
azure-keyvault-vault
azure-managedidentity-user-assigned-identity
azure-network-application-gateway
azure-network-application-security-group
azure-network-ddos-protection-plan
azure-network-default-security-rule
azure-network-dns-record-set
azure-network-dns-virtual-network-link
azure-network-load-balancer
azure-network-nat-gateway
azure-network-network-interface
azure-network-network-security-group
azure-network-private-dns-zone
azure-network-private-endpoint
azure-network-public-ip-address
azure-network-public-ip-prefix
azure-network-route
azure-network-route-table
azure-network-security-rule
azure-network-subnet
azure-network-virtual-network
azure-network-virtual-network-gateway
azure-network-virtual-network-peering
azure-network-zone
azure-sql-database
azure-sql-elastic-pool
azure-sql-server
azure-sql-server-firewall-rule
azure-sql-server-private-endpoint-connection
azure-sql-server-virtual-network-rule
azure-storage-account
azure-storage-blob-container
azure-storage-encryption-scope
azure-storage-file-share
azure-storage-queue
azure-storage-storage-account-private-endpoint-connection
azure-storage-table
```

### gcp (manual adapters)

```
gcp-big-query-dataset
gcp-big-query-model
gcp-big-query-routine
gcp-big-query-table
gcp-certificate-manager-certificate
gcp-cloud-kms-crypto-key
gcp-cloud-kms-crypto-key-version
gcp-cloud-kms-key-ring
gcp-compute-address
gcp-compute-autoscaler
gcp-compute-backend-service
gcp-compute-disk
gcp-compute-health-check
gcp-compute-image
gcp-compute-instant-snapshot
gcp-compute-instance
gcp-compute-instance-group
gcp-compute-instance-group-manager
gcp-compute-machine-image
gcp-compute-node-group
gcp-compute-node-template
gcp-compute-regional-instance-group-manager
gcp-compute-reservation
gcp-compute-security-policy
gcp-compute-snapshot
gcp-iam-service-account
gcp-iam-service-account-key
gcp-logging-sink
gcp-storage-bucket-iam-policy
```

### gcp (dynamic adapters — auto-generated from item-types.go)

```
gcp-ai-platform-batch-prediction-job
gcp-ai-platform-custom-job
gcp-ai-platform-deployment-resource-pool
gcp-ai-platform-endpoint
gcp-ai-platform-experiment
gcp-ai-platform-experiment-run
gcp-ai-platform-model
gcp-ai-platform-model-deployment-monitoring-job
gcp-ai-platform-persistent-resource
gcp-ai-platform-pipeline-job
gcp-ai-platform-schedule
gcp-ai-platform-tensor-board
gcp-app-engine-service
gcp-artifact-registry-docker-image
gcp-artifact-registry-package
gcp-artifact-registry-package-tag
gcp-artifact-registry-package-version
gcp-artifact-registry-repository
gcp-big-query-connection
gcp-big-query-data-transfer-data-source
gcp-big-query-data-transfer-transfer-config
gcp-big-query-data-transfer-transfer-run
gcp-big-query-model
gcp-big-query-table
gcp-big-table-admin-app-profile
gcp-big-table-admin-backup
gcp-big-table-admin-cluster
gcp-big-table-admin-instance
gcp-big-table-admin-table
gcp-binary-authorization-binary-authorization-policy
gcp-certificate-manager-certificate
gcp-certificate-manager-certificate-issuance-config
gcp-certificate-manager-certificate-map
gcp-certificate-manager-certificate-map-entry
gcp-certificate-manager-dns-authorization
gcp-cloud-billing-billing-account
gcp-cloud-billing-billing-info
gcp-cloud-build-build
gcp-cloud-build-trigger
gcp-cloud-functions-function
gcp-cloud-kms-crypto-key
gcp-cloud-kms-crypto-key-version
gcp-cloud-kms-ekm-connection
gcp-cloud-kms-import-job
gcp-cloud-kms-key-ring
gcp-cloud-resource-manager-folder
gcp-cloud-resource-manager-organization
gcp-cloud-resource-manager-project
gcp-cloud-resource-manager-tag-key
gcp-cloud-resource-manager-tag-value
gcp-compute-accelerator-type
gcp-compute-address
gcp-compute-autoscaler
gcp-compute-backend-bucket
gcp-compute-backend-service
gcp-compute-bgp-route
gcp-compute-disk
gcp-compute-disk-type
gcp-compute-external-vpn-gateway
gcp-compute-firewall
gcp-compute-firewall-policy
gcp-compute-forwarding-rule
gcp-compute-gateway
gcp-compute-global-address
gcp-compute-global-forwarding-rule
gcp-compute-health-check
gcp-compute-http-health-check
gcp-compute-image
gcp-compute-instance
gcp-compute-instance-group
gcp-compute-instance-group-manager
gcp-compute-instance-settings
gcp-compute-instance-template
gcp-compute-instant-snapshot
gcp-compute-interconnect-attachment
gcp-compute-license
gcp-compute-machine-image
gcp-compute-network
gcp-compute-network-attachment
gcp-compute-network-endpoint-group
gcp-compute-network-peering
gcp-compute-node-group
gcp-compute-node-template
gcp-compute-project
gcp-compute-public-advertised-prefix
gcp-compute-public-delegated-prefix
gcp-compute-region
gcp-compute-region-commitment
gcp-compute-region-instance-group-manager
gcp-compute-regional-instance-template
gcp-compute-reservation
gcp-compute-resource-policy
gcp-compute-route
gcp-compute-route-policy
gcp-compute-router
gcp-compute-security-policy
gcp-compute-security-policy-rule
gcp-compute-service-attachment
gcp-compute-snapshot
gcp-compute-ssl-certificate
gcp-compute-ssl-policy
gcp-compute-storage-pool
gcp-compute-storage-pool-type
gcp-compute-subnetwork
gcp-compute-target-http-proxy
gcp-compute-target-https-proxy
gcp-compute-target-instance
gcp-compute-target-pool
gcp-compute-target-ssl-proxy
gcp-compute-target-tcp-proxy
gcp-compute-target-vpn-gateway
gcp-compute-url-map
gcp-compute-vpn-gateway
gcp-compute-vpn-tunnel
gcp-compute-zone
gcp-container-cluster
gcp-container-node-pool
gcp-dataform-compilation-result
gcp-dataform-repository
gcp-dataform-workflow-invocation
gcp-dataform-workspace
gcp-dataplex-aspect-type
gcp-dataplex-data-scan
gcp-dataplex-entry-group
gcp-dataplex-entity
gcp-dataproc-autoscaling-policy
gcp-dataproc-cluster
gcp-dataproc-metastore-service
gcp-dns-managed-zone
gcp-essential-contacts-contact
gcp-eventarc-channel
gcp-eventarc-trigger
gcp-file-backup
gcp-file-instance
gcp-iam-policy
gcp-iam-role
gcp-iam-service-account
gcp-iam-service-account-key
gcp-logging-bucket
gcp-logging-link
gcp-logging-saved-query
gcp-logging-sink
gcp-monitoring-alert-policy
gcp-monitoring-custom-dashboard
gcp-monitoring-notification-channel
gcp-network-connectivity-hub
gcp-network-connectivity-internal-range
gcp-network-security-client-tls-policy
gcp-network-services-mesh
gcp-network-services-service-binding
gcp-network-services-service-lb-policy
gcp-orgpolicy-policy
gcp-pub-sub-schema
gcp-pub-sub-subscription
gcp-pub-sub-topic
gcp-redis-instance
gcp-run-revision
gcp-run-service
gcp-run-worker-pool
gcp-secret-manager-secret
gcp-secret-manager-secret-version
gcp-security-center-management-effective-event-threat-detection-custom-module
gcp-security-center-management-effective-security-health-analytics-custom-module
gcp-security-center-management-event-threat-detection-custom-module
gcp-security-center-management-security-center-service
gcp-security-center-management-security-health-analytics-custom-module
gcp-service-directory-endpoint
gcp-service-directory-namespace
gcp-service-directory-service
gcp-service-usage-service
gcp-spanner-backup
gcp-spanner-backup-schedule
gcp-spanner-database
gcp-spanner-database-operation
gcp-spanner-database-role
gcp-spanner-instance
gcp-spanner-instance-config
gcp-spanner-instance-partition
gcp-spanner-session
gcp-sql-admin-backup
gcp-sql-admin-backup-run
gcp-sql-admin-database
gcp-sql-admin-instance
gcp-sql-admin-ssl-certificate
gcp-sql-admin-user
gcp-storage-bucket
gcp-storage-bucket-access-control
gcp-storage-bucket-iam-policy
gcp-storage-default-object-access-control
gcp-storage-notification-config
gcp-storage-transfer-agent-pool
gcp-storage-transfer-transfer-job
gcp-storage-transfer-transfer-operation
gcp-workflows-workflow
gcp-vpc-access-connector
```

### stdlib

```
certificate
dns
http
ip
```

*(The `rdap-*` adapters exist in the codebase but are disabled and not registered.)*

---

## Common Mistakes to Avoid

1. **Using `iam-role` for GCP** — the correct GCP type is `gcp-iam-role`.
2. **Using `gcp-iam-role` for AWS** — the correct AWS type is `iam-role` (no prefix).
3. **Using `azure-iam-role`** — Azure has no `iam-` prefix; use `azure-authorization-role-assignment`.
4. **Using `gcp-compute-load-balancer`** — this type does not exist; use `gcp-compute-forwarding-rule` + `gcp-compute-backend-service`.
5. **Using `gcp-compute-subnet`** — the correct GCP type is `gcp-compute-subnetwork`.
6. **Using `azure-network-network-security-group` and thinking the doubled "network" is a typo** — it is correct.
7. **Using `gcp-cloud-functions-function` and thinking the doubled "function" is a typo** — it is correct.
8. **Using `autoscaling-auto-scaling-group` and thinking the repeated "scaling" is a typo** — it is correct.
9. **Using `gcp-kms-key`** — the correct GCP type is `gcp-cloud-kms-crypto-key` (note `cloud-kms-` prefix and `crypto-key` noun).
10. **Using `ssm-secret` for AWS secrets** — the correct AWS type is `ssm-parameter`.
11. **Using `ec2-disk` or `ec2-block-device`** — the correct AWS type for a block volume is `ec2-volume`.
12. **Using `azure-compute-vm`** — the full type is `azure-compute-virtual-machine`.
13. **Using `gcp-container-kubernetes-cluster`** — the correct GCP type is `gcp-container-cluster`.
14. **Using `eks-node-group`** — the correct AWS type is `eks-nodegroup` (no hyphen between "node" and "group").
15. **Using `gcp-pubsub-topic` or `gcp-pub-sub-topic`** — the hyphens matter: `gcp-pub-sub-topic`.
