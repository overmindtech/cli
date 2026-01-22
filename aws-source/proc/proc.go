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
	"github.com/overmindtech/cli/sdpcache"
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

// wrapRegionError wraps misleading AWS errors with more helpful context
func wrapRegionError(err error, region string) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Check for OIDC-related errors which often indicate disabled opt-in regions
	if strings.Contains(errMsg, "No OpenIDConnect provider found") {
		return fmt.Errorf("%w. This error often occurs when region '%s' is not enabled in the target AWS account", err, region)
	}

	return err
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

	// Create a shared cache for all adapters in this source
	sharedCache := sdpcache.NewCache(ctx)

	var startupErrorMutex sync.Mutex
	startupError := errors.New("source is starting")
	if ec.HeartbeatOptions == nil {
		ec.HeartbeatOptions = &discovery.HeartbeatOptions{}
	}
	ec.HeartbeatOptions.HealthCheck = func(_ context.Context) error {
		startupErrorMutex.Lock()
		defer startupErrorMutex.Unlock()
		return startupError
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

			// Clear any adapters from previous retry attempts to avoid
			// duplicate registration errors
			e.ClearAdapters()
			globalDone.Store(false)

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

						// Wrap misleading OIDC errors with helpful region enablement context
						wrappedErr := wrapRegionError(err, cfg.Region)

						log.WithError(wrappedErr).WithFields(lf).Error("Error retrieving account information")
						return fmt.Errorf("error getting caller identity for region %v: %w", cfg.Region, wrappedErr)
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
						adapters.NewEC2AddressAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2CapacityReservationFleetAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2CapacityReservationAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2EgressOnlyInternetGatewayAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2IamInstanceProfileAssociationAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2ImageAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2InstanceEventWindowAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2InstanceAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2InstanceStatusAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2InternetGatewayAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2KeyPairAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2LaunchTemplateAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2LaunchTemplateVersionAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2NatGatewayAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2NetworkAclAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2NetworkInterfacePermissionAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2NetworkInterfaceAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2PlacementGroupAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2ReservedInstanceAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2RouteTableAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2SecurityGroupRuleAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2SecurityGroupAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2SnapshotAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2SubnetAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2VolumeAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2VolumeStatusAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2VpcEndpointAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2VpcPeeringConnectionAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEC2VpcAdapter(ec2Client, *callerID.Account, cfg.Region, sharedCache),

						// EFS (I'm assuming it shares its rate limit with EC2))
						adapters.NewEFSAccessPointAdapter(efsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEFSBackupPolicyAdapter(efsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEFSFileSystemAdapter(efsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEFSMountTargetAdapter(efsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEFSReplicationConfigurationAdapter(efsClient, *callerID.Account, cfg.Region, sharedCache),

						// EKS
						adapters.NewEKSAddonAdapter(eksClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEKSClusterAdapter(eksClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEKSFargateProfileAdapter(eksClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewEKSNodegroupAdapter(eksClient, *callerID.Account, cfg.Region, sharedCache),

						// Route 53
						adapters.NewRoute53HealthCheckAdapter(route53Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewRoute53HostedZoneAdapter(route53Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewRoute53ResourceRecordSetAdapter(route53Client, *callerID.Account, cfg.Region, sharedCache),

						// Cloudwatch
						adapters.NewCloudwatchAlarmAdapter(cloudwatchClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewCloudwatchInstanceMetricAdapter(cloudwatchClient, *callerID.Account, cfg.Region, sharedCache),

						// Lambda
						adapters.NewLambdaFunctionAdapter(lambdaClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewLambdaLayerAdapter(lambdaClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewLambdaLayerVersionAdapter(lambdaClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewLambdaEventSourceMappingAdapter(lambdaClient, *callerID.Account, cfg.Region, sharedCache),

						// ECS
						adapters.NewECSCapacityProviderAdapter(ecsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewECSClusterAdapter(ecsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewECSContainerInstanceAdapter(ecsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewECSServiceAdapter(ecsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewECSTaskDefinitionAdapter(ecsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewECSTaskAdapter(ecsClient, *callerID.Account, cfg.Region, sharedCache),

						// DynamoDB
						adapters.NewDynamoDBBackupAdapter(dynamodbClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDynamoDBTableAdapter(dynamodbClient, *callerID.Account, cfg.Region, sharedCache),

						// RDS
						adapters.NewRDSDBClusterParameterGroupAdapter(rdsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewRDSDBClusterAdapter(rdsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewRDSDBInstanceAdapter(rdsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewRDSDBParameterGroupAdapter(rdsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewRDSDBSubnetGroupAdapter(rdsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewRDSOptionGroupAdapter(rdsClient, *callerID.Account, cfg.Region, sharedCache),

						// AutoScaling
						adapters.NewAutoScalingGroupAdapter(autoscalingClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewAutoScalingPolicyAdapter(autoscalingClient, *callerID.Account, cfg.Region, sharedCache),

						// ELB
						adapters.NewELBInstanceHealthAdapter(elbClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewELBLoadBalancerAdapter(elbClient, *callerID.Account, cfg.Region, sharedCache),

						// ELBv2
						adapters.NewELBv2ListenerAdapter(elbv2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewELBv2LoadBalancerAdapter(elbv2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewELBv2RuleAdapter(elbv2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewELBv2TargetGroupAdapter(elbv2Client, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewELBv2TargetHealthAdapter(elbv2Client, *callerID.Account, cfg.Region, sharedCache),

						// Network Firewall
						adapters.NewNetworkFirewallFirewallAdapter(networkfirewallClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkFirewallFirewallPolicyAdapter(networkfirewallClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkFirewallRuleGroupAdapter(networkfirewallClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkFirewallTLSInspectionConfigurationAdapter(networkfirewallClient, *callerID.Account, cfg.Region, sharedCache),

						// Direct Connect
						adapters.NewDirectConnectGatewayAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDirectConnectGatewayAssociationAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDirectConnectGatewayAssociationProposalAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDirectConnectConnectionAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDirectConnectGatewayAttachmentAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDirectConnectVirtualInterfaceAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDirectConnectVirtualGatewayAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDirectConnectCustomerMetadataAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDirectConnectLagAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDirectConnectLocationAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDirectConnectHostedConnectionAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDirectConnectInterconnectAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewDirectConnectRouterConfigurationAdapter(directconnectClient, *callerID.Account, cfg.Region, sharedCache),

						// Network Manager
						adapters.NewNetworkManagerConnectAttachmentAdapter(networkmanagerClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkManagerConnectPeerAssociationAdapter(networkmanagerClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkManagerConnectPeerAdapter(networkmanagerClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkManagerCoreNetworkPolicyAdapter(networkmanagerClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkManagerCoreNetworkAdapter(networkmanagerClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkManagerNetworkResourceRelationshipsAdapter(networkmanagerClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkManagerSiteToSiteVpnAttachmentAdapter(networkmanagerClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkManagerTransitGatewayConnectPeerAssociationAdapter(networkmanagerClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkManagerTransitGatewayPeeringAdapter(networkmanagerClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkManagerTransitGatewayRegistrationAdapter(networkmanagerClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkManagerTransitGatewayRouteTableAttachmentAdapter(networkmanagerClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewNetworkManagerVPCAttachmentAdapter(networkmanagerClient, *callerID.Account, cfg.Region, sharedCache),

						// SQS
						adapters.NewSQSQueueAdapter(sqsClient, *callerID.Account, cfg.Region, sharedCache),

						// SNS
						adapters.NewSNSSubscriptionAdapter(snsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewSNSTopicAdapter(snsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewSNSPlatformApplicationAdapter(snsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewSNSEndpointAdapter(snsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewSNSDataProtectionPolicyAdapter(snsClient, *callerID.Account, cfg.Region, sharedCache),

						// KMS
						adapters.NewKMSKeyAdapter(kmsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewKMSCustomKeyStoreAdapter(kmsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewKMSAliasAdapter(kmsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewKMSGrantAdapter(kmsClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewKMSKeyPolicyAdapter(kmsClient, *callerID.Account, cfg.Region, sharedCache),

						// ApiGateway
						adapters.NewAPIGatewayRestApiAdapter(apigatewayClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewAPIGatewayResourceAdapter(apigatewayClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewAPIGatewayDomainNameAdapter(apigatewayClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewAPIGatewayMethodAdapter(apigatewayClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewAPIGatewayMethodResponseAdapter(apigatewayClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewAPIGatewayIntegrationAdapter(apigatewayClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewAPIGatewayApiKeyAdapter(apigatewayClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewAPIGatewayAuthorizerAdapter(apigatewayClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewAPIGatewayDeploymentAdapter(apigatewayClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewAPIGatewayStageAdapter(apigatewayClient, *callerID.Account, cfg.Region, sharedCache),
						adapters.NewAPIGatewayModelAdapter(apigatewayClient, *callerID.Account, cfg.Region, sharedCache),

						// SSM
						adapters.NewSSMParameterAdapter(ssmClient, *callerID.Account, cfg.Region, sharedCache),
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
						globalAdapters := []discovery.Adapter{
							// Cloudfront
							adapters.NewCloudfrontCachePolicyAdapter(cloudfrontClient, *callerID.Account, sharedCache),
							adapters.NewCloudfrontContinuousDeploymentPolicyAdapter(cloudfrontClient, *callerID.Account, sharedCache),
							adapters.NewCloudfrontDistributionAdapter(cloudfrontClient, *callerID.Account, sharedCache),
							adapters.NewCloudfrontCloudfrontFunctionAdapter(cloudfrontClient, *callerID.Account, sharedCache),
							adapters.NewCloudfrontKeyGroupAdapter(cloudfrontClient, *callerID.Account, sharedCache),
							adapters.NewCloudfrontOriginAccessControlAdapter(cloudfrontClient, *callerID.Account, sharedCache),
							adapters.NewCloudfrontOriginRequestPolicyAdapter(cloudfrontClient, *callerID.Account, sharedCache),
							adapters.NewCloudfrontResponseHeadersPolicyAdapter(cloudfrontClient, *callerID.Account, sharedCache),
							adapters.NewCloudfrontRealtimeLogConfigsAdapter(cloudfrontClient, *callerID.Account, sharedCache),
							adapters.NewCloudfrontStreamingDistributionAdapter(cloudfrontClient, *callerID.Account, sharedCache),

							// S3
							adapters.NewS3Adapter(cfg, *callerID.Account, sharedCache),

							// Networkmanager
							adapters.NewNetworkManagerGlobalNetworkAdapter(networkmanagerClient, *callerID.Account, sharedCache),
							adapters.NewNetworkManagerSiteAdapter(networkmanagerClient, *callerID.Account, sharedCache),
							adapters.NewNetworkManagerLinkAdapter(networkmanagerClient, *callerID.Account, sharedCache),
							adapters.NewNetworkManagerDeviceAdapter(networkmanagerClient, *callerID.Account, sharedCache),
							adapters.NewNetworkManagerLinkAssociationAdapter(networkmanagerClient, *callerID.Account, sharedCache),
							adapters.NewNetworkManagerConnectionAdapter(networkmanagerClient, *callerID.Account, sharedCache),

							// IAM
							adapters.NewIAMPolicyAdapter(iamClient, *callerID.Account, sharedCache),
							adapters.NewIAMGroupAdapter(iamClient, *callerID.Account, sharedCache),
							adapters.NewIAMInstanceProfileAdapter(iamClient, *callerID.Account, sharedCache),
							adapters.NewIAMRoleAdapter(iamClient, *callerID.Account, sharedCache),
							adapters.NewIAMUserAdapter(iamClient, *callerID.Account, sharedCache),
						}

						err = e.AddAdapters(globalAdapters...)
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
