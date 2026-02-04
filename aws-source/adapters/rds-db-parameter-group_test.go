package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/overmindtech/cli/sdpcache"
)

func TestDBParameterGroupOutputMapper(t *testing.T) {
	group := ParameterGroup{
		DBParameterGroup: types.DBParameterGroup{
			DBParameterGroupName:   PtrString("default.aurora-mysql5.7"),
			DBParameterGroupFamily: PtrString("aurora-mysql5.7"),
			Description:            PtrString("Default parameter group for aurora-mysql5.7"),
			DBParameterGroupArn:    PtrString("arn:aws:rds:eu-west-1:052392120703:pg:default.aurora-mysql5.7"),
		},
		Parameters: []types.Parameter{
			{
				ParameterName:  PtrString("activate_all_roles_on_login"),
				ParameterValue: PtrString("0"),
				Description:    PtrString("Automatically set all granted roles as active after the user has authenticated successfully."),
				Source:         PtrString("engine-default"),
				ApplyType:      PtrString("dynamic"),
				DataType:       PtrString("boolean"),
				AllowedValues:  PtrString("0,1"),
				IsModifiable:   PtrBool(true),
				ApplyMethod:    types.ApplyMethodPendingReboot,
			},
			{
				ParameterName: PtrString("allow-suspicious-udfs"),
				Description:   PtrString("Controls whether user-defined functions that have only an xxx symbol for the main function can be loaded"),
				Source:        PtrString("engine-default"),
				ApplyType:     PtrString("static"),
				DataType:      PtrString("boolean"),
				AllowedValues: PtrString("0,1"),
				IsModifiable:  PtrBool(false),
				ApplyMethod:   types.ApplyMethodPendingReboot,
			},
			{
				ParameterName: PtrString("aurora_parallel_query"),
				Description:   PtrString("This parameter can be used to enable and disable Aurora Parallel Query."),
				Source:        PtrString("engine-default"),
				ApplyType:     PtrString("dynamic"),
				DataType:      PtrString("boolean"),
				AllowedValues: PtrString("0,1"),
				IsModifiable:  PtrBool(true),
				ApplyMethod:   types.ApplyMethodPendingReboot,
			},
			{
				ParameterName: PtrString("autocommit"),
				Description:   PtrString("Sets the autocommit mode"),
				Source:        PtrString("engine-default"),
				ApplyType:     PtrString("dynamic"),
				DataType:      PtrString("boolean"),
				AllowedValues: PtrString("0,1"),
				IsModifiable:  PtrBool(true),
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
