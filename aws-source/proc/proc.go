package proc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	awsapigateway "github.com/aws/aws-sdk-go-v2/service/apigateway"
	awsautoscaling "github.com/aws/aws-sdk-go-v2/service/autoscaling"
	awscloudfront "github.com/aws/aws-sdk-go-v2/service/cloudfront"
	awscloudwatch "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	awsdirectconnect "github.com/aws/aws-sdk-go-v2/service/directconnect"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"
	awsefs "github.com/aws/aws-sdk-go-v2/service/efs"
	awseks "github.com/aws/aws-sdk-go-v2/service/eks"
	awselasticloadbalancing "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	awselasticloadbalancingv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	awsiam "github.com/aws/aws-sdk-go-v2/service/iam"
	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
	awslambda "github.com/aws/aws-sdk-go-v2/service/lambda"
	awsnetworkfirewall "github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	awsnetworkmanager "github.com/aws/aws-sdk-go-v2/service/networkmanager"
	awsrds "github.com/aws/aws-sdk-go-v2/service/rds"
	awsroute53 "github.com/aws/aws-sdk-go-v2/service/route53"
	awssns "github.com/aws/aws-sdk-go-v2/service/sns"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/cenkalti/backoff/v5"
	"github.com/sourcegraph/conc/pool"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	stscredsv2 "github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/overmindtech/cli/aws-source/adapters"
	"github.com/overmindtech/cli/discovery"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// This package contains a few functions needed by the CLI to load this in-proc.
// These can not go into `/sources` because that would cause an import cycle
// with everything else.

type AwsAuthConfig struct {
	Strategy        string
	AccessKeyID     string
	SecretAccessKey string
	ExternalID      string
	TargetRoleARN   string
	Profile         string
	AutoConfig      bool

	Regions []string
}

func (c AwsAuthConfig) GetAWSConfig(region string) (aws.Config, error) {
	// Validate inputs
	if region == "" {
		return aws.Config{}, errors.New("aws-region cannot be blank")
	}

	ctx := context.Background()

	options := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithAppID("Overmind"),
	}

	if c.AutoConfig {
		if c.Strategy != "defaults" {
			log.WithField("aws-access-strategy", c.Strategy).Warn("auto-config is set to true, but aws-access-strategy is not set to 'defaults'. This may cause unexpected behaviour")
		}
		return config.LoadDefaultConfig(ctx, options...)
	}

	switch c.Strategy {
	case "defaults":
		return config.LoadDefaultConfig(ctx, options...)
	case "access-key":
		if c.AccessKeyID == "" {
			return aws.Config{}, errors.New("with access-key strategy, aws-access-key-id cannot be blank")
		}
		if c.SecretAccessKey == "" {
			return aws.Config{}, errors.New("with access-key strategy, aws-secret-access-key cannot be blank")
		}
		if c.ExternalID != "" {
			return aws.Config{}, errors.New("with access-key strategy, aws-external-id must be blank")
		}
		if c.TargetRoleARN != "" {
			return aws.Config{}, errors.New("with access-key strategy, aws-target-role-arn must be blank")
		}
		if c.Profile != "" {
			return aws.Config{}, errors.New("with access-key strategy, aws-profile must be blank")
		}

		options = append(options, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, ""),
		))

		return config.LoadDefaultConfig(ctx, options...)
	case "external-id":
		if c.AccessKeyID != "" {
			return aws.Config{}, errors.New("with external-id strategy, aws-access-key-id must be blank")
		}
		if c.SecretAccessKey != "" {
			return aws.Config{}, errors.New("with external-id strategy, aws-secret-access-key must be blank")
		}
		if c.ExternalID == "" {
			return aws.Config{}, errors.New("with external-id strategy, aws-external-id cannot be blank")
		}
		if c.TargetRoleARN == "" {
			return aws.Config{}, errors.New("with external-id strategy, aws-target-role-arn cannot be blank")
		}
		if c.Profile != "" {
			return aws.Config{}, errors.New("with external-id strategy, aws-profile must be blank")
		}

		assumeConfig, err := config.LoadDefaultConfig(ctx, options...)
		if err != nil {
			return aws.Config{}, fmt.Errorf("could not load default config from environment: %w", err)
		}

		options = append(options, config.WithCredentialsProvider(aws.NewCredentialsCache(
			stscredsv2.NewAssumeRoleProvider(
				sts.NewFromConfig(assumeConfig),
				c.TargetRoleARN,
				func(aro *stscredsv2.AssumeRoleOptions) {
					aro.ExternalID = &c.ExternalID
				},
			)),
		))

		return config.LoadDefaultConfig(ctx, options...)
	case "sso-profile":
		if c.AccessKeyID != "" {
			return aws.Config{}, errors.New("with sso-profile strategy, aws-access-key-id must be blank")
		}
		if c.SecretAccessKey != "" {
			return aws.Config{}, errors.New("with sso-profile strategy, aws-secret-access-key must be blank")
		}
		if c.ExternalID != "" {
			return aws.Config{}, errors.New("with sso-profile strategy, aws-external-id must be blank")
		}
		if c.TargetRoleARN != "" {
			return aws.Config{}, errors.New("with sso-profile strategy, aws-target-role-arn must be blank")
		}
		if c.Profile == "" {
			return aws.Config{}, errors.New("with sso-profile strategy, aws-profile cannot be blank")
		}

		options = append(options, config.WithSharedConfigProfile(c.Profile))

		return config.LoadDefaultConfig(ctx, options...)
	default:
		return aws.Config{}, errors.New("invalid aws-access-strategy")
	}
}

// Takes AwsAuthConfig options and converts these into a slice of AWS configs,
// one for each region. These can then be passed to
// `InitializeAwsSourceEngine()“ to actually start the source
func CreateAWSConfigs(awsAuthConfig AwsAuthConfig) ([]aws.Config, error) {
	if len(awsAuthConfig.Regions) == 0 {
		return nil, errors.New("no regions specified")
	}

	configs := make([]aws.Config, 0, len(awsAuthConfig.Regions))

	for _, region := range awsAuthConfig.Regions {
		region = strings.Trim(region, " ")

		cfg, err := awsAuthConfig.GetAWSConfig(region)
		if err != nil {
			return nil, fmt.Errorf("error getting AWS config for region %v: %w", region, err)
		}

		// Add OTel instrumentation
		cfg.HTTPClient = &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}

		configs = append(configs, cfg)
	}

	return configs, nil
}

// InitializeAwsSourceEngine initializes an Engine with AWS sources, returns the
// engine, and an error if any. The context provided will be used for the rate
// limit buckets and should not be cancelled until the source is shut down. AWS
// configs should be provided for each region that is enabled
func InitializeAwsSourceEngine(ctx context.Context, ec *discovery.EngineConfig, maxRetries int, configs ...aws.Config) (*discovery.Engine, error) {
	e, err := discovery.NewEngine(ec)
	if err != nil {
		return nil, fmt.Errorf("error initializing Engine: %w", err)
	}

	var startupErrorMutex sync.Mutex
	startupError := errors.New("source is starting")
	if ec.HeartbeatOptions != nil {
		ec.HeartbeatOptions.HealthCheck = func(_ context.Context) error {
			startupErrorMutex.Lock()
			defer startupErrorMutex.Unlock()
			return startupError
		}
	}

	e.StartSendingHeartbeats(ctx)
	if len(configs) == 0 {
		return nil, errors.New("No configs specified")
	}

	var globalDone atomic.Bool
	b := backoff.NewExponentialBackOff()
	b.MaxInterval = 30 * time.Second
	tick := backoff.NewTicker(b)

	try := 0

	for {
		try++
		if try > maxRetries {
			return nil, fmt.Errorf("maximum retries (%d) exceeded", maxRetries)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case _, ok := <-tick.C:
			if !ok {
				// If the backoff stops, then we should stop trying to
				// initialize and just return the error
				return nil, err
			}

			p := pool.New().WithContext(ctx)

			for _, cfg := range configs {
				p.Go(func(ctx context.Context) error {
					configCtx, configCancel := context.WithTimeout(ctx, 10*time.Second)
					defer configCancel()

					log.WithFields(log.Fields{
						"region": cfg.Region,
					}).Info("Initializing AWS source")

					// Work out what account we're using. This will be used in item scopes
					stsClient := sts.NewFromConfig(cfg)

					callerID, err := stsClient.GetCallerIdentity(configCtx, &sts.GetCallerIdentityInput{})
					if err != nil {
						lf := log.Fields{
							"region": cfg.Region,
						}
						log.WithError(err).WithFields(lf).Error("Error retrieving account information")
						return fmt.Errorf("error getting caller identity for region %v: %w", cfg.Region, err)
					}

					// Create shared clients for each API
					autoscalingClient := awsautoscaling.NewFromConfig(cfg, func(o *awsautoscaling.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					cloudfrontClient := awscloudfront.NewFromConfig(cfg, func(o *awscloudfront.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					cloudwatchClient := awscloudwatch.NewFromConfig(cfg, func(o *awscloudwatch.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					directconnectClient := awsdirectconnect.NewFromConfig(cfg, func(o *awsdirectconnect.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					dynamodbClient := awsdynamodb.NewFromConfig(cfg, func(o *awsdynamodb.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					ec2Client := awsec2.NewFromConfig(cfg, func(o *awsec2.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					ecsClient := awsecs.NewFromConfig(cfg, func(o *awsecs.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					efsClient := awsefs.NewFromConfig(cfg, func(o *awsefs.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					eksClient := awseks.NewFromConfig(cfg, func(o *awseks.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					elbClient := awselasticloadbalancing.NewFromConfig(cfg, func(o *awselasticloadbalancing.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					elbv2Client := awselasticloadbalancingv2.NewFromConfig(cfg, func(o *awselasticloadbalancingv2.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					lambdaClient := awslambda.NewFromConfig(cfg, func(o *awslambda.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					networkfirewallClient := awsnetworkfirewall.NewFromConfig(cfg, func(o *awsnetworkfirewall.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					rdsClient := awsrds.NewFromConfig(cfg, func(o *awsrds.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					snsClient := awssns.NewFromConfig(cfg, func(o *awssns.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					sqsClient := awssqs.NewFromConfig(cfg, func(o *awssqs.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					route53Client := awsroute53.NewFromConfig(cfg, func(o *awsroute53.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					networkmanagerClient := awsnetworkmanager.NewFromConfig(cfg, func(o *awsnetworkmanager.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					iamClient := awsiam.NewFromConfig(cfg, func(o *awsiam.Options) {
						o.RetryMode = aws.RetryModeAdaptive
						// Increase this from the default of 3 since IAM as such low rate limits
						o.RetryMaxAttempts = 5
					})
					kmsClient := awskms.NewFromConfig(cfg, func(o *awskms.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					apigatewayClient := awsapigateway.NewFromConfig(cfg, func(o *awsapigateway.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})
					ssmClient := ssm.NewFromConfig(cfg, func(o *ssm.Options) {
						o.RetryMode = aws.RetryModeAdaptive
					})

					configuredAdapters := []discovery.Adapter{
						// EC2
						adapters.NewEC2AddressAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2CapacityReservationFleetAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2CapacityReservationAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2EgressOnlyInternetGatewayAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2IamInstanceProfileAssociationAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2ImageAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2InstanceEventWindowAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2InstanceAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2InstanceStatusAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2InternetGatewayAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2KeyPairAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2LaunchTemplateAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2LaunchTemplateVersionAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2NatGatewayAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2NetworkAclAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2NetworkInterfacePermissionAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2NetworkInterfaceAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2PlacementGroupAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2ReservedInstanceAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2RouteTableAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2SecurityGroupRuleAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2SecurityGroupAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2SnapshotAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2SubnetAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2VolumeAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2VolumeStatusAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2VpcEndpointAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2VpcPeeringConnectionAdapter(ec2Client, *callerID.Account, cfg.Region),
						adapters.NewEC2VpcAdapter(ec2Client, *callerID.Account, cfg.Region),

						// EFS (I'm assuming it shares its rate limit with EC2))
						adapters.NewEFSAccessPointAdapter(efsClient, *callerID.Account, cfg.Region),
						adapters.NewEFSBackupPolicyAdapter(efsClient, *callerID.Account, cfg.Region),
						adapters.NewEFSFileSystemAdapter(efsClient, *callerID.Account, cfg.Region),
						adapters.NewEFSMountTargetAdapter(efsClient, *callerID.Account, cfg.Region),
						adapters.NewEFSReplicationConfigurationAdapter(efsClient, *callerID.Account, cfg.Region),

						// EKS
						adapters.NewEKSAddonAdapter(eksClient, *callerID.Account, cfg.Region),
						adapters.NewEKSClusterAdapter(eksClient, *callerID.Account, cfg.Region),
						adapters.NewEKSFargateProfileAdapter(eksClient, *callerID.Account, cfg.Region),
						adapters.NewEKSNodegroupAdapter(eksClient, *callerID.Account, cfg.Region),

						// Route 53
						adapters.NewRoute53HealthCheckAdapter(route53Client, *callerID.Account, cfg.Region),
						adapters.NewRoute53HostedZoneAdapter(route53Client, *callerID.Account, cfg.Region),
						adapters.NewRoute53ResourceRecordSetAdapter(route53Client, *callerID.Account, cfg.Region),

						// Cloudwatch
						adapters.NewCloudwatchAlarmAdapter(cloudwatchClient, *callerID.Account, cfg.Region),

						// Lambda
						adapters.NewLambdaFunctionAdapter(lambdaClient, *callerID.Account, cfg.Region),
						adapters.NewLambdaLayerAdapter(lambdaClient, *callerID.Account, cfg.Region),
						adapters.NewLambdaLayerVersionAdapter(lambdaClient, *callerID.Account, cfg.Region),

						// ECS
						adapters.NewECSCapacityProviderAdapter(ecsClient, *callerID.Account, cfg.Region),
						adapters.NewECSClusterAdapter(ecsClient, *callerID.Account, cfg.Region),
						adapters.NewECSContainerInstanceAdapter(ecsClient, *callerID.Account, cfg.Region),
						adapters.NewECSServiceAdapter(ecsClient, *callerID.Account, cfg.Region),
						adapters.NewECSTaskDefinitionAdapter(ecsClient, *callerID.Account, cfg.Region),
						adapters.NewECSTaskAdapter(ecsClient, *callerID.Account, cfg.Region),

						// DynamoDB
						adapters.NewDynamoDBBackupAdapter(dynamodbClient, *callerID.Account, cfg.Region),
						adapters.NewDynamoDBTableAdapter(dynamodbClient, *callerID.Account, cfg.Region),

						// RDS
						adapters.NewRDSDBClusterParameterGroupAdapter(rdsClient, *callerID.Account, cfg.Region),
						adapters.NewRDSDBClusterAdapter(rdsClient, *callerID.Account, cfg.Region),
						adapters.NewRDSDBInstanceAdapter(rdsClient, *callerID.Account, cfg.Region),
						adapters.NewRDSDBParameterGroupAdapter(rdsClient, *callerID.Account, cfg.Region),
						adapters.NewRDSDBSubnetGroupAdapter(rdsClient, *callerID.Account, cfg.Region),
						adapters.NewRDSOptionGroupAdapter(rdsClient, *callerID.Account, cfg.Region),

						// Autoscaling
						adapters.NewAutoScalingGroupAdapter(autoscalingClient, *callerID.Account, cfg.Region),

						// ELB
						adapters.NewELBInstanceHealthAdapter(elbClient, *callerID.Account, cfg.Region),
						adapters.NewELBLoadBalancerAdapter(elbClient, *callerID.Account, cfg.Region),

						// ELBv2
						adapters.NewELBv2ListenerAdapter(elbv2Client, *callerID.Account, cfg.Region),
						adapters.NewELBv2LoadBalancerAdapter(elbv2Client, *callerID.Account, cfg.Region),
						adapters.NewELBv2RuleAdapter(elbv2Client, *callerID.Account, cfg.Region),
						adapters.NewELBv2TargetGroupAdapter(elbv2Client, *callerID.Account, cfg.Region),
						adapters.NewELBv2TargetHealthAdapter(elbv2Client, *callerID.Account, cfg.Region),

						// Network Firewall
						adapters.NewNetworkFirewallFirewallAdapter(networkfirewallClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkFirewallFirewallPolicyAdapter(networkfirewallClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkFirewallRuleGroupAdapter(networkfirewallClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkFirewallTLSInspectionConfigurationAdapter(networkfirewallClient, *callerID.Account, cfg.Region),

						// Direct Connect
						adapters.NewDirectConnectGatewayAdapter(directconnectClient, *callerID.Account, cfg.Region),
						adapters.NewDirectConnectGatewayAssociationAdapter(directconnectClient, *callerID.Account, cfg.Region),
						adapters.NewDirectConnectGatewayAssociationProposalAdapter(directconnectClient, *callerID.Account, cfg.Region),
						adapters.NewDirectConnectConnectionAdapter(directconnectClient, *callerID.Account, cfg.Region),
						adapters.NewDirectConnectGatewayAttachmentAdapter(directconnectClient, *callerID.Account, cfg.Region),
						adapters.NewDirectConnectVirtualInterfaceAdapter(directconnectClient, *callerID.Account, cfg.Region),
						adapters.NewDirectConnectVirtualGatewayAdapter(directconnectClient, *callerID.Account, cfg.Region),
						adapters.NewDirectConnectCustomerMetadataAdapter(directconnectClient, *callerID.Account, cfg.Region),
						adapters.NewDirectConnectLagAdapter(directconnectClient, *callerID.Account, cfg.Region),
						adapters.NewDirectConnectLocationAdapter(directconnectClient, *callerID.Account, cfg.Region),
						adapters.NewDirectConnectHostedConnectionAdapter(directconnectClient, *callerID.Account, cfg.Region),
						adapters.NewDirectConnectInterconnectAdapter(directconnectClient, *callerID.Account, cfg.Region),
						adapters.NewDirectConnectRouterConfigurationAdapter(directconnectClient, *callerID.Account, cfg.Region),

						// Network Manager
						adapters.NewNetworkManagerConnectAttachmentAdapter(networkmanagerClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkManagerConnectPeerAssociationAdapter(networkmanagerClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkManagerConnectPeerAdapter(networkmanagerClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkManagerCoreNetworkPolicyAdapter(networkmanagerClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkManagerCoreNetworkAdapter(networkmanagerClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkManagerNetworkResourceRelationshipsAdapter(networkmanagerClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkManagerSiteToSiteVpnAttachmentAdapter(networkmanagerClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkManagerTransitGatewayConnectPeerAssociationAdapter(networkmanagerClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkManagerTransitGatewayPeeringAdapter(networkmanagerClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkManagerTransitGatewayRegistrationAdapter(networkmanagerClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkManagerTransitGatewayRouteTableAttachmentAdapter(networkmanagerClient, *callerID.Account, cfg.Region),
						adapters.NewNetworkManagerVPCAttachmentAdapter(networkmanagerClient, *callerID.Account, cfg.Region),

						// SQS
						adapters.NewSQSQueueAdapter(sqsClient, *callerID.Account, cfg.Region),

						// SNS
						adapters.NewSNSSubscriptionAdapter(snsClient, *callerID.Account, cfg.Region),
						adapters.NewSNSTopicAdapter(snsClient, *callerID.Account, cfg.Region),
						adapters.NewSNSPlatformApplicationAdapter(snsClient, *callerID.Account, cfg.Region),
						adapters.NewSNSEndpointAdapter(snsClient, *callerID.Account, cfg.Region),
						adapters.NewSNSDataProtectionPolicyAdapter(snsClient, *callerID.Account, cfg.Region),

						// KMS
						adapters.NewKMSKeyAdapter(kmsClient, *callerID.Account, cfg.Region),
						adapters.NewKMSCustomKeyStoreAdapter(kmsClient, *callerID.Account, cfg.Region),
						adapters.NewKMSAliasAdapter(kmsClient, *callerID.Account, cfg.Region),
						adapters.NewKMSGrantAdapter(kmsClient, *callerID.Account, cfg.Region),
						adapters.NewKMSKeyPolicyAdapter(kmsClient, *callerID.Account, cfg.Region),

						// ApiGateway
						adapters.NewAPIGatewayRestApiAdapter(apigatewayClient, *callerID.Account, cfg.Region),
						adapters.NewAPIGatewayResourceAdapter(apigatewayClient, *callerID.Account, cfg.Region),
						adapters.NewAPIGatewayDomainNameAdapter(apigatewayClient, *callerID.Account, cfg.Region),
						adapters.NewAPIGatewayMethodAdapter(apigatewayClient, *callerID.Account, cfg.Region),
						adapters.NewAPIGatewayMethodResponseAdapter(apigatewayClient, *callerID.Account, cfg.Region),
						adapters.NewAPIGatewayIntegrationAdapter(apigatewayClient, *callerID.Account, cfg.Region),
						adapters.NewAPIGatewayApiKeyAdapter(apigatewayClient, *callerID.Account, cfg.Region),
						adapters.NewAPIGatewayAuthorizerAdapter(apigatewayClient, *callerID.Account, cfg.Region),
						adapters.NewAPIGatewayDeploymentAdapter(apigatewayClient, *callerID.Account, cfg.Region),
						adapters.NewAPIGatewayStageAdapter(apigatewayClient, *callerID.Account, cfg.Region),
						adapters.NewAPIGatewayModelAdapter(apigatewayClient, *callerID.Account, cfg.Region),

						// SSM
						adapters.NewSSMParameterAdapter(ssmClient, *callerID.Account, cfg.Region),
					}

					err = e.AddAdapters(configuredAdapters...)
					if err != nil {
						return err
					}

					// Add "global" sources (those that aren't tied to a region, like
					// cloudfront). but only do this once for the first region. For
					// these APIs it doesn't matter which region we call them from, we
					// get global results
					if globalDone.CompareAndSwap(false, true) {
						err = e.AddAdapters(
							// Cloudfront
							adapters.NewCloudfrontCachePolicyAdapter(cloudfrontClient, *callerID.Account),
							adapters.NewCloudfrontContinuousDeploymentPolicyAdapter(cloudfrontClient, *callerID.Account),
							adapters.NewCloudfrontDistributionAdapter(cloudfrontClient, *callerID.Account),
							adapters.NewCloudfrontCloudfrontFunctionAdapter(cloudfrontClient, *callerID.Account),
							adapters.NewCloudfrontKeyGroupAdapter(cloudfrontClient, *callerID.Account),
							adapters.NewCloudfrontOriginAccessControlAdapter(cloudfrontClient, *callerID.Account),
							adapters.NewCloudfrontOriginRequestPolicyAdapter(cloudfrontClient, *callerID.Account),
							adapters.NewCloudfrontResponseHeadersPolicyAdapter(cloudfrontClient, *callerID.Account),
							adapters.NewCloudfrontRealtimeLogConfigsAdapter(cloudfrontClient, *callerID.Account),
							adapters.NewCloudfrontStreamingDistributionAdapter(cloudfrontClient, *callerID.Account),

							// S3
							adapters.NewS3Adapter(cfg, *callerID.Account),

							// Networkmanager
							adapters.NewNetworkManagerGlobalNetworkAdapter(networkmanagerClient, *callerID.Account),
							adapters.NewNetworkManagerSiteAdapter(networkmanagerClient, *callerID.Account),
							adapters.NewNetworkManagerLinkAdapter(networkmanagerClient, *callerID.Account),
							adapters.NewNetworkManagerDeviceAdapter(networkmanagerClient, *callerID.Account),
							adapters.NewNetworkManagerLinkAssociationAdapter(networkmanagerClient, *callerID.Account),
							adapters.NewNetworkManagerConnectionAdapter(networkmanagerClient, *callerID.Account),

							// IAM
							adapters.NewIAMPolicyAdapter(iamClient, *callerID.Account),
							adapters.NewIAMGroupAdapter(iamClient, *callerID.Account),
							adapters.NewIAMInstanceProfileAdapter(iamClient, *callerID.Account),
							adapters.NewIAMRoleAdapter(iamClient, *callerID.Account),
							adapters.NewIAMUserAdapter(iamClient, *callerID.Account),
						)
						if err != nil {
							return err
						}
					}
					return nil
				})
			}

			err = p.Wait()
			startupErrorMutex.Lock()
			startupError = err
			startupErrorMutex.Unlock()
			brokenHeart := e.SendHeartbeat(ctx, nil) // Send the error immediately through the custom health check func
			if brokenHeart != nil {
				log.WithError(brokenHeart).Error("Error sending heartbeat")
			}

			if err != nil {
				log.WithError(err).Debug("Error initializing sources")
			} else {
				log.Debug("Sources initialized")
				// If there is no error then return the engine
				return e, nil
			}
		}
	}
}
