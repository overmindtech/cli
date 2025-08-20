package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

var testFuncConfig = &types.FunctionConfiguration{
	FunctionName: adapterhelpers.PtrString("aws-controltower-NotificationForwarder"),
	FunctionArn:  adapterhelpers.PtrString("arn:aws:lambda:eu-west-2:052392120703:function:aws-controltower-NotificationForwarder"),
	Runtime:      types.RuntimePython39,
	Role:         adapterhelpers.PtrString("arn:aws:iam::052392120703:role/aws-controltower-ForwardSnsNotificationRole"), // link
	Handler:      adapterhelpers.PtrString("index.lambda_handler"),
	CodeSize:     473,
	Description:  adapterhelpers.PtrString("SNS message forwarding function for aggregating account notifications."),
	Timeout:      adapterhelpers.PtrInt32(60),
	MemorySize:   adapterhelpers.PtrInt32(128),
	LastModified: adapterhelpers.PtrString("2022-12-13T15:22:48.157+0000"),
	CodeSha256:   adapterhelpers.PtrString("3zU7iYiZektHRaog6qOFvv34ggadB56rd/UMjnYms6A="),
	Version:      adapterhelpers.PtrString("$LATEST"),
	Environment: &types.EnvironmentResponse{
		Variables: map[string]string{
			"sns_arn": "arn:aws:sns:eu-west-2:347195421325:aws-controltower-AggregateSecurityNotifications",
		},
	},
	TracingConfig: &types.TracingConfigResponse{
		Mode: types.TracingModePassThrough,
	},
	RevisionId:       adapterhelpers.PtrString("b00dd2e6-eec3-48b0-abf1-f84406e00a3e"),
	State:            types.StateActive,
	LastUpdateStatus: types.LastUpdateStatusSuccessful,
	PackageType:      types.PackageTypeZip,
	Architectures: []types.Architecture{
		types.ArchitectureX8664,
	},
	EphemeralStorage: &types.EphemeralStorage{
		Size: adapterhelpers.PtrInt32(512),
	},
	DeadLetterConfig: &types.DeadLetterConfig{
		TargetArn: adapterhelpers.PtrString("arn:aws:sns:us-east-2:444455556666:MyTopic"), // links
	},
	FileSystemConfigs: []types.FileSystemConfig{
		{
			Arn:            adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // links
			LocalMountPath: adapterhelpers.PtrString("/config"),
		},
	},
	ImageConfigResponse: &types.ImageConfigResponse{
		Error: &types.ImageConfigError{
			ErrorCode: adapterhelpers.PtrString("500"),
			Message:   adapterhelpers.PtrString("borked"),
		},
		ImageConfig: &types.ImageConfig{
			Command:          []string{"echo", "foo"},
			EntryPoint:       []string{"/bin"},
			WorkingDirectory: adapterhelpers.PtrString("/"),
		},
	},
	KMSKeyArn:                  adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
	LastUpdateStatusReason:     adapterhelpers.PtrString("reason"),
	LastUpdateStatusReasonCode: types.LastUpdateStatusReasonCodeDisabledKMSKey,
	Layers: []types.Layer{
		{
			Arn:                      adapterhelpers.PtrString("arn:aws:service:region:account:layer:name:version"), // link
			CodeSize:                 128,
			SigningJobArn:            adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
			SigningProfileVersionArn: adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
		},
	},
	MasterArn:                adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
	SigningJobArn:            adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
	SigningProfileVersionArn: adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
	SnapStart: &types.SnapStartResponse{
		ApplyOn:            types.SnapStartApplyOnPublishedVersions,
		OptimizationStatus: types.SnapStartOptimizationStatusOn,
	},
	StateReason:     adapterhelpers.PtrString("reason"),
	StateReasonCode: types.StateReasonCodeCreating,
	VpcConfig: &types.VpcConfigResponse{
		SecurityGroupIds: []string{
			"id", // link
		},
		SubnetIds: []string{
			"id", // link
		},
		VpcId: adapterhelpers.PtrString("id"), // link
	},
}

var testFuncCode = &types.FunctionCodeLocation{
	RepositoryType:   adapterhelpers.PtrString("S3"),
	Location:         adapterhelpers.PtrString("https://awslambda-eu-west-2-tasks.s3.eu-west-2.amazonaws.com/snapshots/052392120703/aws-controltower-NotificationForwarder-bcea303b-7721-4cf0-b8db-7a0e6dca76dd?versionId=3Lk06tjGEoY451GYYupIohtTV96CkVKC&X-Amz-Security-Token=IQoJb3JpZ2l&X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Etc=etcetcetc"), // link
	ImageUri:         adapterhelpers.PtrString("https://foo"),                                                                                                                                                                                                                                                                                      // link
	ResolvedImageUri: adapterhelpers.PtrString("https://foo"),                                                                                                                                                                                                                                                                                      // link
}

func (t *TestLambdaClient) GetFunction(ctx context.Context, params *lambda.GetFunctionInput, optFns ...func(*lambda.Options)) (*lambda.GetFunctionOutput, error) {
	return &lambda.GetFunctionOutput{
		Configuration: testFuncConfig,
		Code:          testFuncCode,
		Tags: map[string]string{
			"aws:cloudformation:stack-name": "StackSet-AWSControlTowerBP-BASELINE-CLOUDWATCH-6e84f2e0-f223-4b38-ac9c-d7a7ac2e8ef4",
			"aws:cloudformation:stack-id":   "arn:aws:cloudformation:eu-west-2:052392120703:stack/StackSet-AWSControlTowerBP-BASELINE-CLOUDWATCH-6e84f2e0-f223-4b38-ac9c-d7a7ac2e8ef4/f61d15a0-7af9-11ed-a39d-068d53de7052",
			"aws:cloudformation:logical-id": "ForwardSnsNotification",
		},
	}, nil
}

func (t *TestLambdaClient) ListFunctionEventInvokeConfigs(context.Context, *lambda.ListFunctionEventInvokeConfigsInput, ...func(*lambda.Options)) (*lambda.ListFunctionEventInvokeConfigsOutput, error) {
	return &lambda.ListFunctionEventInvokeConfigsOutput{
		FunctionEventInvokeConfigs: []types.FunctionEventInvokeConfig{
			{
				DestinationConfig: &types.DestinationConfig{
					OnFailure: &types.OnFailure{
						Destination: adapterhelpers.PtrString("arn:aws:events:region:account:event-bus/event-bus-name"), // link
					},
					OnSuccess: &types.OnSuccess{
						Destination: adapterhelpers.PtrString("arn:aws:events:region:account:event-bus/event-bus-name"), // link
					},
				},
				FunctionArn:              adapterhelpers.PtrString("arn:aws:service:region:account:type/id"),
				LastModified:             adapterhelpers.PtrTime(time.Now()),
				MaximumEventAgeInSeconds: adapterhelpers.PtrInt32(10),
				MaximumRetryAttempts:     adapterhelpers.PtrInt32(20),
			},
		},
	}, nil
}

func (t *TestLambdaClient) ListFunctionUrlConfigs(context.Context, *lambda.ListFunctionUrlConfigsInput, ...func(*lambda.Options)) (*lambda.ListFunctionUrlConfigsOutput, error) {
	return &lambda.ListFunctionUrlConfigsOutput{
		FunctionUrlConfigs: []types.FunctionUrlConfig{
			{
				AuthType:         types.FunctionUrlAuthTypeNone,
				CreationTime:     adapterhelpers.PtrString("recently"),
				FunctionArn:      adapterhelpers.PtrString("arn:aws:service:region:account:type/id"),
				FunctionUrl:      adapterhelpers.PtrString("https://bar"), // link
				LastModifiedTime: adapterhelpers.PtrString("recently"),
				Cors: &types.Cors{
					AllowCredentials: adapterhelpers.PtrBool(true),
					AllowHeaders:     []string{"X-Forwarded-For"},
					AllowMethods:     []string{"GET"},
					AllowOrigins:     []string{"https://bar"},
					ExposeHeaders:    []string{"X-Authentication"},
					MaxAge:           adapterhelpers.PtrInt32(10),
				},
			},
		},
	}, nil
}

func (t *TestLambdaClient) ListFunctions(context.Context, *lambda.ListFunctionsInput, ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error) {
	return &lambda.ListFunctionsOutput{
		Functions: []types.FunctionConfiguration{
			*testFuncConfig,
		},
	}, nil
}

func (t *TestLambdaClient) GetPolicy(ctx context.Context, params *lambda.GetPolicyInput, optFns ...func(*lambda.Options)) (*lambda.GetPolicyOutput, error) {
	return &lambda.GetPolicyOutput{
		Policy: &testPolicyJSON,
	}, nil
}

func TestFunctionGetFunc(t *testing.T) {
	item, err := functionGetFunc(context.Background(), &TestLambdaClient{}, "foo", &lambda.GetFunctionInput{})

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "http",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "https://awslambda-eu-west-2-tasks.s3.eu-west-2.amazonaws.com/snapshots/052392120703/aws-controltower-NotificationForwarder-bcea303b-7721-4cf0-b8db-7a0e6dca76dd?versionId=3Lk06tjGEoY451GYYupIohtTV96CkVKC",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "http",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "https://foo",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "http",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "https://foo",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "iam-role",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:iam::052392120703:role/aws-controltower-ForwardSnsNotificationRole",
			ExpectedScope:  "052392120703",
		},
		{
			ExpectedType:   "sns-topic",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:sns:us-east-2:444455556666:MyTopic",
			ExpectedScope:  "444455556666.us-east-2",
		},
		{
			ExpectedType:   "efs-access-point",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "lambda-layer-version",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "name:version",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "signer-signing-job",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "signer-signing-profile",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "lambda-function",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "signer-signing-job",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "signer-signing-profile",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "ec2-security-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-vpc",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "sns-topic",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "arn:aws:sns:eu-west-2:540044833068:example-topic",
			ExpectedScope:  "540044833068.eu-west-2",
		},
		{
			ExpectedType:   "elbv2-target-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:elasticloadbalancing:eu-west-2:540044833068:targetgroup/lambda-rvaaio9n3auuhnvvvjmp/6f23de9c63bd4653",
			ExpectedScope:  "540044833068.eu-west-2",
		},
		{
			ExpectedType:   "vpc-lattice-target-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:vpc-lattice:eu-west-2:540044833068:targetgroup/tg-0510fc8a1fef35ef0",
			ExpectedScope:  "540044833068.eu-west-2",
		},
		{
			ExpectedType:   "logs-log-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:logs:eu-west-2:540044833068:log-group:/aws/ecs/example:*",
			ExpectedScope:  "540044833068.eu-west-2",
		},
		{
			ExpectedType:   "events-rule",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:events:eu-west-2:540044833068:rule/test",
			ExpectedScope:  "540044833068.eu-west-2",
		},
		{
			ExpectedType:   "s3-bucket",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:s3:::second-example-profound-lamb",
			ExpectedScope:  "540044833068",
		},
	}

	tests.Execute(t, item)
}

func TestGetEventLinkedItem(t *testing.T) {
	type EventLinkedItemTest struct {
		ARN          string
		ExpectedType string
		ExpectError  bool
	}

	tests := []EventLinkedItemTest{
		{
			ARN:          "arn:aws:events:region:account:event-bus/event-bus-name",
			ExpectedType: "events-event-bus",
			ExpectError:  false,
		},
		{
			ARN:          "arn:aws:sqs:us-east-2:444455556666:MyQueue",
			ExpectedType: "sqs-queue",
			ExpectError:  false,
		},
		{
			ARN:          "arn:aws:sns:us-east-2:444455556666:MyTopic",
			ExpectedType: "sns-topic",
			ExpectError:  false,
		},
		{
			ARN:          "arn:aws:lambda:eu-west-2:052392120703:function:aws-controltower-NotificationForwarder",
			ExpectedType: "lambda-function",
			ExpectError:  false,
		},
		{
			ARN:         "something-bad",
			ExpectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.ARN, func(t *testing.T) {
			req, err := GetEventLinkedItem(test.ARN)

			if test.ExpectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Error(err)
				}

				if req.GetQuery().GetType() != test.ExpectedType {
					t.Errorf("expected request type to be %v, got %v", test.ExpectedType, req.GetQuery().GetType())
				}
			}
		})
	}
}

func TestNewLambdaFunctionAdapter(t *testing.T) {
	client, account, region := lambdaGetAutoConfig(t)

	adapter := NewLambdaFunctionAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
