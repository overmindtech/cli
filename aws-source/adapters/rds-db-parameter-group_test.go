package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestDBParameterGroupOutputMapper(t *testing.T) {
	group := ParameterGroup{
		DBParameterGroup: types.DBParameterGroup{
			DBParameterGroupName:   adapterhelpers.PtrString("default.aurora-mysql5.7"),
			DBParameterGroupFamily: adapterhelpers.PtrString("aurora-mysql5.7"),
			Description:            adapterhelpers.PtrString("Default parameter group for aurora-mysql5.7"),
			DBParameterGroupArn:    adapterhelpers.PtrString("arn:aws:rds:eu-west-1:052392120703:pg:default.aurora-mysql5.7"),
		},
		Parameters: []types.Parameter{
			{
				ParameterName:  adapterhelpers.PtrString("activate_all_roles_on_login"),
				ParameterValue: adapterhelpers.PtrString("0"),
				Description:    adapterhelpers.PtrString("Automatically set all granted roles as active after the user has authenticated successfully."),
				Source:         adapterhelpers.PtrString("engine-default"),
				ApplyType:      adapterhelpers.PtrString("dynamic"),
				DataType:       adapterhelpers.PtrString("boolean"),
				AllowedValues:  adapterhelpers.PtrString("0,1"),
				IsModifiable:   adapterhelpers.PtrBool(true),
				ApplyMethod:    types.ApplyMethodPendingReboot,
			},
			{
				ParameterName: adapterhelpers.PtrString("allow-suspicious-udfs"),
				Description:   adapterhelpers.PtrString("Controls whether user-defined functions that have only an xxx symbol for the main function can be loaded"),
				Source:        adapterhelpers.PtrString("engine-default"),
				ApplyType:     adapterhelpers.PtrString("static"),
				DataType:      adapterhelpers.PtrString("boolean"),
				AllowedValues: adapterhelpers.PtrString("0,1"),
				IsModifiable:  adapterhelpers.PtrBool(false),
				ApplyMethod:   types.ApplyMethodPendingReboot,
			},
			{
				ParameterName: adapterhelpers.PtrString("aurora_parallel_query"),
				Description:   adapterhelpers.PtrString("This parameter can be used to enable and disable Aurora Parallel Query."),
				Source:        adapterhelpers.PtrString("engine-default"),
				ApplyType:     adapterhelpers.PtrString("dynamic"),
				DataType:      adapterhelpers.PtrString("boolean"),
				AllowedValues: adapterhelpers.PtrString("0,1"),
				IsModifiable:  adapterhelpers.PtrBool(true),
				ApplyMethod:   types.ApplyMethodPendingReboot,
			},
			{
				ParameterName: adapterhelpers.PtrString("autocommit"),
				Description:   adapterhelpers.PtrString("Sets the autocommit mode"),
				Source:        adapterhelpers.PtrString("engine-default"),
				ApplyType:     adapterhelpers.PtrString("dynamic"),
				DataType:      adapterhelpers.PtrString("boolean"),
				AllowedValues: adapterhelpers.PtrString("0,1"),
				IsModifiable:  adapterhelpers.PtrBool(true),
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

	adapter := NewRDSDBParameterGroupAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
