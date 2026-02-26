package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestDBParameterGroupOutputMapper(t *testing.T) {
	group := ParameterGroup{
		DBParameterGroup: types.DBParameterGroup{
			DBParameterGroupName:   new("default.aurora-mysql5.7"),
			DBParameterGroupFamily: new("aurora-mysql5.7"),
			Description:            new("Default parameter group for aurora-mysql5.7"),
			DBParameterGroupArn:    new("arn:aws:rds:eu-west-1:052392120703:pg:default.aurora-mysql5.7"),
		},
		Parameters: []types.Parameter{
			{
				ParameterName:  new("activate_all_roles_on_login"),
				ParameterValue: new("0"),
				Description:    new("Automatically set all granted roles as active after the user has authenticated successfully."),
				Source:         new("engine-default"),
				ApplyType:      new("dynamic"),
				DataType:       new("boolean"),
				AllowedValues:  new("0,1"),
				IsModifiable:   new(true),
				ApplyMethod:    types.ApplyMethodPendingReboot,
			},
			{
				ParameterName: new("allow-suspicious-udfs"),
				Description:   new("Controls whether user-defined functions that have only an xxx symbol for the main function can be loaded"),
				Source:        new("engine-default"),
				ApplyType:     new("static"),
				DataType:      new("boolean"),
				AllowedValues: new("0,1"),
				IsModifiable:  new(false),
				ApplyMethod:   types.ApplyMethodPendingReboot,
			},
			{
				ParameterName: new("aurora_parallel_query"),
				Description:   new("This parameter can be used to enable and disable Aurora Parallel Query."),
				Source:        new("engine-default"),
				ApplyType:     new("dynamic"),
				DataType:      new("boolean"),
				AllowedValues: new("0,1"),
				IsModifiable:  new(true),
				ApplyMethod:   types.ApplyMethodPendingReboot,
			},
			{
				ParameterName: new("autocommit"),
				Description:   new("Sets the autocommit mode"),
				Source:        new("engine-default"),
				ApplyType:     new("dynamic"),
				DataType:      new("boolean"),
				AllowedValues: new("0,1"),
				IsModifiable:  new(true),
				ApplyMethod:   types.ApplyMethodPendingReboot,
			},
		},
	}

	item, err := dBParameterGroupItemMapper("", "foo", &group)

	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}
}

func TestNewRDSDBParameterGroupAdapter(t *testing.T) {
	client, account, region := rdsGetAutoConfig(t)

	adapter := NewRDSDBParameterGroupAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
