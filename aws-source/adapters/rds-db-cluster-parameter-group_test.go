package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/overmindtech/cli/sdpcache"
)

func TestDBClusterParameterGroupOutputMapper(t *testing.T) {
	group := ClusterParameterGroup{
		DBClusterParameterGroup: types.DBClusterParameterGroup{
			DBClusterParameterGroupName: PtrString("default.aurora-mysql5.7"),
			DBParameterGroupFamily:      PtrString("aurora-mysql5.7"),
			Description:                 PtrString("Default cluster parameter group for aurora-mysql5.7"),
			DBClusterParameterGroupArn:  PtrString("arn:aws:rds:eu-west-1:052392120703:cluster-pg:default.aurora-mysql5.7"),
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
				SupportedEngineModes: []string{
					"provisioned",
				},
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
				SupportedEngineModes: []string{
					"provisioned",
				},
			},
			{
				ParameterName: PtrString("aurora_binlog_replication_max_yield_seconds"),
				Description:   PtrString("Controls the number of seconds that binary log dump thread waits up to for the current binlog file to be filled by transactions. This wait period avoids contention that can arise from replicating each binlog event individually."),
				Source:        PtrString("engine-default"),
				ApplyType:     PtrString("dynamic"),
				DataType:      PtrString("integer"),
				AllowedValues: PtrString("0-36000"),
				IsModifiable:  PtrBool(true),
				ApplyMethod:   types.ApplyMethodPendingReboot,
				SupportedEngineModes: []string{
					"provisioned",
				},
			},
			{
				ParameterName: PtrString("aurora_enable_staggered_replica_restart"),
				Description:   PtrString("Allow Aurora replicas to follow a staggered restart schedule to increase cluster availability."),
				Source:        PtrString("system"),
				ApplyType:     PtrString("dynamic"),
				DataType:      PtrString("boolean"),
				AllowedValues: PtrString("0,1"),
				IsModifiable:  PtrBool(true),
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

	adapter := NewRDSDBClusterParameterGroupAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
