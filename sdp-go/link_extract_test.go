package sdp

import (
	"log"
	"testing"

	"gopkg.in/yaml.v3"
)

// Create a very large set of attributes for the benchmark
func createTestData() (*ItemAttributes, interface{}) {
	yamlString := `---
creationTimestamp: 2024-07-09T11:16:31Z
data:
  AUTH0_AUDIENCE: https://api.example.com
  AUTH0_DOMAIN: domain.eu.auth0.com
  AUTH_COOKIE_NAME: overmind_app_access_token
  GATEWAY_CLIENT_ID: 1234567890
  GATEWAY_CORS_ALLOW_ORIGINS: https://app.example.com https://*.app.example.com
  GATEWAY_OVERMIND_AUTH_URL: https://domain.eu.auth0.com/oauth/token
  GATEWAY_OVERMIND_TOKEN_API: http://service:8080/api
  GATEWAY_PGDBNAME: user
  GATEWAY_PGHOST: name.cluster-id.eu-west-2.rds.amazonaws.com
  GATEWAY_PGPORT: "5432"
  GATEWAY_PGUSER: user
  GATEWAY_RUN_MODE: release
  GATEWAY_SERVICE_PORT: "8080"
  LOG: info
immutable: false
name: foo-config
namespace: default
resourceVersion: "167230088"
uid: c1c1be5e-e11e-46da-8ef4-ce243fe7056e
generateName: 49731160-e407-4148-bd4d-e00b8eb56cd2-5b76f5987b-
labels:
  app: test
  config-hash: 2be88ca42
  pod-template-hash: 5b76f5987b
  source: 49731160-e407-4148-bd4d-e00b8eb56cd2
spec:
  containers:
    - env:
        - name: NATS_SERVICE_HOST
          value: fdb4:5627:96ee::bfa3
        - name: NATS_SERVICE_PORT
          value: "4222"
        - name: NATS_NAME_PREFIX
          value: source.default
        - name: SERVICE_PORT
          value: "8080"
        - name: NATS_JWT
          valueFrom:
            secretKeyRef:
              key: jwt
              name: 49731160-e407-4148-bd4d-e00b8eb56cd2-nats-auth
        - name: NATS_NKEY_SEED
          valueFrom:
            secretKeyRef:
              key: nkeySeed
              name: 49731160-e407-4148-bd4d-e00b8eb56cd2-nats-auth
        - name: NATS_CA_FILE
          value: /etc/srcman/certs/ca.pem
      envFrom:
        - secretRef:
            name: prod-tracing-secrets
      image: ghcr.io/example/example:main
      imagePullPolicy: Always
      name: 49731160-e407-4148-bd4d-e00b8eb56cd2
      readinessProbe:
        failureThreshold: 3
        httpGet:
          path: healthz
          port: 8080
          scheme: HTTP
        periodSeconds: 10
        successThreshold: 1
        timeoutSeconds: 1
      resources: {}
      terminationMessagePath: /dev/termination-log
      terminationMessagePolicy: File
      volumeMounts:
        - mountPath: /etc/srcman/config
          name: source-config
          readOnly: true
        - mountPath: /etc/srcman/certs
          name: nats-certs
          readOnly: true
        - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
          name: kube-api-access-vjgp7
          readOnly: true
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeName: ip-10-0-4-118.eu-west-2.compute.internal
  preemptionPolicy: PreemptLowerPriority
  priority: 0
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 30
  tolerations:
    - effect: NoExecute
      key: node.kubernetes.io/not-ready
      operator: Exists
      tolerationSeconds: 300
    - effect: NoExecute
      key: node.kubernetes.io/unreachable
      operator: Exists
      tolerationSeconds: 300
  volumes:
    - configMap:
        defaultMode: 420
        name: 49731160-e407-4148-bd4d-e00b8eb56cd2
      name: source-config
    - configMap:
        defaultMode: 420
        name: prod-ca
      name: nats-certs
    - name: kube-api-access-vjgp7
      projected:
        defaultMode: 420
        sources:
          - serviceAccountToken:
              expirationSeconds: 3607
              path: token
          - configMap:
              items:
                - key: ca.crt
                  path: ca.crt
              name: kube-root-ca.crt
          - downwardAPI:
              items:
                - fieldRef:
                    apiVersion: v1
                    fieldPath: metadata.namespace
                  path: namespace
status:
  conditions:
    - lastTransitionTime: 2024-08-22T13:42:26Z
      status: "True"
      type: Initialized
    - lastTransitionTime: 2024-08-22T13:43:17Z
      status: "True"
      type: Ready
    - lastTransitionTime: 2024-08-22T13:43:17Z
      status: "True"
      type: ContainersReady
    - lastTransitionTime: 2024-08-22T13:42:26Z
      status: "True"
      type: PodScheduled
  containerStatuses:
    - containerID: containerd://6274579a84ea3bee8cb9bd68092f4ccd6fff13852c1e5c09672c8b3489f3c082
      image: ghcr.io/example/example:main
      imageID: ghcr.io/example/example@sha256:c3fd0767e82105e9127267bda3bdb77f51a9e6fbeb79d20c4d25ae0a71876719
      lastState: {}
      name: 49731160-e407-4148-bd4d-e00b8eb56cd2
      ready: true
      restartCount: 0
      started: true
      state:
        running:
          startedAt: 2024-08-22T13:42:32Z
  hostIP: 2a05:d01c:40:7600::6c81
  phase: Running
  podIP: 2a05:d01c:40:7600:fbac::4
  podIPs:
    - ip: 2a05:d01c:40:7600:fbac::3
  qosClass: BestEffort
  startTime: 2024-08-22T13:42:26Z
code:
  location: https://awslambda-eu-west-2-tasks.s3.eu-west-2.amazonaws.com/snapshots/123456789/ingress_log-dd9f17f1-0694-4e79-9855-300a7f9b7300?versionId=sxmueRdM4S.HXwlebzlb7rOpD2ZHmS3Z&X-Amz-Security-Token=IQoJb3JpZ2luX2VjEIn%2F%2F%2F%2F%2F%2F%2F%2F%2F%2FwEaCWV1LXdlc3QtMiJGMEQCIBWNIrhNoAqNUG%2BoZLmKNSxY9ncDogcyFTGeJFef0zVMAiBhkAW9JWxVna%2FoCXe4u9S3364dCavXEvZP%2FXcD6iwfISq7BQiR%2F%2F%2F%2F%2F%2F%2F%2F%2F%2F8BEAUaDDQ3MjAzODg2NDE4OC
  repositoryType: S3
configuration:
  architectures:
    - x86_64
  codeSha256: JxWQc4FaGuW8503fcWt5S2Ua+HHpIX2z2SMhyo/gzBU=
  codeSize: 7586073
  description: Parses LB access logs from S3, sending them to Honeycomb as structured events
  environment:
    variables:
      APIHOST: https://api.honeycomb.io
      DATASET: ingress
      ENVIRONMENT: ""
      FILTERFIELDS: ""
      FORCEGUNZIP: "true"
      HONEYCOMBWRITEKEY: foobar
      KMSKEYID: ""
      PARSERTYPE: alb
      RENAMEFIELDS: ""
      SAMPLERATE: "1"
      SAMPLERATERULES: "[]"
  ephemeralStorage:
    size: 512
  functionArn: arn:aws:lambda:eu-west-2:123456789:function:ingress_log
  functionName: ingress_log
  handler: s3-handler
  lastModified: 2024-05-10T14:33:45.279+0000
  lastUpdateStatus: Successful
  lastUpdateStatusReasonCode: ""
  loggingConfig:
    applicationLogLevel: ""
    logFormat: Text
    logGroup: /aws/lambda/ingress_log
    systemLogLevel: ""
  memorySize: 192
  packageType: Zip
  revisionId: 876d6948-2e4c-41e0-9a62-d9be8a6a59f5
  role: arn:aws:iam::123456789:role/ingress_log
  runtime: provided.al2
  runtimeVersionConfig:
    runtimeVersionArn: arn:aws:lambda:eu-west-2::runtime:f4d7a18770044f40f09a49471782a2a42431d746fcfb30bf1cadeda985858aa0
  snapStart:
    applyOn: None
    optimizationStatus: Off
  state: Active
  stateReasonCode: ""
  timeout: 600
  tracingConfig:
    mode: PassThrough
  version: $LATEST
tags:
  honeycombAgentless: "true"
  terraform: "true"
capacityProviderStrategy:
  - base: 0
    capacityProvider: FARGATE
    weight: 100
clusterArn: arn:aws:ecs:eu-west-2:123456789:cluster/example-tfc
createdAt: 2024-08-01T16:06:18.906Z
createdBy: arn:aws:iam::123456789:role/terraform-example
deploymentConfiguration:
  deploymentCircuitBreaker:
    enable: false
    rollback: false
  maximumPercent: 200
  minimumHealthyPercent: 100
deploymentController:
  type: ECS
deployments:
  - capacityProviderStrategy:
      - base: 0
        capacityProvider: FARGATE
        weight: 100
    createdAt: 2024-08-01T16:42:08.6Z
    desiredCount: 1
    failedTasks: 0
    id: ecs-svc/5699741454300708027
    launchType: ""
    networkConfiguration:
      awsvpcConfiguration:
        assignPublicIp: DISABLED
        securityGroups:
          - sg-0826c8494b61cac1f
        subnets:
          - subnet-0a393cf4c844bf32d
          - subnet-0fafe900a3dc4ba78
    pendingCount: 0
    platformFamily: Linux
    platformVersion: 1.4.0
    rolloutState: COMPLETED
    rolloutStateReason: ECS deployment ecs-svc/5699741454300708027 completed.
    runningCount: 1
    status: PRIMARY
    taskDefinition: arn:aws:ecs:eu-west-2:123456789:task-definition/facial-recognition-tfc:1
    updatedAt: 2024-08-01T17:20:11.853Z
desiredCount: 1
enableECSManagedTags: false
enableExecuteCommand: false
events:
  - createdAt: 2024-08-01T16:37:45.222Z
    id: f8240f68-73d0-497f-bf8e-4cb5185bd76c
    message: "(service facial-recognition) has started 1 tasks: (task
      d0fd4b687ebf4c968482a9814e1de455)."
  - createdAt: 2024-08-22T15:50:56.905Z
    id: 769e21aa-7a70-4270-88b9-55f902ddb727
    message: (service facial-recognition) has reached a steady state.
healthCheckGracePeriodSeconds: 0
launchType: ""
loadBalancers:
  - containerName: facial-recognition
    containerPort: 1234
    targetGroupArn: arn:aws:elasticloadbalancing:eu-west-2:123456789:targetgroup/facerec-tfc/0b6a17c7de07be40
networkConfiguration:
  awsvpcConfiguration:
    assignPublicIp: DISABLED
    securityGroups:
      - sg-0826c8494b61cac1f
    subnets:
      - subnet-0a393cf4c844bf32d
      - subnet-0fafe900a3dc4ba78
pendingCount: 0
placementConstraints: []
placementStrategy: []
platformFamily: Linux
platformVersion: LATEST
propagateTags: NONE
roleArn: arn:aws:iam::123456789:role/aws-service-role/ecs.amazonaws.com/AWSServiceRoleForECS
runningCount: 1
schedulingStrategy: REPLICA
serviceArn: arn:aws:ecs:eu-west-2:123456789:service/example-tfc/facial-recognition
serviceFullName: service/example-tfc/facial-recognition
serviceName: facial-recognition
serviceRegistries: []
taskSets: []
compatibilities:
  - EC2
  - FARGATE
containerDefinitions:
  - cpu: 1024
    environment: []
    essential: true
    healthCheck:
      command:
        - CMD-SHELL
        - wget -q --spider localhost:1234
      interval: 30
      retries: 3
      timeout: 5
    image: harshmanvar/face-detection-tensorjs:slim-amd
    memory: 2048
    mountPoints: []
    name: facial-recognition
    portMappings:
      - appProtocol: http
        containerPort: 1234
        hostPort: 1234
        protocol: tcp
    systemControls: []
    volumesFrom: []
cpu: "1024"
family: facial-recognition-tfc
ipcMode: ""
memory: "2048"
networkMode: awsvpc
pidMode: ""
registeredAt: 2024-08-01T15:27:30.781Z
registeredBy: arn:aws:sts::123456789:assumed-role/terraform-example/terraform-run-nKsGeEsVUcxfxEuY
requiresAttributes:
  - name: com.amazonaws.ecs.capability.docker-remote-api.1.18
    targetType: ""
  - name: com.amazonaws.ecs.capability.docker-remote-api.1.24
    targetType: ""
  - name: ecs.capability.container-health-check
    targetType: ""
  - name: ecs.capability.task-eni
    targetType: ""
requiresCompatibilities:
  - FARGATE
revision: 1
volumes: []
attachments:
  - details:
      - name: macAddress
        value: 0a:98:2a:a1:8c:cd
      - name: networkInterfaceId
        value: eni-0c99da7dff9025194
      - name: privateDnsName
        value: ip-10-0-2-101.eu-west-2.compute.internal
      - name: privateIPv4Address
        value: 10.0.2.101
      - name: subnetId
        value: subnet-0fafe900a3dc4ba78
    id: f2dc881c-d3c5-49ca-904e-30358f1675d8
    status: ATTACHED
    type: ElasticNetworkInterface
attributes:
  - name: ecs.cpu-architecture
    targetType: ""
    value: x86_64
availabilityZone: eu-west-2b
capacityProviderName: FARGATE
connectivity: CONNECTED
connectivityAt: 2024-08-01T17:16:34.995Z
containers:
  - containerArn: arn:aws:ecs:eu-west-2:123456789:container/example-tfc/ded4f8eebe4144ddb9a93a27b5661008/778778dd-3a31-44f0-a84f-42a2e75403d9
    cpu: "1024"
    healthStatus: HEALTHY
    image: harshmanvar/face-detection-tensorjs:slim-amd
    imageDigest: sha256:a12d885a6d05efa01735e5dd60b2580eece2f21d962e38b9bbdf8cfeb81c6894
    lastStatus: RUNNING
    memory: "2048"
    name: facial-recognition
    networkBindings: []
    networkInterfaces:
      - attachmentId: f2dc881c-d3c5-49ca-904e-30358f1675d8
    runtimeId: ded4f8eebe4144ddb9a93a27b5661008-4091029319
desiredStatus: RUNNING
ephemeralStorage:
  sizeInGiB: 20
fargateEphemeralStorage:
  sizeInGiB: 20
group: service:facial-recognition
healthStatus: HEALTHY
id: example-tfc/ded4f8eebe4144ddb9a93a27b5661008
lastStatus: RUNNING
overrides:
  containerOverrides:
    - name: facial-recognition
  inferenceAcceleratorOverrides: []
pullStartedAt: 2024-08-01T17:18:22.901Z
pullStoppedAt: 2024-08-01T17:18:34.827Z
startedAt: 2024-08-01T17:18:48.139Z
startedBy: ecs-svc/5699741454300708027
stopCode: ""
taskArn: arn:aws:ecs:eu-west-2:123456789:task/example-tfc/ded4f8eebe4144ddb9a93a27b5661008
version: 5
`

	mapData := make(map[string]interface{})
	_ = yaml.Unmarshal([]byte(yamlString), &mapData)

	attrs, _ := ToAttributes(mapData)

	return attrs, mapData
}

// Current performance:
// BenchmarkExtractLinksFromAttributes-10    	    5676	    193114 ns/op	   58868 B/op	     721 allocs/op
func BenchmarkExtractLinksFromAttributes(b *testing.B) {
	attrs, _ := createTestData()

	for range b.N {
		_ = ExtractLinksFromAttributes(attrs)
	}
}

// Current performance:
// BenchmarkExtractLinksFrom-10    	    2671	    451209 ns/op	  231509 B/op	    4241 allocs/op
func BenchmarkExtractLinksFrom(b *testing.B) {
	_, data := createTestData()

	for range b.N {
		_, _ = ExtractLinksFrom(data)
	}
}

func TestExtractLinksFromAttributes(t *testing.T) {
	attrs, _ := createTestData()

	queries := ExtractLinksFromAttributes(attrs)

	tests := []struct {
		ExpectedType  string
		ExpectedQuery string
		ExpectedScope string
	}{
		{
			ExpectedType:  "ip",
			ExpectedQuery: "2a05:d01c:40:7600::6c81",
		},
		{
			ExpectedType:  "ip",
			ExpectedQuery: "2a05:d01c:40:7600:fbac::3",
		},
		{
			ExpectedType:  "ip",
			ExpectedQuery: "2a05:d01c:40:7600:fbac::4",
		},
		{
			ExpectedType:  "ip",
			ExpectedQuery: "10.0.2.101",
		},
		{
			ExpectedType:  "ip",
			ExpectedQuery: "fdb4:5627:96ee::bfa3",
		},
		{
			ExpectedType:  "http",
			ExpectedQuery: "https://api.example.com",
		},
		{
			ExpectedType:  "http",
			ExpectedQuery: "https://domain.eu.auth0.com/oauth/token",
		},
		{
			ExpectedType:  "dns",
			ExpectedQuery: "domain.eu.auth0.com",
		},
		{
			ExpectedType:  "http",
			ExpectedQuery: "http://service:8080/api",
		},
		{
			ExpectedType:  "http",
			ExpectedQuery: "https://awslambda-eu-west-2-tasks.s3.eu-west-2.amazonaws.com/snapshots/123456789/ingress_log-dd9f17f1-0694-4e79-9855-300a7f9b7300?versionId=sxmueRdM4S.HXwlebzlb7rOpD2ZHmS3Z&X-Amz-Security-Token=IQoJb3JpZ2luX2VjEIn%2F%2F%2F%2F%2F%2F%2F%2F%2F%2FwEaCWV1LXdlc3QtMiJGMEQCIBWNIrhNoAqNUG%2BoZLmKNSxY9ncDogcyFTGeJFef0zVMAiBhkAW9JWxVna%2FoCXe4u9S3364dCavXEvZP%2FXcD6iwfISq7BQiR%2F%2F%2F%2F%2F%2F%2F%2F%2F%2F8BEAUaDDQ3MjAzODg2NDE4OC",
		},
		{
			ExpectedType:  "http",
			ExpectedQuery: "https://api.honeycomb.io",
		},
		{
			ExpectedType:  "dns",
			ExpectedQuery: "ip-10-0-2-101.eu-west-2.compute.internal",
		},
		{
			ExpectedType:  "dns",
			ExpectedQuery: "ip-10-0-4-118.eu-west-2.compute.internal",
		},
		{
			ExpectedType:  "dns",
			ExpectedQuery: "name.cluster-id.eu-west-2.rds.amazonaws.com",
		},
		{
			ExpectedType:  "lambda-function",
			ExpectedQuery: "arn:aws:lambda:eu-west-2:123456789:function:ingress_log",
			ExpectedScope: "123456789.eu-west-2",
		},
		{
			ExpectedType:  "ecs-cluster",
			ExpectedQuery: "arn:aws:ecs:eu-west-2:123456789:cluster/example-tfc",
			ExpectedScope: "123456789.eu-west-2",
		},
		{
			ExpectedType:  "ecs-container",
			ExpectedQuery: "arn:aws:ecs:eu-west-2:123456789:container/example-tfc/ded4f8eebe4144ddb9a93a27b5661008/778778dd-3a31-44f0-a84f-42a2e75403d9",
			ExpectedScope: "123456789.eu-west-2",
		},
		{
			ExpectedType:  "ecs-service",
			ExpectedQuery: "arn:aws:ecs:eu-west-2:123456789:service/example-tfc/facial-recognition",
			ExpectedScope: "123456789.eu-west-2",
		},
		{
			ExpectedType:  "ecs-task-definition",
			ExpectedQuery: "arn:aws:ecs:eu-west-2:123456789:task-definition/facial-recognition-tfc:1",
			ExpectedScope: "123456789.eu-west-2",
		},
		{
			ExpectedType:  "ecs-task",
			ExpectedQuery: "arn:aws:ecs:eu-west-2:123456789:task/example-tfc/ded4f8eebe4144ddb9a93a27b5661008",
			ExpectedScope: "123456789.eu-west-2",
		},
		{
			ExpectedType:  "elasticloadbalancing-targetgroup",
			ExpectedQuery: "arn:aws:elasticloadbalancing:eu-west-2:123456789:targetgroup/facerec-tfc/0b6a17c7de07be40",
			ExpectedScope: "123456789.eu-west-2",
		},
		{
			ExpectedType:  "iam-role",
			ExpectedQuery: "arn:aws:iam::123456789:role/aws-service-role/ecs.amazonaws.com/AWSServiceRoleForECS",
			ExpectedScope: "123456789",
		},
		{
			ExpectedType:  "iam-role",
			ExpectedQuery: "arn:aws:iam::123456789:role/ingress_log",
			ExpectedScope: "123456789",
		},
		{
			ExpectedType:  "iam-role",
			ExpectedQuery: "arn:aws:iam::123456789:role/terraform-example",
			ExpectedScope: "123456789",
		},
		{
			ExpectedType:  "sts-assumed-role",
			ExpectedQuery: "arn:aws:sts::123456789:assumed-role/terraform-example/terraform-run-nKsGeEsVUcxfxEuY",
			ExpectedScope: "123456789",
		},
	}

	if len(queries) > len(tests) {
		for i, q := range queries {
			log.Printf("%v: %v", i, q.GetQuery().GetQuery())
		}
		t.Errorf("expected %d queries, got %d", len(tests), len(queries))
	}

	for _, test := range tests {
		found := false
		for _, query := range queries {
			if query.GetQuery().GetQuery() == test.ExpectedQuery && query.GetQuery().GetType() == test.ExpectedType {
				if test.ExpectedScope == "" {
					// If we don't care about the scope then it's a match
					found = true
					break
				} else {
					// If we do care about the scope then check that it matches
					if query.GetQuery().GetScope() == test.ExpectedScope {
						found = true
						break
					}
				}
			}
		}

		if !found {
			t.Errorf("expected query not found: %s %s", test.ExpectedType, test.ExpectedQuery)
		}
	}
}

func TestExtractLinksFrom(t *testing.T) {
	tests := []struct {
		Name            string
		Object          interface{}
		ExpectedQueries []string
	}{
		{
			Name: "Env var structure array",
			Object: []struct {
				Name  string
				Value string
			}{
				{
					Name:  "example",
					Value: "https://example.com",
				},
			},
			ExpectedQueries: []string{"https://example.com"},
		},
		{
			Name:            "Just a raw string",
			Object:          "https://example.com",
			ExpectedQueries: []string{"https://example.com"},
		},
		{
			Name:            "Nil",
			Object:          nil,
			ExpectedQueries: []string{},
		},
		{
			Name: "Struct",
			Object: struct {
				Name  string
				Value string
			}{
				Name:  "example",
				Value: "https://example.com",
			},
			ExpectedQueries: []string{"https://example.com"},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			queries, err := ExtractLinksFrom(test.Object)
			if err != nil {
				t.Fatal(err)
			}

			if len(queries) != len(test.ExpectedQueries) {
				t.Errorf("expected %d queries, got %d", len(test.ExpectedQueries), len(queries))
			}

			for i, query := range queries {
				if query.GetQuery().GetQuery() != test.ExpectedQueries[i] {
					t.Errorf("expected query %s, got %s", test.ExpectedQueries[i], query.GetQuery().GetQuery())
				}
			}
		})
	}
}
