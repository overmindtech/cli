package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

// LambdaClient Represents the client we need to talk to Lambda, usually this is
// *lambda.Client
type LambdaClient interface {
	GetFunction(ctx context.Context, params *lambda.GetFunctionInput, optFns ...func(*lambda.Options)) (*lambda.GetFunctionOutput, error)
	GetLayerVersion(ctx context.Context, params *lambda.GetLayerVersionInput, optFns ...func(*lambda.Options)) (*lambda.GetLayerVersionOutput, error)
	GetPolicy(ctx context.Context, params *lambda.GetPolicyInput, optFns ...func(*lambda.Options)) (*lambda.GetPolicyOutput, error)

	lambda.ListFunctionEventInvokeConfigsAPIClient
	lambda.ListFunctionUrlConfigsAPIClient
	lambda.ListFunctionsAPIClient
	lambda.ListLayerVersionsAPIClient
}

// This is derived from the AWS example:
// https://github.com/awsdocs/aws-doc-sdk-examples/blob/main/gov2/iam/actions/policies.go#L21C1-L32C2
// and represents the structure of an IAM policy document
type PolicyDocument struct {
	Version   string            `json:""`
	Statement []PolicyStatement `json:""`
}

// PolicyStatement defines a statement in a policy document.
type PolicyStatement struct {
	Action    string
	Principal Principal `json:",omitempty"`
	Condition Condition `json:",omitempty"`
}

type Principal struct {
	Service string `json:",omitempty"`
}

type Condition struct {
	ArnLike      ArnLikeCondition      `json:",omitempty"`
	StringEquals StringEqualsCondition `json:",omitempty"`
}

type StringEqualsCondition struct {
	AWSSourceAccount string `json:"AWS:SourceAccount,omitempty"`
}

type ArnLikeCondition struct {
	AWSSourceArn string `json:"AWS:SourceArn,omitempty"`
}
