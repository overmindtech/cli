package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestDBClusterOutputMapper(t *testing.T) {
	output := rds.DescribeDBClustersOutput{
		DBClusters: []types.DBCluster{
			{
				AllocatedStorage: adapterhelpers.PtrInt32(100),
				AvailabilityZones: []string{
					"eu-west-2c", // link
				},
				BackupRetentionPeriod:      adapterhelpers.PtrInt32(7),
				DBClusterIdentifier:        adapterhelpers.PtrString("database-2"),
				DBClusterParameterGroup:    adapterhelpers.PtrString("default.postgres13"),
				DBSubnetGroup:              adapterhelpers.PtrString("default-vpc-0d7892e00e573e701"), // link
				Status:                     adapterhelpers.PtrString("available"),
				EarliestRestorableTime:     adapterhelpers.PtrTime(time.Now()),
				Endpoint:                   adapterhelpers.PtrString("database-2.cluster-camcztjohmlj.eu-west-2.rds.amazonaws.com"),    // link
				ReaderEndpoint:             adapterhelpers.PtrString("database-2.cluster-ro-camcztjohmlj.eu-west-2.rds.amazonaws.com"), // link
				MultiAZ:                    adapterhelpers.PtrBool(true),
				Engine:                     adapterhelpers.PtrString("postgres"),
				EngineVersion:              adapterhelpers.PtrString("13.7"),
				LatestRestorableTime:       adapterhelpers.PtrTime(time.Now()),
				Port:                       adapterhelpers.PtrInt32(5432), // link
				MasterUsername:             adapterhelpers.PtrString("postgres"),
				PreferredBackupWindow:      adapterhelpers.PtrString("04:48-05:18"),
				PreferredMaintenanceWindow: adapterhelpers.PtrString("fri:04:05-fri:04:35"),
				ReadReplicaIdentifiers: []string{
					"arn:aws:rds:eu-west-1:052392120703:cluster:read-replica", // link
				},
				DBClusterMembers: []types.DBClusterMember{
					{
						DBInstanceIdentifier:          adapterhelpers.PtrString("database-2-instance-3"), // link
						IsClusterWriter:               adapterhelpers.PtrBool(false),
						DBClusterParameterGroupStatus: adapterhelpers.PtrString("in-sync"),
						PromotionTier:                 adapterhelpers.PtrInt32(1),
					},
				},
				VpcSecurityGroups: []types.VpcSecurityGroupMembership{
					{
						VpcSecurityGroupId: adapterhelpers.PtrString("sg-094e151c9fc5da181"), // link
						Status:             adapterhelpers.PtrString("active"),
					},
				},
				HostedZoneId:                     adapterhelpers.PtrString("Z1TTGA775OQIYO"), // link
				StorageEncrypted:                 adapterhelpers.PtrBool(true),
				KmsKeyId:                         adapterhelpers.PtrString("arn:aws:kms:eu-west-2:052392120703:key/9653cbdd-1590-464a-8456-67389cef6933"), // link
				DbClusterResourceId:              adapterhelpers.PtrString("cluster-2EW4PDVN7F7V57CUJPYOEAA74M"),
				DBClusterArn:                     adapterhelpers.PtrString("arn:aws:rds:eu-west-2:052392120703:cluster:database-2"),
				IAMDatabaseAuthenticationEnabled: adapterhelpers.PtrBool(false),
				ClusterCreateTime:                adapterhelpers.PtrTime(time.Now()),
				EngineMode:                       adapterhelpers.PtrString("provisioned"),
				DeletionProtection:               adapterhelpers.PtrBool(false),
				HttpEndpointEnabled:              adapterhelpers.PtrBool(false),
				ActivityStreamStatus:             types.ActivityStreamStatusStopped,
				CopyTagsToSnapshot:               adapterhelpers.PtrBool(false),
				CrossAccountClone:                adapterhelpers.PtrBool(false),
				DomainMemberships:                []types.DomainMembership{},
				TagList:                          []types.Tag{},
				DBClusterInstanceClass:           adapterhelpers.PtrString("db.m5d.large"),
				StorageType:                      adapterhelpers.PtrString("io1"),
				Iops:                             adapterhelpers.PtrInt32(1000),
				PubliclyAccessible:               adapterhelpers.PtrBool(true),
				AutoMinorVersionUpgrade:          adapterhelpers.PtrBool(true),
				MonitoringInterval:               adapterhelpers.PtrInt32(0),
				PerformanceInsightsEnabled:       adapterhelpers.PtrBool(false),
				NetworkType:                      adapterhelpers.PtrString("IPV4"),
				ActivityStreamKinesisStreamName:  adapterhelpers.PtrString("aws-rds-das-db-AB1CDEFG23GHIJK4LMNOPQRST"), // link
				ActivityStreamKmsKeyId:           adapterhelpers.PtrString("ab12345e-1111-2bc3-12a3-ab1cd12345e"),      // Not linking at the moment because there are too many possible formats. If you want to change this, submit a PR
				ActivityStreamMode:               types.ActivityStreamModeAsync,
				AutomaticRestartTime:             adapterhelpers.PtrTime(time.Now()),
				AssociatedRoles:                  []types.DBClusterRole{}, // EC2 classic roles, ignore
				BacktrackConsumedChangeRecords:   adapterhelpers.PtrInt64(1),
				BacktrackWindow:                  adapterhelpers.PtrInt64(2),
				Capacity:                         adapterhelpers.PtrInt32(2),
				CharacterSetName:                 adapterhelpers.PtrString("english"),
				CloneGroupId:                     adapterhelpers.PtrString("id"),
				CustomEndpoints: []string{
					"endpoint1", // link dns
				},
				DBClusterOptionGroupMemberships: []types.DBClusterOptionGroupStatus{
					{
						DBClusterOptionGroupName: adapterhelpers.PtrString("optionGroupName"), // link
						Status:                   adapterhelpers.PtrString("good"),
					},
				},
				DBSystemId:            adapterhelpers.PtrString("systemId"),
				DatabaseName:          adapterhelpers.PtrString("databaseName"),
				EarliestBacktrackTime: adapterhelpers.PtrTime(time.Now()),
				EnabledCloudwatchLogsExports: []string{
					"logExport1",
				},
				GlobalWriteForwardingRequested: adapterhelpers.PtrBool(true),
				GlobalWriteForwardingStatus:    types.WriteForwardingStatusDisabled,
				MasterUserSecret: &types.MasterUserSecret{
					KmsKeyId:     adapterhelpers.PtrString("arn:aws:kms:eu-west-2:052392120703:key/something"), // link
					SecretArn:    adapterhelpers.PtrString("arn:aws:service:region:account:type/id"),           // link
					SecretStatus: adapterhelpers.PtrString("okay"),
				},
				MonitoringRoleArn:                  adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
				PendingModifiedValues:              &types.ClusterPendingModifiedValues{},
				PercentProgress:                    adapterhelpers.PtrString("99"),
				PerformanceInsightsKMSKeyId:        adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link, assuming it's an ARN
				PerformanceInsightsRetentionPeriod: adapterhelpers.PtrInt32(99),
				ReplicationSourceIdentifier:        adapterhelpers.PtrString("arn:aws:rds:eu-west-2:052392120703:cluster:database-1"), // link
				ScalingConfigurationInfo: &types.ScalingConfigurationInfo{
					AutoPause:             adapterhelpers.PtrBool(true),
					MaxCapacity:           adapterhelpers.PtrInt32(10),
					MinCapacity:           adapterhelpers.PtrInt32(1),
					SecondsBeforeTimeout:  adapterhelpers.PtrInt32(10),
					SecondsUntilAutoPause: adapterhelpers.PtrInt32(10),
					TimeoutAction:         adapterhelpers.PtrString("error"),
				},
				ServerlessV2ScalingConfiguration: &types.ServerlessV2ScalingConfigurationInfo{
					MaxCapacity: adapterhelpers.PtrFloat64(10),
					MinCapacity: adapterhelpers.PtrFloat64(1),
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

	tests := adapterhelpers.QueryTests{
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

	adapter := NewRDSDBClusterAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
