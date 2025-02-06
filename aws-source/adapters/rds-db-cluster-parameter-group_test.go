package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestDBClusterParameterGroupOutputMapper(t *testing.T) {
	group := ClusterParameterGroup{
		DBClusterParameterGroup: types.DBClusterParameterGroup{
			DBClusterParameterGroupName: adapterhelpers.PtrString("default.aurora-mysql5.7"),
			DBParameterGroupFamily:      adapterhelpers.PtrString("aurora-mysql5.7"),
			Description:                 adapterhelpers.PtrString("Default cluster parameter group for aurora-mysql5.7"),
			DBClusterParameterGroupArn:  adapterhelpers.PtrString("arn:aws:rds:eu-west-1:052392120703:cluster-pg:default.aurora-mysql5.7"),
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
				SupportedEngineModes: []string{
					"provisioned",
				},
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
				SupportedEngineModes: []string{
					"provisioned",
				},
			},
			{
				ParameterName: adapterhelpers.PtrString("aurora_binlog_replication_max_yield_seconds"),
				Description:   adapterhelpers.PtrString("Controls the number of seconds that binary log dump thread waits up to for the current binlog file to be filled by transactions. This wait period avoids contention that can arise from replicating each binlog event individually."),
				Source:        adapterhelpers.PtrString("engine-default"),
				ApplyType:     adapterhelpers.PtrString("dynamic"),
				DataType:      adapterhelpers.PtrString("integer"),
				AllowedValues: adapterhelpers.PtrString("0-36000"),
				IsModifiable:  adapterhelpers.PtrBool(true),
				ApplyMethod:   types.ApplyMethodPendingReboot,
				SupportedEngineModes: []string{
					"provisioned",
				},
			},
			{
				ParameterName: adapterhelpers.PtrString("aurora_enable_staggered_replica_restart"),
				Description:   adapterhelpers.PtrString("Allow Aurora replicas to follow a staggered restart schedule to increase cluster availability."),
				Source:        adapterhelpers.PtrString("system"),
				ApplyType:     adapterhelpers.PtrString("dynamic"),
				DataType:      adapterhelpers.PtrString("boolean"),
				AllowedValues: adapterhelpers.PtrString("0,1"),
				IsModifiable:  adapterhelpers.PtrBool(true),
				ApplyMethod:   types.ApplyMethodImmediate,
				SupportedEngineModes: []string{
					"provisioned",
				},
			},
		},
	}

	item, err := dBClusterParameterGroupItemMapper("", "foo", &group)

	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}
}

func TestNewRDSDBClusterParameterGroupAdapter(t *testing.T) {
	client, account, region := rdsGetAutoConfig(t)

	adapter := NewRDSDBClusterParameterGroupAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
