package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestDBClusterParameterGroupOutputMapper(t *testing.T) {
	group := ClusterParameterGroup{
		DBClusterParameterGroup: types.DBClusterParameterGroup{
			DBClusterParameterGroupName: new("default.aurora-mysql5.7"),
			DBParameterGroupFamily:      new("aurora-mysql5.7"),
			Description:                 new("Default cluster parameter group for aurora-mysql5.7"),
			DBClusterParameterGroupArn:  new("arn:aws:rds:eu-west-1:052392120703:cluster-pg:default.aurora-mysql5.7"),
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
				SupportedEngineModes: []string{
					"provisioned",
				},
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
				SupportedEngineModes: []string{
					"provisioned",
				},
			},
			{
				ParameterName: new("aurora_binlog_replication_max_yield_seconds"),
				Description:   new("Controls the number of seconds that binary log dump thread waits up to for the current binlog file to be filled by transactions. This wait period avoids contention that can arise from replicating each binlog event individually."),
				Source:        new("engine-default"),
				ApplyType:     new("dynamic"),
				DataType:      new("integer"),
				AllowedValues: new("0-36000"),
				IsModifiable:  new(true),
				ApplyMethod:   types.ApplyMethodPendingReboot,
				SupportedEngineModes: []string{
					"provisioned",
				},
			},
			{
				ParameterName: new("aurora_enable_staggered_replica_restart"),
				Description:   new("Allow Aurora replicas to follow a staggered restart schedule to increase cluster availability."),
				Source:        new("system"),
				ApplyType:     new("dynamic"),
				DataType:      new("boolean"),
				AllowedValues: new("0,1"),
				IsModifiable:  new(true),
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
