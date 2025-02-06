package adapterhelpers

import "context"

const DefaultMaxResultsPerPage = 100

// These `any` types exist just for documentation

// ClientStructType represents the AWS API client that actions are run against. This is
// usually a struct that comes from the `New()` or `NewFromConfig()` functions
// in the relevant package e.g.
// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/eks@v1.26.0#Client
type ClientStructType any

// InputType is the type of data that will be sent to the a List/Describe
// function. This is typically a struct ending with the word Input such as:
// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/eks@v1.26.0#DescribeClusterInput
type InputType any

// OutputType is the type of output to expect from the List/Describe function,
// this is usually named the same as the input type, but with `Output` on the
// end e.g.
// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/eks@v1.26.0#DescribeClusterOutput
type OutputType any

// OptionsType The options struct that is passed to the client when it created,
// and also to `optFns` when getting more pages:
// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/eks@v1.26.0#ListClustersPaginator.NextPage
type OptionsType any

// AWSItemType A struct that represents the item in the AWS API e.g.
// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/route53@v1.25.2/types#HostedZone
type AWSItemType any

// Paginator Represents an AWS API Paginator:
// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#using-paginators
// The Output param should be the type of output that this specific paginator
// returns e.g. *ec2.DescribeInstancesOutput
type Paginator[Output OutputType, Options OptionsType] interface {
	HasMorePages() bool
	NextPage(context.Context, ...func(Options)) (Output, error)
}
