package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/overmindtech/cli/sdp-go"
)

func TestDBClusterOutputMapper(t *testing.T) {
	output := rds.DescribeDBClustersOutput{
		DBClusters: []types.DBCluster{
			{
				AllocatedStorage: PtrInt32(100),
				AvailabilityZones: []string{
					"eu-west-2c", // link
				},
				BackupRetentionPeriod:      PtrInt32(7),
				DBClusterIdentifier:        PtrString("database-2"),
				DBClusterParameterGroup:    PtrString("default.postgres13"),
				DBSubnetGroup:              PtrString("default-vpc-0d7892e00e573e701"), // link
				Status:                     PtrString("available"),
				EarliestRestorableTime:     PtrTime(time.Now()),
				Endpoint:                   PtrString("database-2.cluster-camcztjohmlj.eu-west-2.rds.amazonaws.com"),    // link
				ReaderEndpoint:             PtrString("database-2.cluster-ro-camcztjohmlj.eu-west-2.rds.amazonaws.com"), // link
				MultiAZ:                    PtrBool(true),
				Engine:                     PtrString("postgres"),
				EngineVersion:              PtrString("13.7"),
				LatestRestorableTime:       PtrTime(time.Now()),
				Port:                       PtrInt32(5432), // link
				MasterUsername:             PtrString("postgres"),
				PreferredBackupWindow:      PtrString("04:48-05:18"),
				PreferredMaintenanceWindow: PtrString("fri:04:05-fri:04:35"),
				ReadReplicaIdentifiers: []string{
					"arn:aws:rds:eu-west-1:052392120703:cluster:read-replica", // link
				},
				DBClusterMembers: []types.DBClusterMember{
					{
						DBInstanceIdentifier:          PtrString("database-2-instance-3"), // link
						IsClusterWriter:               PtrBool(false),
						DBClusterParameterGroupStatus: PtrString("in-sync"),
						PromotionTier:                 PtrInt32(1),
					},
				},
				VpcSecurityGroups: []types.VpcSecurityGroupMembership{
					{
						VpcSecurityGroupId: PtrString("sg-094e151c9fc5da181"), // link
						Status:             PtrString("active"),
					},
				},
				HostedZoneId:                     PtrString("Z1TTGA775OQIYO"), // link
				StorageEncrypted:                 PtrBool(true),
				KmsKeyId:                         PtrString("arn:aws:kms:eu-west-2:052392120703:key/9653cbdd-1590-464a-8456-67389cef6933"), // link
				DbClusterResourceId:              PtrString("cluster-2EW4PDVN7F7V57CUJPYOEAA74M"),
				DBClusterArn:                     PtrString("arn:aws:rds:eu-west-2:052392120703:cluster:database-2"),
				IAMDatabaseAuthenticationEnabled: PtrBool(false),
				ClusterCreateTime:                PtrTime(time.Now()),
				EngineMode:                       PtrString("provisioned"),
				DeletionProtection:               PtrBool(false),
				HttpEndpointEnabled:              PtrBool(false),
				ActivityStreamStatus:             types.ActivityStreamStatusStopped,
				CopyTagsToSnapshot:               PtrBool(false),
				CrossAccountClone:                PtrBool(false),
				DomainMemberships:                []types.DomainMembership{},
				TagList:                          []types.Tag{},
				DBClusterInstanceClass:           PtrString("db.m5d.large"),
				StorageType:                      PtrString("io1"),
				Iops:                             PtrInt32(1000),
				PubliclyAccessible:               PtrBool(true),
				AutoMinorVersionUpgrade:          PtrBool(true),
				MonitoringInterval:               PtrInt32(0),
				PerformanceInsightsEnabled:       PtrBool(false),
				NetworkType:                      PtrString("IPV4"),
				ActivityStreamKinesisStreamName:  PtrString("aws-rds-das-db-AB1CDEFG23GHIJK4LMNOPQRST"), // link
				ActivityStreamKmsKeyId:           PtrString("ab12345e-1111-2bc3-12a3-ab1cd12345e"),      // Not linking at the moment because there are too many possible formats. If you want to change this, submit a PR
				ActivityStreamMode:               types.ActivityStreamModeAsync,
				AutomaticRestartTime:             PtrTime(time.Now()),
				AssociatedRoles:                  []types.DBClusterRole{}, // EC2 classic roles, ignore
				BacktrackConsumedChangeRecords:   PtrInt64(1),
				BacktrackWindow:                  PtrInt64(2),
				Capacity:                         PtrInt32(2),
				CharacterSetName:                 PtrString("english"),
				CloneGroupId:                     PtrString("id"),
				CustomEndpoints: []string{
					"endpoint1", // link dns
				},
				DBClusterOptionGroupMemberships: []types.DBClusterOptionGroupStatus{
					{
						DBClusterOptionGroupName: PtrString("optionGroupName"), // link
						Status:                   PtrString("good"),
					},
				},
				DBSystemId:            PtrString("systemId"),
				DatabaseName:          PtrString("databaseName"),
				EarliestBacktrackTime: PtrTime(time.Now()),
				EnabledCloudwatchLogsExports: []string{
					"logExport1",
				},
				GlobalWriteForwardingRequested: PtrBool(true),
				GlobalWriteForwardingStatus:    types.WriteForwardingStatusDisabled,
				MasterUserSecret: &types.MasterUserSecret{
					KmsKeyId:     PtrString("arn:aws:kms:eu-west-2:052392120703:key/something"), // link
					SecretArn:    PtrString("arn:aws:service:region:account:type/id"),           // link
					SecretStatus: PtrString("okay"),
				},
				MonitoringRoleArn:                  PtrString("arn:aws:service:region:account:type/id"), // link
				PendingModifiedValues:              &types.ClusterPendingModifiedValues{},
				PercentProgress:                    PtrString("99"),
				PerformanceInsightsKMSKeyId:        PtrString("arn:aws:service:region:account:type/id"), // link, assuming it's an ARN
				PerformanceInsightsRetentionPeriod: PtrInt32(99),
				ReplicationSourceIdentifier:        PtrString("arn:aws:rds:eu-west-2:052392120703:cluster:database-1"), // link
				ScalingConfigurationInfo: &types.ScalingConfigurationInfo{
					AutoPause:             PtrBool(true),
					MaxCapacity:           PtrInt32(10),
					MinCapacity:           PtrInt32(1),
					SecondsBeforeTimeout:  PtrInt32(10),
					SecondsUntilAutoPause: PtrInt32(10),
					TimeoutAction:         PtrString("error"),
				},
				ServerlessV2ScalingConfiguration: &types.ServerlessV2ScalingConfigurationInfo{
					MaxCapacity: PtrFloat64(10),
					MinCapacity: PtrFloat64(1),
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

	adapter := NewRDSDBClusterAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
