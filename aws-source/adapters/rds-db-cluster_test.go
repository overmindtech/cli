package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestDBClusterOutputMapper(t *testing.T) {
	output := rds.DescribeDBClustersOutput{
		DBClusters: []types.DBCluster{
			{
				AllocatedStorage: new(int32(100)),
				AvailabilityZones: []string{
					"eu-west-2c", // link
				},
				BackupRetentionPeriod:      new(int32(7)),
				DBClusterIdentifier:        new("database-2"),
				DBClusterParameterGroup:    new("default.postgres13"),
				DBSubnetGroup:              new("default-vpc-0d7892e00e573e701"), // link
				Status:                     new("available"),
				EarliestRestorableTime:     new(time.Now()),
				Endpoint:                   new("database-2.cluster-camcztjohmlj.eu-west-2.rds.amazonaws.com"),    // link
				ReaderEndpoint:             new("database-2.cluster-ro-camcztjohmlj.eu-west-2.rds.amazonaws.com"), // link
				MultiAZ:                    new(true),
				Engine:                     new("postgres"),
				EngineVersion:              new("13.7"),
				LatestRestorableTime:       new(time.Now()),
				Port:                       new(int32(5432)), // link
				MasterUsername:             new("postgres"),
				PreferredBackupWindow:      new("04:48-05:18"),
				PreferredMaintenanceWindow: new("fri:04:05-fri:04:35"),
				ReadReplicaIdentifiers: []string{
					"arn:aws:rds:eu-west-1:052392120703:cluster:read-replica", // link
				},
				DBClusterMembers: []types.DBClusterMember{
					{
						DBInstanceIdentifier:          new("database-2-instance-3"), // link
						IsClusterWriter:               new(false),
						DBClusterParameterGroupStatus: new("in-sync"),
						PromotionTier:                 new(int32(1)),
					},
				},
				VpcSecurityGroups: []types.VpcSecurityGroupMembership{
					{
						VpcSecurityGroupId: new("sg-094e151c9fc5da181"), // link
						Status:             new("active"),
					},
				},
				HostedZoneId:                     new("Z1TTGA775OQIYO"), // link
				StorageEncrypted:                 new(true),
				KmsKeyId:                         new("arn:aws:kms:eu-west-2:052392120703:key/9653cbdd-1590-464a-8456-67389cef6933"), // link
				DbClusterResourceId:              new("cluster-2EW4PDVN7F7V57CUJPYOEAA74M"),
				DBClusterArn:                     new("arn:aws:rds:eu-west-2:052392120703:cluster:database-2"),
				IAMDatabaseAuthenticationEnabled: new(false),
				ClusterCreateTime:                new(time.Now()),
				EngineMode:                       new("provisioned"),
				DeletionProtection:               new(false),
				HttpEndpointEnabled:              new(false),
				ActivityStreamStatus:             types.ActivityStreamStatusStopped,
				CopyTagsToSnapshot:               new(false),
				CrossAccountClone:                new(false),
				DomainMemberships:                []types.DomainMembership{},
				TagList:                          []types.Tag{},
				DBClusterInstanceClass:           new("db.m5d.large"),
				StorageType:                      new("io1"),
				Iops:                             new(int32(1000)),
				PubliclyAccessible:               new(true),
				AutoMinorVersionUpgrade:          new(true),
				MonitoringInterval:               new(int32(0)),
				PerformanceInsightsEnabled:       new(false),
				NetworkType:                      new("IPV4"),
				ActivityStreamKinesisStreamName:  new("aws-rds-das-db-AB1CDEFG23GHIJK4LMNOPQRST"), // link
				ActivityStreamKmsKeyId:           new("ab12345e-1111-2bc3-12a3-ab1cd12345e"),      // Not linking at the moment because there are too many possible formats. If you want to change this, submit a PR
				ActivityStreamMode:               types.ActivityStreamModeAsync,
				AutomaticRestartTime:             new(time.Now()),
				AssociatedRoles:                  []types.DBClusterRole{}, // EC2 classic roles, ignore
				BacktrackConsumedChangeRecords:   new(int64(1)),
				BacktrackWindow:                  new(int64(2)),
				Capacity:                         new(int32(2)),
				CharacterSetName:                 new("english"),
				CloneGroupId:                     new("id"),
				CustomEndpoints: []string{
					"endpoint1", // link dns
				},
				DBClusterOptionGroupMemberships: []types.DBClusterOptionGroupStatus{
					{
						DBClusterOptionGroupName: new("optionGroupName"), // link
						Status:                   new("good"),
					},
				},
				DBSystemId:            new("systemId"),
				DatabaseName:          new("databaseName"),
				EarliestBacktrackTime: new(time.Now()),
				EnabledCloudwatchLogsExports: []string{
					"logExport1",
				},
				GlobalWriteForwardingRequested: new(true),
				GlobalWriteForwardingStatus:    types.WriteForwardingStatusDisabled,
				MasterUserSecret: &types.MasterUserSecret{
					KmsKeyId:     new("arn:aws:kms:eu-west-2:052392120703:key/something"), // link
					SecretArn:    new("arn:aws:service:region:account:type/id"),           // link
					SecretStatus: new("okay"),
				},
				MonitoringRoleArn:                  new("arn:aws:service:region:account:type/id"), // link
				PendingModifiedValues:              &types.ClusterPendingModifiedValues{},
				PercentProgress:                    new("99"),
				PerformanceInsightsKMSKeyId:        new("arn:aws:service:region:account:type/id"), // link, assuming it's an ARN
				PerformanceInsightsRetentionPeriod: new(int32(99)),
				ReplicationSourceIdentifier:        new("arn:aws:rds:eu-west-2:052392120703:cluster:database-1"), // link
				ScalingConfigurationInfo: &types.ScalingConfigurationInfo{
					AutoPause:             new(true),
					MaxCapacity:           new(int32(10)),
					MinCapacity:           new(int32(1)),
					SecondsBeforeTimeout:  new(int32(10)),
					SecondsUntilAutoPause: new(int32(10)),
					TimeoutAction:         new("error"),
				},
				ServerlessV2ScalingConfiguration: &types.ServerlessV2ScalingConfigurationInfo{
					MaxCapacity: new(float64(10)),
					MinCapacity: new(float64(1)),
				},
			},
		},
	}

	items, err := dBClusterOutputMapper(context.Background(), mockRdsClient{}, "foo", nil, &output)

	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("got %v items, expected 1", len(items))
	}

	item := items[0]

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	if item.GetTags()["key"] != "value" {
		t.Errorf("expected tag key to be value, got %v", item.GetTags()["key"])
	}

	tests := QueryTests{
		{
			ExpectedType:   "rds-db-subnet-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default-vpc-0d7892e00e573e701",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "database-2.cluster-ro-camcztjohmlj.eu-west-2.rds.amazonaws.com",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "database-2.cluster-camcztjohmlj.eu-west-2.rds.amazonaws.com",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "rds-db-cluster",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:rds:eu-west-1:052392120703:cluster:read-replica",
			ExpectedScope:  "052392120703.eu-west-1",
		},
		{
			ExpectedType:   "rds-db-instance",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "database-2-instance-3",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-security-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "sg-094e151c9fc5da181",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "route53-hosted-zone",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "Z1TTGA775OQIYO",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:kms:eu-west-2:052392120703:key/9653cbdd-1590-464a-8456-67389cef6933",
			ExpectedScope:  "052392120703.eu-west-2",
		},
		{
			ExpectedType:   "kinesis-stream",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "aws-rds-das-db-AB1CDEFG23GHIJK4LMNOPQRST",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "endpoint1",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "rds-option-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "optionGroupName",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:kms:eu-west-2:052392120703:key/something",
			ExpectedScope:  "052392120703.eu-west-2",
		},
		{
			ExpectedType:   "secretsmanager-secret",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "iam-role",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "rds-db-cluster",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:rds:eu-west-2:052392120703:cluster:database-1",
			ExpectedScope:  "052392120703.eu-west-2",
		},
	}

	tests.Execute(t, item)
}

func TestNewRDSDBClusterAdapter(t *testing.T) {
	client, account, region := rdsGetAutoConfig(t)

	adapter := NewRDSDBClusterAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
