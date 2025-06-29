package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type FunctionDetails struct {
	Code               *types.FunctionCodeLocation
	Concurrency        *types.Concurrency
	Configuration      *types.FunctionConfiguration
	UrlConfigs         []*types.FunctionUrlConfig
	EventInvokeConfigs []*types.FunctionEventInvokeConfig
	Policy             *PolicyDocument
	Tags               map[string]string
}

// FunctionGetFunc Gets the details of a specific lambda function
func functionGetFunc(ctx context.Context, client LambdaClient, scope string, input *lambda.GetFunctionInput) (*sdp.Item, error) {
	out, err := client.GetFunction(ctx, input)

	if err != nil {
		return nil, err
	}

	if out.Configuration == nil {
		return nil, errors.New("function has nil configuration")
	}

	if out.Configuration.FunctionName == nil {
		return nil, errors.New("function has empty name")
	}

	function := FunctionDetails{
		Code:          out.Code,
		Concurrency:   out.Concurrency,
		Configuration: out.Configuration,
		Tags:          out.Tags,
	}

	// Get details of all URL configs
	urlConfigs := lambda.NewListFunctionUrlConfigsPaginator(client, &lambda.ListFunctionUrlConfigsInput{
		FunctionName: out.Configuration.FunctionName,
	})

	for urlConfigs.HasMorePages() {
		urlOut, err := urlConfigs.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, config := range urlOut.FunctionUrlConfigs {
			function.UrlConfigs = append(function.UrlConfigs, &config)
		}

		err = ctx.Err()
		if err != nil {
			// If the context is done, we should stop processing and return an error, as the results are not complete
			return nil, err
		}
	}

	// Get details of event configs
	eventConfigs := lambda.NewListFunctionEventInvokeConfigsPaginator(client, &lambda.ListFunctionEventInvokeConfigsInput{
		FunctionName: out.Configuration.FunctionName,
	})

	for eventConfigs.HasMorePages() {
		eventOut, err := eventConfigs.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, event := range eventOut.FunctionEventInvokeConfigs {
			function.EventInvokeConfigs = append(function.EventInvokeConfigs, &event)
		}

		err = ctx.Err()
		if err != nil {
			// If the context is done, we should stop processing and return an error, as the results are not complete
			return nil, err
		}
	}

	// Get policies as this is often where triggers are stored
	policyResponse, err := client.GetPolicy(ctx, &lambda.GetPolicyInput{
		FunctionName: out.Configuration.FunctionName,
	})

	var linkedItemQueries []*sdp.LinkedItemQuery

	if err == nil && policyResponse != nil && policyResponse.Policy != nil {
		// Try to parse the policy
		policy := PolicyDocument{}
		err := json.Unmarshal([]byte(*policyResponse.Policy), &policy)

		if err == nil {
			linkedItemQueries = ExtractLinksFromPolicy(&policy)
		}
	}

	attributes, err := adapterhelpers.ToAttributesWithExclude(function, "resultMetadata")

	if err != nil {
		return nil, err
	}

	err = attributes.Set("Name", *out.Configuration.FunctionName)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:              "lambda-function",
		UniqueAttribute:   "Name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              out.Tags,
		LinkedItemQueries: linkedItemQueries,
	}

	if function.Code != nil {
		if function.Code.Location != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "http",
					Method: sdp.QueryMethod_GET,
					Query:  *function.Code.Location,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// These are tightly linked
					In:  true,
					Out: false,
				},
			})
		}

		if function.Code.ImageUri != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "http",
					Method: sdp.QueryMethod_GET,
					Query:  *function.Code.ImageUri,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the image will affect the function
					In: true,
					// Changing the function won't affect the image
					Out: false,
				},
			})
		}

		if function.Code.ResolvedImageUri != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "http",
					Method: sdp.QueryMethod_GET,
					Query:  *function.Code.ResolvedImageUri,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the image will affect the function
					In: true,
					// Changing the function won't affect the image
					Out: false,
				},
			})
		}
	}

	var a *adapterhelpers.ARN

	if function.Configuration != nil {
		switch function.Configuration.State {
		case types.StatePending:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.StateActive:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.StateInactive:
			item.Health = nil
		case types.StateFailed:
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		}

		if function.Configuration.Role != nil {
			if a, err = adapterhelpers.ParseARN(*function.Configuration.Role); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "iam-role",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *function.Configuration.Role,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the role will affect the function
						In: true,
						// Changing the function won't affect the role
						Out: false,
					},
				})
			}
		}

		if function.Configuration.DeadLetterConfig != nil {
			if function.Configuration.DeadLetterConfig.TargetArn != nil {
				if req, err := GetEventLinkedItem(*function.Configuration.DeadLetterConfig.TargetArn); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, req)
				}
			}
		}

		if function.Configuration.Environment != nil {
			// Automatically extract links from the environment variables
			newQueries, err := sdp.ExtractLinksFrom(function.Configuration.Environment.Variables)
			if err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, newQueries...)
			}
		}

		for _, fsConfig := range function.Configuration.FileSystemConfigs {
			if fsConfig.Arn != nil {
				if a, err = adapterhelpers.ParseARN(*fsConfig.Arn); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "efs-access-point",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *fsConfig.Arn,
							Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
						},
						BlastPropagation: &sdp.BlastPropagation{
							// These are really tightly linked
							In:  true,
							Out: true,
						},
					})
				}
			}
		}

		if function.Configuration.KMSKeyArn != nil {
			if a, err = adapterhelpers.ParseARN(*function.Configuration.KMSKeyArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "kms-key",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *function.Configuration.KMSKeyArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the key will affect the function
						In: true,
						// Changing the function won't affect the key
						Out: false,
					},
				})
			}
		}

		for _, layer := range function.Configuration.Layers {
			if layer.Arn != nil {
				if a, err = adapterhelpers.ParseARN(*layer.Arn); err == nil {
					// Strip the leading "layer:"
					name := strings.TrimPrefix(a.Resource, "layer:")

					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "lambda-layer-version",
							Method: sdp.QueryMethod_GET,
							Query:  name,
							Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
						},
						BlastPropagation: &sdp.BlastPropagation{
							// These are tightly linked
							In:  true,
							Out: true,
						},
					})
				}
			}

			if layer.SigningJobArn != nil {
				if a, err = adapterhelpers.ParseARN(*layer.SigningJobArn); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "signer-signing-job",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *layer.SigningJobArn,
							Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the signing will affect the function
							In: true,
							// Changing the function won't affect the signing
							Out: false,
						},
					})
				}
			}

			if layer.SigningProfileVersionArn != nil {
				if a, err = adapterhelpers.ParseARN(*layer.SigningProfileVersionArn); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "signer-signing-profile",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *layer.SigningProfileVersionArn,
							Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the signing will affect the function
							In: true,
							// Changing the function won't affect the signing
							Out: false,
						},
					})
				}
			}
		}

		if function.Configuration.MasterArn != nil {
			if a, err = adapterhelpers.ParseARN(*function.Configuration.MasterArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "lambda-function",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *function.Configuration.MasterArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Tightly linked
						In:  true,
						Out: true,
					},
				})
			}
		}

		if function.Configuration.SigningJobArn != nil {
			if a, err = adapterhelpers.ParseARN(*function.Configuration.SigningJobArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "signer-signing-job",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *function.Configuration.SigningJobArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the signing will affect the function
						In: true,
						// Changing the function won't affect the signing
						Out: false,
					},
				})
			}
		}

		if function.Configuration.SigningProfileVersionArn != nil {
			if a, err = adapterhelpers.ParseARN(*function.Configuration.SigningProfileVersionArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "signer-signing-profile",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *function.Configuration.SigningProfileVersionArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the signing will affect the function
						In: true,
						// Changing the function won't affect the signing
						Out: false,
					},
				})
			}
		}

		if function.Configuration.VpcConfig != nil {
			for _, id := range function.Configuration.VpcConfig.SecurityGroupIds {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-security-group",
						Method: sdp.QueryMethod_GET,
						Query:  id,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the security group will affect the function
						In: true,
						// Changing the function won't affect the security group
						Out: false,
					},
				})
			}

			for _, id := range function.Configuration.VpcConfig.SubnetIds {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-subnet",
						Method: sdp.QueryMethod_GET,
						Query:  id,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the subnet will affect the function
						In: true,
						// Changing the function won't affect the subnet
						Out: false,
					},
				})
			}

			if function.Configuration.VpcConfig.VpcId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-vpc",
						Method: sdp.QueryMethod_GET,
						Query:  *function.Configuration.VpcConfig.VpcId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{},
				})
			}
		}
	}

	for _, config := range function.UrlConfigs {
		if config.FunctionUrl != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "http",
					Method: sdp.QueryMethod_GET,
					Query:  *config.FunctionUrl,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// These are tightly linked
					In:  true,
					Out: true,
				},
			})
		}
	}

	for _, config := range function.EventInvokeConfigs {
		if config.DestinationConfig != nil {
			if config.DestinationConfig.OnFailure != nil {
				if config.DestinationConfig.OnFailure.Destination != nil {
					// Possible links from `GetEventLinkedItem()`

					lir, err := GetEventLinkedItem(*config.DestinationConfig.OnFailure.Destination)

					if err == nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, lir)
					}
				}
			}

			if config.DestinationConfig.OnSuccess != nil {
				if config.DestinationConfig.OnSuccess.Destination != nil {
					lir, err := GetEventLinkedItem(*config.DestinationConfig.OnSuccess.Destination)

					if err == nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, lir)
					}

				}
			}
		}
	}

	return &item, nil
}

func ExtractLinksFromPolicy(policy *PolicyDocument) []*sdp.LinkedItemQuery {
	links := make([]*sdp.LinkedItemQuery, 0)

	for _, statement := range policy.Statement {
		var queryType string
		var scope string
		method := sdp.QueryMethod_SEARCH

		switch statement.Principal.Service {
		case "sns.amazonaws.com":
			queryType = "sns-topic"
			method = sdp.QueryMethod_GET
		case "elasticloadbalancing.amazonaws.com":
			queryType = "elbv2-target-group"
		case "vpc-lattice.amazonaws.com":
			queryType = "vpc-lattice-target-group"
		case "logs.amazonaws.com":
			queryType = "logs-log-group"
		case "events.amazonaws.com":
			queryType = "events-rule"
		case "s3.amazonaws.com":
			// S3 is global and runs in an account scope so we need to extract
			// that from the policy as the ARN doesn't contain the account that
			// the bucket is in
			queryType = "s3-bucket"
			scope = adapterhelpers.FormatScope(statement.Condition.StringEquals.AWSSourceAccount, "")
		default:
			continue
		}

		if scope == "" {
			// If we don't have a scope set then extract it from the target ARN
			parsedARN, err := adapterhelpers.ParseARN(statement.Condition.ArnLike.AWSSourceArn)

			if err != nil {
				continue
			}

			scope = adapterhelpers.FormatScope(parsedARN.AccountID, parsedARN.Region)
		}

		links = append(links, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   queryType,
				Method: method,
				Query:  statement.Condition.ArnLike.AWSSourceArn,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changing a lambda shouldn't affect the upstream source
				Out: false,
				// Changing the source should affect the lambda
				In: true,
			},
		})
	}

	return links
}

// GetEventLinkedItem Gets the linked item request for a given destination ARN
func GetEventLinkedItem(destinationARN string) (*sdp.LinkedItemQuery, error) {
	parsed, err := adapterhelpers.ParseARN(destinationARN)

	if err != nil {
		return nil, err
	}

	scope := adapterhelpers.FormatScope(parsed.AccountID, parsed.Region)

	switch parsed.Service {
	case "sns":
		// In this case it's an SNS topic
		return &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "sns-topic",
				Method: sdp.QueryMethod_SEARCH,
				Query:  destinationARN,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// These are tightly linked
				In:  true,
				Out: true,
			},
		}, nil
	case "sqs":
		return &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "sqs-queue",
				Method: sdp.QueryMethod_SEARCH,
				Query:  destinationARN,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// These are tightly linked
				In:  true,
				Out: true,
			},
		}, nil
	case "lambda":
		return &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "lambda-function",
				Method: sdp.QueryMethod_SEARCH,
				Query:  destinationARN,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// These are tightly linked
				In:  true,
				Out: true,
			},
		}, nil
	case "events":
		return &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "events-event-bus",
				Method: sdp.QueryMethod_SEARCH,
				Query:  destinationARN,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// These are tightly linked
				In:  true,
				Out: true,
			},
		}, nil
	}

	return nil, errors.New("could not find matching request")
}

func NewLambdaFunctionAdapter(client LambdaClient, accountID string, region string) *adapterhelpers.AlwaysGetAdapter[*lambda.ListFunctionsInput, *lambda.ListFunctionsOutput, *lambda.GetFunctionInput, *lambda.GetFunctionOutput, LambdaClient, *lambda.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*lambda.ListFunctionsInput, *lambda.ListFunctionsOutput, *lambda.GetFunctionInput, *lambda.GetFunctionOutput, LambdaClient, *lambda.Options]{
		ItemType:        "lambda-function",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		ListInput:       &lambda.ListFunctionsInput{},
		GetFunc:         functionGetFunc,
		AdapterMetadata: lambdaFunctionAdapterMetadata,
		GetInputMapper: func(scope, query string) *lambda.GetFunctionInput {
			return &lambda.GetFunctionInput{
				FunctionName: &query,
			}
		},
		ListFuncPaginatorBuilder: func(client LambdaClient, input *lambda.ListFunctionsInput) adapterhelpers.Paginator[*lambda.ListFunctionsOutput, *lambda.Options] {
			return lambda.NewListFunctionsPaginator(client, input)
		},
		ListFuncOutputMapper: func(output *lambda.ListFunctionsOutput, input *lambda.ListFunctionsInput) ([]*lambda.GetFunctionInput, error) {
			inputs := make([]*lambda.GetFunctionInput, 0, len(output.Functions))

			for i := range output.Functions {
				inputs = append(inputs, &lambda.GetFunctionInput{
					FunctionName: output.Functions[i].FunctionName,
				})
			}

			return inputs, nil
		},
	}
}

var lambdaFunctionAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "lambda-function",
	DescriptiveName: "Lambda Function",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a lambda function by name",
		ListDescription:   "List all lambda functions",
		SearchDescription: "Search for lambda functions by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_lambda_function.arn"},
		{TerraformQueryMap: "aws_lambda_function_event_invoke_config.id"},
		{TerraformQueryMap: "aws_lambda_function_url.function_arn"},
	},
	PotentialLinks: []string{"iam-role", "s3-bucket", "sns-topic", "sqs-queue", "lambda-function", "events-event-bus", "elbv2-target-group", "vpc-lattice-target-group", "logs-log-group"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
})
