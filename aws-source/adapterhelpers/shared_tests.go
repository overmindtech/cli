package adapterhelpers

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/sdp-go"
)

func PtrString(v string) *string {
	return &v
}

func PtrInt32(v int32) *int32 {
	return &v
}

func PtrInt64(v int64) *int64 {
	return &v
}

func PtrFloat32(v float32) *float32 {
	return &v
}

func PtrFloat64(v float64) *float64 {
	return &v
}

func PtrTime(v time.Time) *time.Time {
	return &v
}

func PtrBool(v bool) *bool {
	return &v
}

type Subnet struct {
	ID               *string
	CIDR             string
	AvailabilityZone string
}

type VPCConfig struct {
	// These are populated after Fetching
	ID *string

	// Subnets in this VPC
	Subnets []*Subnet

	cleanupFunctions []func()
}

var purposeKey = "Purpose"
var nameKey = "Name"
var tagValue = "automated-testing-" + time.Now().Format("2006-01-02T15:04:05.000Z")
var TestTags = []types.Tag{
	{
		Key:   &purposeKey,
		Value: &tagValue,
	},
	{
		Key:   &nameKey,
		Value: &tagValue,
	},
}

func (v *VPCConfig) Cleanup(f func()) {
	v.cleanupFunctions = append(v.cleanupFunctions, f)
}

func (v *VPCConfig) RunCleanup() {
	for len(v.cleanupFunctions) > 0 {
		n := len(v.cleanupFunctions) - 1 // Top element

		v.cleanupFunctions[n]()

		v.cleanupFunctions = v.cleanupFunctions[:n] // Pop
	}
}

// Fetch Fetches the VPC and subnets and registers cleanup actions for them
func (v *VPCConfig) Fetch(client *ec2.Client) error {
	// manually configured VPC in eu-west-2
	vpcid := "vpc-061f0bb58acec88ad"
	v.ID = &vpcid // vpcOutput.Vpc.VpcId
	filterName := "vpc-id"
	subnetOutput, err := client.DescribeSubnets(
		context.Background(),
		&ec2.DescribeSubnetsInput{
			Filters: []types.Filter{
				{
					Name:   &filterName,
					Values: []string{vpcid},
				},
			},
		},
	)

	if err != nil {
		return err
	}

	for _, subnet := range subnetOutput.Subnets {
		v.Subnets = append(v.Subnets, &Subnet{
			ID:               subnet.SubnetId,
			CIDR:             *subnet.CidrBlock,
			AvailabilityZone: *subnet.AvailabilityZone,
		})
	}

	return nil
}

// CreateGateway Creates a new internet gateway for the duration of the test to save 40$ per month vs running it 24/7
func (v *VPCConfig) CreateGateway(client *ec2.Client) error {
	var err error

	// Create internet gateway and assign to VPC
	var gatewayOutput *ec2.CreateInternetGatewayOutput

	gatewayOutput, err = client.CreateInternetGateway(
		context.Background(),
		&ec2.CreateInternetGatewayInput{
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeInternetGateway,
					Tags:         TestTags,
				},
			},
		},
	)

	if err != nil {
		return err
	}

	internetGatewayId := gatewayOutput.InternetGateway.InternetGatewayId

	v.Cleanup(func() {
		del := func() error {
			_, err := client.DeleteInternetGateway(
				context.Background(),
				&ec2.DeleteInternetGatewayInput{
					InternetGatewayId: internetGatewayId,
				},
			)

			return err
		}

		err := retry(10, time.Second, del)

		if err != nil {
			log.Println(err)
		}
	})

	_, err = client.AttachInternetGateway(
		context.Background(),
		&ec2.AttachInternetGatewayInput{
			InternetGatewayId: internetGatewayId,
			VpcId:             v.ID,
		},
	)

	if err != nil {
		return err
	}

	v.Cleanup(func() {
		del := func() error {
			_, err := client.DetachInternetGateway(
				context.Background(),
				&ec2.DetachInternetGatewayInput{
					InternetGatewayId: internetGatewayId,
					VpcId:             v.ID,
				},
			)

			return err
		}

		err := retry(10, time.Second, del)

		if err != nil {
			log.Println(err)
		}
	})
	return nil
}

func retry(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := range attempts {
		if i > 0 {
			time.Sleep(sleep)
			sleep *= 2
		}
		err = f()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("after %d attempts, last error: %w", attempts, err)
}

type QueryTest struct {
	ExpectedType   string
	ExpectedMethod sdp.QueryMethod
	ExpectedQuery  string
	ExpectedScope  string
}

type QueryTests []QueryTest

func (i QueryTests) Execute(t *testing.T, item *sdp.Item) {
	for _, test := range i {
		var found bool

		// TODO(LIQs): update this to receive and evaluate edges instead of linked item queries
		for _, lir := range item.GetLinkedItemQueries() {
			if lirMatches(test, lir.GetQuery()) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("could not find linked item request in %v requests.\nType: %v\nQuery: %v\nScope: %v", len(item.GetLinkedItemQueries()), test.ExpectedType, test.ExpectedQuery, test.ExpectedScope)
		}
	}
}

func lirMatches(test QueryTest, req *sdp.Query) bool {
	return (test.ExpectedMethod == req.GetMethod() &&
		test.ExpectedQuery == req.GetQuery() &&
		test.ExpectedScope == req.GetScope() &&
		test.ExpectedType == req.GetType())
}

// CheckQuery Checks that an item request matches the expected params
func CheckQuery(t *testing.T, item *sdp.Query, itemName string, expectedType string, expectedQuery string, expectedScope string) {
	if item.GetType() != expectedType {
		t.Errorf("%s.Type '%v' != '%v'", itemName, item.GetType(), expectedType)
	}
	if item.GetMethod() != sdp.QueryMethod_GET {
		t.Errorf("%s.Method '%v' != '%v'", itemName, item.GetMethod(), sdp.QueryMethod_GET)
	}
	if item.GetQuery() != expectedQuery {
		t.Errorf("%s.Query '%v' != '%v'", itemName, item.GetQuery(), expectedQuery)
	}
	if item.GetScope() != expectedScope {
		t.Errorf("%s.Scope '%v' != '%v'", itemName, item.GetScope(), expectedScope)
	}
}
