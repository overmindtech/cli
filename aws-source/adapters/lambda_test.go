package adapters

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

type TestLambdaClient struct{}

func lambdaGetAutoConfig(t *testing.T) (*lambda.Client, string, string) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := lambda.NewFromConfig(config)

	return client, account, region
}

var testPolicyJSON string = `{
	"Version": "2012-10-17",
	"Id": "default",
	"Statement": [
		{
			"Sid": "lambda-191096b5-9db0-4ff2-87ce-d90c8869cb93",
			"Effect": "Allow",
			"Principal": {
				"Service": "sns.amazonaws.com"
			},
			"Action": "lambda:InvokeFunction",
			"Resource": "arn:aws:lambda:eu-west-2:540044833068:function:example_lambda_function",
			"Condition": {
				"ArnLike": {
					"AWS:SourceArn": "arn:aws:sns:eu-west-2:540044833068:example-topic"
				}
			}
		},
		{
			"Sid": "lambda-e881f390-21ed-4d5a-9e64-50ddb5562873",
			"Effect": "Allow",
			"Principal": {
				"Service": "elasticloadbalancing.amazonaws.com"
			},
			"Action": "lambda:InvokeFunction",
			"Resource": "arn:aws:lambda:eu-west-2:540044833068:function:test",
			"Condition": {
				"ArnLike": {
					"AWS:SourceArn": "arn:aws:elasticloadbalancing:eu-west-2:540044833068:targetgroup/lambda-rvaaio9n3auuhnvvvjmp/6f23de9c63bd4653"
				}
			}
		},
		{
			"Sid": "lambda-e137420e-640f-47bf-a37f-3f3c3134c110",
			"Effect": "Allow",
			"Principal": {
				"Service": "vpc-lattice.amazonaws.com"
			},
			"Action": "lambda:InvokeFunction",
			"Resource": "arn:aws:lambda:eu-west-2:540044833068:function:test",
			"Condition": {
				"ArnLike": {
					"AWS:SourceArn": "arn:aws:vpc-lattice:eu-west-2:540044833068:targetgroup/tg-0510fc8a1fef35ef0"
				}
			}
		},
		{
			"Sid": "lambda-945e8a2a-f5d2-4b32-869e-bca6227133b6",
			"Effect": "Allow",
			"Principal": {
				"Service": "logs.amazonaws.com"
			},
			"Action": "lambda:InvokeFunction",
			"Resource": "arn:aws:lambda:eu-west-2:540044833068:function:test",
			"Condition": {
				"StringEquals": {
					"AWS:SourceAccount": "540044833068"
				},
				"ArnLike": {
					"AWS:SourceArn": "arn:aws:logs:eu-west-2:540044833068:log-group:/aws/ecs/example:*"
				}
			}
		},
		{
			"Sid": "lambda-1b87395a-6f9a-406d-bc4c-4366044c1a06",
			"Effect": "Allow",
			"Principal": {
				"Service": "events.amazonaws.com"
			},
			"Action": "lambda:InvokeFunction",
			"Resource": "arn:aws:lambda:eu-west-2:540044833068:function:test",
			"Condition": {
				"ArnLike": {
					"AWS:SourceArn": "arn:aws:events:eu-west-2:540044833068:rule/test"
				}
			}
		},
		{
			"Sid": "lambda-e0070e15-19c9-4e75-8705-075d618113a4",
			"Effect": "Allow",
			"Principal": {
				"Service": "s3.amazonaws.com"
			},
			"Action": "lambda:InvokeFunction",
			"Resource": "arn:aws:lambda:eu-west-2:540044833068:function:test",
			"Condition": {
				"StringEquals": {
					"AWS:SourceAccount": "540044833068"
				},
				"ArnLike": {
					"AWS:SourceArn": "arn:aws:s3:::second-example-profound-lamb"
				}
			}
		}
	]
}`

func TestParsePolicy(t *testing.T) {
	policy := PolicyDocument{}
	err := json.Unmarshal([]byte(testPolicyJSON), &policy)

	if err != nil {
		t.Error(err)
	}

	if policy.Version != "2012-10-17" {
		t.Errorf("Expected Version to be 2012-10-17, got %s", policy.Version)
	}

	if len(policy.Statement) != 6 {
		t.Errorf("Expected 6 statements, got %d", len(policy.Statement))
	}

	if policy.Statement[0].Principal.Service != "sns.amazonaws.com" {
		t.Errorf("Expected Principal.Service to be sns.amazonaws.com, got %s", policy.Statement[0].Principal.Service)
	}

	if policy.Statement[0].Condition.ArnLike.AWSSourceArn != "arn:aws:sns:eu-west-2:540044833068:example-topic" {
		t.Errorf("Expected Condition.ArnLike.AWSSourceArn to be arn:aws:sns:eu-west-2:540044833068:example-topic, got %s", policy.Statement[0].Condition.ArnLike.AWSSourceArn)
	}

	if policy.Statement[1].Principal.Service != "elasticloadbalancing.amazonaws.com" {
		t.Errorf("Expected Principal.Service to be elasticloadbalancing.amazonaws.com, got %s", policy.Statement[1].Principal.Service)
	}

	if policy.Statement[1].Condition.ArnLike.AWSSourceArn != "arn:aws:elasticloadbalancing:eu-west-2:540044833068:targetgroup/lambda-rvaaio9n3auuhnvvvjmp/6f23de9c63bd4653" {
		t.Errorf("Expected Condition.ArnLike.AWSSourceArn to be arn:aws:elasticloadbalancing:eu-west-2:540044833068:targetgroup/lambda-rvaaio9n3auuhnvvvjmp/6f23de9c63bd4653, got %s", policy.Statement[1].Condition.ArnLike.AWSSourceArn)
	}

	if policy.Statement[2].Principal.Service != "vpc-lattice.amazonaws.com" {
		t.Errorf("Expected Principal.Service to be vpc-lattice.amazonaws.com, got %s", policy.Statement[2].Principal.Service)
	}

	if policy.Statement[2].Condition.ArnLike.AWSSourceArn != "arn:aws:vpc-lattice:eu-west-2:540044833068:targetgroup/tg-0510fc8a1fef35ef0" {
		t.Errorf("Expected Condition.ArnLike.AWSSourceArn to be arn:aws:vpc-lattice:eu-west-2:540044833068:targetgroup/tg-0510fc8a1fef35ef0, got %s", policy.Statement[2].Condition.ArnLike.AWSSourceArn)
	}

	if policy.Statement[3].Principal.Service != "logs.amazonaws.com" {
		t.Errorf("Expected Principal.Service to be logs.amazonaws.com, got %s", policy.Statement[3].Principal.Service)
	}

	if policy.Statement[3].Condition.ArnLike.AWSSourceArn != "arn:aws:logs:eu-west-2:540044833068:log-group:/aws/ecs/example:*" {
		t.Errorf("Expected Condition.ArnLike.AWSSourceArn to be arn:aws:logs:eu-west-2:540044833068:log-group:/aws/ecs/example:*, got %s", policy.Statement[3].Condition.ArnLike.AWSSourceArn)
	}

	if policy.Statement[4].Principal.Service != "events.amazonaws.com" {
		t.Errorf("Expected Principal.Service to be events.amazonaws.com, got %s", policy.Statement[4].Principal.Service)
	}

	if policy.Statement[4].Condition.ArnLike.AWSSourceArn != "arn:aws:events:eu-west-2:540044833068:rule/test" {
		t.Errorf("Expected Condition.ArnLike.AWSSourceArn to be arn:aws:events:eu-west-2:540044833068:rule/test, got %s", policy.Statement[4].Condition.ArnLike.AWSSourceArn)
	}

	if policy.Statement[5].Principal.Service != "s3.amazonaws.com" {
		t.Errorf("Expected Principal.Service to be s3.amazonaws.com, got %s", policy.Statement[5].Principal.Service)
	}

	if policy.Statement[5].Condition.ArnLike.AWSSourceArn != "arn:aws:s3:::second-example-profound-lamb" {
		t.Errorf("Expected Condition.ArnLike.AWSSourceArn to be arn:aws:s3:::second-example-profound-lamb, got %s", policy.Statement[5].Condition.ArnLike.AWSSourceArn)
	}

	if policy.Statement[5].Condition.StringEquals.AWSSourceAccount != "540044833068" {
		t.Errorf("Expected Condition.StringEquals.AWSSourceAccount to be 540044833068, got %s", policy.Statement[5].Condition.StringEquals.AWSSourceAccount)
	}
}
