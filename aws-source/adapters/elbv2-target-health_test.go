package adapters

import (
	"context"
	"testing"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestTargetHealthOutputMapper(t *testing.T) {
	output := elbv2.DescribeTargetHealthOutput{
		TargetHealthDescriptions: []types.TargetHealthDescription{
			{
				Target: &types.TargetDescription{
					Id:               adapterhelpers.PtrString("10.0.6.64"), // link
					Port:             adapterhelpers.PtrInt32(8080),
					AvailabilityZone: adapterhelpers.PtrString("eu-west-2c"),
				},
				HealthCheckPort: adapterhelpers.PtrString("8080"),
				TargetHealth: &types.TargetHealth{
					State:       types.TargetHealthStateEnumHealthy,
					Reason:      types.TargetHealthReasonEnumDeregistrationInProgress,
					Description: adapterhelpers.PtrString("Health checks failed with these codes: [404]"),
				},
			},
			{
				Target: &types.TargetDescription{
					Id:               adapterhelpers.PtrString("arn:aws:elasticloadbalancing:eu-west-2:944651592624:loadbalancer/app/ingress/1bf10920c5bd199d"), // link
					Port:             adapterhelpers.PtrInt32(8080),
					AvailabilityZone: adapterhelpers.PtrString("eu-west-2c"),
				},
				HealthCheckPort: adapterhelpers.PtrString("8080"),
				TargetHealth: &types.TargetHealth{
					State:       types.TargetHealthStateEnumHealthy,
					Reason:      types.TargetHealthReasonEnumDeregistrationInProgress,
					Description: adapterhelpers.PtrString("Health checks failed with these codes: [404]"),
				},
			},
			{
				Target: &types.TargetDescription{
					Id:               adapterhelpers.PtrString("i-foo"), // link
					Port:             adapterhelpers.PtrInt32(8080),
					AvailabilityZone: adapterhelpers.PtrString("eu-west-2c"),
				},
				HealthCheckPort: adapterhelpers.PtrString("8080"),
				TargetHealth: &types.TargetHealth{
					State:       types.TargetHealthStateEnumHealthy,
					Reason:      types.TargetHealthReasonEnumDeregistrationInProgress,
					Description: adapterhelpers.PtrString("Health checks failed with these codes: [404]"),
				},
			},
			{
				Target: &types.TargetDescription{
					Id:               adapterhelpers.PtrString("arn:aws:lambda:eu-west-2:944651592624:function/foobar"), // link
					Port:             adapterhelpers.PtrInt32(8080),
					AvailabilityZone: adapterhelpers.PtrString("eu-west-2c"),
				},
				HealthCheckPort: adapterhelpers.PtrString("8080"),
				TargetHealth: &types.TargetHealth{
					State:       types.TargetHealthStateEnumHealthy,
					Reason:      types.TargetHealthReasonEnumDeregistrationInProgress,
					Description: adapterhelpers.PtrString("Health checks failed with these codes: [404]"),
				},
			},
		},
	}

	items, err := targetHealthOutputMapper(context.Background(), nil, "foo", &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: adapterhelpers.PtrString("arn:aws:elasticloadbalancing:eu-west-2:944651592624:targetgroup/k8s-default-apiserve-d87e8f7010/559d207158e41222"),
	}, &output)

	if err != nil {
		t.Error(err)
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Error(err)
		}
	}

	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %v", len(items))
	}

	item := items[0]

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "ip",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "10.0.6.64",
			ExpectedScope:  "global",
		},
	}

	tests.Execute(t, item)

	item = items[1]

	tests = adapterhelpers.QueryTests{
		{
			ExpectedType:   "elbv2-load-balancer",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:elasticloadbalancing:eu-west-2:944651592624:loadbalancer/app/ingress/1bf10920c5bd199d",
			ExpectedScope:  "944651592624.eu-west-2",
		},
	}

	tests.Execute(t, item)

	item = items[2]

	tests = adapterhelpers.QueryTests{
		{
			ExpectedType:   "ec2-instance",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "i-foo",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)

	item = items[3]

	tests = adapterhelpers.QueryTests{
		{
			ExpectedType:   "lambda-function",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:lambda:eu-west-2:944651592624:function/foobar",
			ExpectedScope:  "944651592624.eu-west-2",
		},
	}

	tests.Execute(t, item)
}

func TestTargetHealthUniqueID(t *testing.T) {
	t.Run("with an ARN as the ID", func(t *testing.T) {
		id := TargetHealthUniqueID{
			TargetGroupArn: "arn:aws:elasticloadbalancing:eu-west-2:944651592624:targetgroup/k8s-default-apiserve-d87e8f7010/559d207158e41222",
			Id:             "arn:partition:service:region:account-id:resource-type:resource-id",
		}

		expected := "arn:aws:elasticloadbalancing:eu-west-2:944651592624:targetgroup/k8s-default-apiserve-d87e8f7010/559d207158e41222|arn:partition:service:region:account-id:resource-type:resource-id||"

		if id.String() != expected {
			t.Errorf("expected string value to be %v\ngot %v", expected, id.String())
		}

		t.Run("converting back", func(t *testing.T) {
			newID, err := ToTargetHealthUniqueID(expected)

			if err != nil {
				t.Error(err)
			}

			CompareTargetHealthUniqueID(newID, id, t)
		})
	})

	t.Run("with an IP as the ID", func(t *testing.T) {
		id := TargetHealthUniqueID{
			TargetGroupArn:   "arn:aws:elasticloadbalancing:eu-west-2:944651592624:targetgroup/k8s-default-apiserve-d87e8f7010/559d207158e41222",
			Id:               "10.0.0.1",
			AvailabilityZone: adapterhelpers.PtrString("eu-west-2"),
			Port:             adapterhelpers.PtrInt32(8080),
		}

		expected := "arn:aws:elasticloadbalancing:eu-west-2:944651592624:targetgroup/k8s-default-apiserve-d87e8f7010/559d207158e41222|10.0.0.1|eu-west-2|8080"

		if id.String() != expected {
			t.Errorf("expected string value to be %v\ngot %v", expected, id.String())
		}

		t.Run("converting back", func(t *testing.T) {
			newID, err := ToTargetHealthUniqueID(expected)

			if err != nil {
				t.Error(err)
			}

			CompareTargetHealthUniqueID(newID, id, t)
		})
	})

	t.Run("with an ARN as the ID and a port", func(t *testing.T) {
		id := TargetHealthUniqueID{
			TargetGroupArn: "arn:aws:elasticloadbalancing:eu-west-2:944651592624:targetgroup/k8s-default-apiserve-d87e8f7010/559d207158e41222",
			Id:             "arn:partition:service:region:account-id:resource-type:resource-id",
			Port:           adapterhelpers.PtrInt32(8080),
		}

		expected := "arn:aws:elasticloadbalancing:eu-west-2:944651592624:targetgroup/k8s-default-apiserve-d87e8f7010/559d207158e41222|arn:partition:service:region:account-id:resource-type:resource-id||8080"

		if id.String() != expected {
			t.Errorf("expected string value to be %v\ngot %v", expected, id.String())
		}

		t.Run("converting back", func(t *testing.T) {
			newID, err := ToTargetHealthUniqueID(expected)

			if err != nil {
				t.Error(err)
			}

			CompareTargetHealthUniqueID(newID, id, t)
		})
	})
}

func CompareTargetHealthUniqueID(x, y TargetHealthUniqueID, t *testing.T) {
	if x.AvailabilityZone != nil {
		if *x.AvailabilityZone != *y.AvailabilityZone {
			t.Errorf("AvailabilityZone mismatch!\nX: %v\nY: %v", x.AvailabilityZone, y.AvailabilityZone)
		}
	}

	if x.Id != y.Id {
		t.Errorf("Id mismatch!\nX: %v\nY: %v", x.Id, y.Id)
	}

	if x.Port != nil {
		if *x.Port != *y.Port {
			t.Errorf("Port mismatch!\nX: %v\nY: %v", x.Port, y.Port)
		}
	}
	if x.TargetGroupArn != y.TargetGroupArn {
		t.Errorf("TargetGroupArn mismatch!\nX: %v\nY: %v", x.TargetGroupArn, y.TargetGroupArn)
	}
}
