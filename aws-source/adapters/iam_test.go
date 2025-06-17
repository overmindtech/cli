package adapters

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/micahhausler/aws-iam-policy/policy"
	"github.com/overmindtech/cli/tracing"
)

// TestIAMClient Test client that returns three pages
type TestIAMClient struct{}

func TestMain(m *testing.M) {
	exitCode := func() int {
		defer tracing.ShutdownTracer(context.Background())

		if err := tracing.InitTracerWithUpstreams("aws-source-tests", os.Getenv("HONEYCOMB_API_KEY"), ""); err != nil {
			log.Fatal(err)
		}

		return m.Run()
	}()

	os.Exit(exitCode)
}

func TestLinksFromPolicy(t *testing.T) {
	t.Run("with a simple policy that extracts a principal", func(t *testing.T) {
		action := policy.NewStringOrSlice(true, "sts:AssumeRole")
		pol := policy.Policy{
			Statements: policy.NewStatementOrSlice(
				policy.Statement{
					Action:    action,
					Effect:    "Allow",
					Principal: policy.NewAWSPrincipal("arn:aws:iam::123456789:role/aws-controltower-AuditAdministratorRole"),
				},
			),
		}

		queries := LinksFromPolicy(&pol)

		if len(queries) != 1 {
			t.Fatalf("expected 1 query got %v", len(queries))
		}
	})

	t.Run("with a simple policy that something from the resource using teh fallback extractor", func(t *testing.T) {
		action := policy.NewStringOrSlice(true, "sts:AssumeRole")
		pol := policy.Policy{
			Statements: policy.NewStatementOrSlice(
				policy.Statement{
					Action:   action,
					Effect:   "Allow",
					Resource: policy.NewStringOrSlice(true, "arn:aws:iam::123456789:role/aws-controltower-AuditAdministratorRole"),
				},
			),
		}

		queries := LinksFromPolicy(&pol)

		if len(queries) != 1 {
			t.Fatalf("expected 1 query got %v", len(queries))
		}
	})

	t.Run("with a simple policy that something from the resource using the SSM extractor", func(t *testing.T) {
		action := policy.NewStringOrSlice(true, "ssm:GetParameter")
		pol := policy.Policy{
			Statements: policy.NewStatementOrSlice(
				policy.Statement{
					Action:   action,
					Effect:   "Allow",
					Resource: policy.NewStringOrSlice(true, "arn:aws:ssm:us-west-2:123456789:parameter/foo"),
				},
			),
		}

		queries := LinksFromPolicy(&pol)

		if len(queries) != 1 {
			t.Fatalf("expected 1 query got %v", len(queries))
		}

		// This should have had an asterisk added
		if queries[0].GetQuery().GetQuery() != "arn:aws:ssm:us-west-2:123456789:parameter/foo*" {
			t.Errorf("expected query to be 'arn:aws:ssm:us-west-2:123456789:parameter/foo*' got %v", queries[0].GetQuery().GetQuery())
		}
	})

}
func TestLinksFromPolicy_EdgeCases(t *testing.T) {
	t.Run("nil policy returns empty slice", func(t *testing.T) {
		queries := LinksFromPolicy(nil)
		if len(queries) != 0 {
			t.Errorf("expected 0 queries, got %v", len(queries))
		}
	})

	t.Run("policy with nil statements returns empty slice", func(t *testing.T) {
		pol := &policy.Policy{}
		queries := LinksFromPolicy(pol)
		if len(queries) != 0 {
			t.Errorf("expected 0 queries, got %v", len(queries))
		}
	})

	t.Run("policy with statement with non-ARN principal", func(t *testing.T) {
		action := policy.NewStringOrSlice(true, "sts:AssumeRole")
		pol := policy.Policy{
			Statements: policy.NewStatementOrSlice(
				policy.Statement{
					Action:    action,
					Effect:    "Allow",
					Principal: policy.NewAWSPrincipal("not-an-arn"),
				},
			),
		}
		queries := LinksFromPolicy(&pol)
		if len(queries) != 0 {
			t.Errorf("expected 0 queries, got %v", len(queries))
		}
	})

	t.Run("policy with statement with principal of unknown type", func(t *testing.T) {
		action := policy.NewStringOrSlice(true, "sts:AssumeRole")
		// This ARN has a made-up type
		pol := policy.Policy{
			Statements: policy.NewStatementOrSlice(
				policy.Statement{
					Action:    action,
					Effect:    "Allow",
					Principal: policy.NewAWSPrincipal("arn:aws:iam::123456789:foobar/aws-controltower-AuditAdministratorRole"),
				},
			),
		}
		queries := LinksFromPolicy(&pol)
		if len(queries) != 0 {
			t.Errorf("expected 0 queries, got %v", len(queries))
		}
	})

	t.Run("policy with statement with resource but no action", func(t *testing.T) {
		pol := policy.Policy{
			Statements: policy.NewStatementOrSlice(
				policy.Statement{
					Effect:   "Allow",
					Resource: policy.NewStringOrSlice(true, "arn:aws:ssm:us-west-2:123456789:parameter/foo"),
				},
			),
		}
		queries := LinksFromPolicy(&pol)
		if len(queries) != 0 {
			t.Errorf("expected 0 queries, got %v", len(queries))
		}
	})

	t.Run("policy with statement with resource that is not an ARN", func(t *testing.T) {
		action := policy.NewStringOrSlice(true, "ssm:GetParameter")
		pol := policy.Policy{
			Statements: policy.NewStatementOrSlice(
				policy.Statement{
					Action:   action,
					Effect:   "Allow",
					Resource: policy.NewStringOrSlice(true, "not-an-arn"),
				},
			),
		}
		queries := LinksFromPolicy(&pol)
		if len(queries) != 0 {
			t.Errorf("expected 0 queries, got %v", len(queries))
		}
	})

	t.Run("policy with multiple statements and mixed valid/invalid principals and resources", func(t *testing.T) {
		action := policy.NewStringOrSlice(true, "sts:AssumeRole")
		ssmAction := policy.NewStringOrSlice(true, "ssm:GetParameter")
		pol := policy.Policy{
			Statements: policy.NewStatementOrSlice(
				policy.Statement{
					Action:    action,
					Effect:    "Allow",
					Principal: policy.NewAWSPrincipal("arn:aws:iam::123456789:role/aws-controltower-AuditAdministratorRole"),
				},
				policy.Statement{
					Action:   ssmAction,
					Effect:   "Allow",
					Resource: policy.NewStringOrSlice(true, "arn:aws:ssm:us-west-2:123456789:parameter/foo"),
				},
				policy.Statement{
					Action:   action,
					Effect:   "Allow",
					Resource: policy.NewStringOrSlice(true, "not-an-arn"),
				},
			),
		}
		queries := LinksFromPolicy(&pol)
		if len(queries) != 2 {
			t.Errorf("expected 2 queries, got %v", len(queries))
		}
	})
}
