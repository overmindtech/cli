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

func TestDBInstanceOutputMapper(t *testing.T) {
	output := &rds.DescribeDBInstancesOutput{
		DBInstances: []types.DBInstance{
			{
				DBInstanceIdentifier: new("database-1-instance-1"),
				DBInstanceClass:      new("db.r6g.large"),
				Engine:               new("aurora-mysql"),
				DBInstanceStatus:     new("available"),
				MasterUsername:       new("admin"),
				Endpoint: &types.Endpoint{
					Address:      new("database-1-instance-1.camcztjohmlj.eu-west-2.rds.amazonaws.com"), // link
					Port:         new(int32(3306)),                                                      // link
					HostedZoneId: new("Z1TTGA775OQIYO"),                                                 // link
				},
				AllocatedStorage:      new(int32(1)),
				InstanceCreateTime:    new(time.Now()),
				PreferredBackupWindow: new("00:05-00:35"),
				BackupRetentionPeriod: new(int32(1)),
				DBSecurityGroups: []types.DBSecurityGroupMembership{
					{
						DBSecurityGroupName: new("name"), // This is EC2Classic only so we're skipping this
					},
				},
				VpcSecurityGroups: []types.VpcSecurityGroupMembership{
					{
						VpcSecurityGroupId: new("sg-094e151c9fc5da181"), // link
						Status:             new("active"),
					},
				},
				DBParameterGroups: []types.DBParameterGroupStatus{
					{
						DBParameterGroupName: new("default.aurora-mysql8.0"), // link
						ParameterApplyStatus: new("in-sync"),
					},
				},
				AvailabilityZone: new("eu-west-2a"), // link
				DBSubnetGroup: &types.DBSubnetGroup{
					DBSubnetGroupName:        new("default-vpc-0d7892e00e573e701"), // link
					DBSubnetGroupDescription: new("Created from the RDS Management Console"),
					VpcId:                    new("vpc-0d7892e00e573e701"), // link
					SubnetGroupStatus:        new("Complete"),
					Subnets: []types.Subnet{
						{
							SubnetIdentifier: new("subnet-0d8ae4b4e07647efa"), // lnk
							SubnetAvailabilityZone: &types.AvailabilityZone{
								Name: new("eu-west-2b"),
							},
							SubnetOutpost: &types.Outpost{
								Arn: new("arn:aws:service:region:account:type/id"), // link
							},
							SubnetStatus: new("Active"),
						},
					},
				},
				PreferredMaintenanceWindow: new("fri:04:49-fri:05:19"),
				PendingModifiedValues:      &types.PendingModifiedValues{},
				MultiAZ:                    new(false),
				EngineVersion:              new("8.0.mysql_aurora.3.02.0"),
				AutoMinorVersionUpgrade:    new(true),
				ReadReplicaDBInstanceIdentifiers: []string{
					"read",
				},
				LicenseModel: new("general-public-license"),
				OptionGroupMemberships: []types.OptionGroupMembership{
					{
						OptionGroupName: new("default:aurora-mysql-8-0"),
						Status:          new("in-sync"),
					},
				},
				PubliclyAccessible:      new(false),
				StorageType:             new("aurora"),
				DbInstancePort:          new(int32(0)),
				DBClusterIdentifier:     new("database-1"), // link
				StorageEncrypted:        new(true),
				KmsKeyId:                new("arn:aws:kms:eu-west-2:052392120703:key/9653cbdd-1590-464a-8456-67389cef6933"), // link
				DbiResourceId:           new("db-ET7CE5D5TQTK7MXNJGJNFQD52E"),
				CACertificateIdentifier: new("rds-ca-2019"),
				DomainMemberships: []types.DomainMembership{
					{
						Domain:      new("domain"),
						FQDN:        new("fqdn"),
						IAMRoleName: new("role"),
						Status:      new("enrolled"),
					},
				},
				CopyTagsToSnapshot:                 new(false),
				MonitoringInterval:                 new(int32(60)),
				EnhancedMonitoringResourceArn:      new("arn:aws:logs:eu-west-2:052392120703:log-group:RDSOSMetrics:log-stream:db-ET7CE5D5TQTK7MXNJGJNFQD52E"), // link
				MonitoringRoleArn:                  new("arn:aws:iam::052392120703:role/rds-monitoring-role"),                                                  // link
				PromotionTier:                      new(int32(1)),
				DBInstanceArn:                      new("arn:aws:rds:eu-west-2:052392120703:db:database-1-instance-1"),
				IAMDatabaseAuthenticationEnabled:   new(false),
				PerformanceInsightsEnabled:         new(true),
				PerformanceInsightsKMSKeyId:        new("arn:aws:kms:eu-west-2:052392120703:key/9653cbdd-1590-464a-8456-67389cef6933"), // link
				PerformanceInsightsRetentionPeriod: new(int32(7)),
				DeletionProtection:                 new(false),
				AssociatedRoles: []types.DBInstanceRole{
					{
						FeatureName: new("something"),
						RoleArn:     new("arn:aws:service:region:account:type/id"), // link
						Status:      new("associated"),
					},
				},
				TagList:                []types.Tag{},
				CustomerOwnedIpEnabled: new(false),
				BackupTarget:           new("region"),
				NetworkType:            new("IPV4"),
				StorageThroughput:      new(int32(0)),
				ActivityStreamEngineNativeAuditFieldsIncluded: new(true),
				ActivityStreamKinesisStreamName:               new("aws-rds-das-db-AB1CDEFG23GHIJK4LMNOPQRST"), // link
				ActivityStreamKmsKeyId:                        new("ab12345e-1111-2bc3-12a3-ab1cd12345e"),      // Not linking at the moment because there are too many possible formats. If you want to change this, submit a PR
				ActivityStreamMode:                            types.ActivityStreamModeAsync,
				ActivityStreamPolicyStatus:                    types.ActivityStreamPolicyStatusLocked,
				ActivityStreamStatus:                          types.ActivityStreamStatusStarted,
				AutomaticRestartTime:                          new(time.Now()),
				AutomationMode:                                types.AutomationModeAllPaused,
				AwsBackupRecoveryPointArn:                     new("arn:aws:service:region:account:type/id"), // link
				CertificateDetails: &types.CertificateDetails{
					CAIdentifier: new("id"),
					ValidTill:    new(time.Now()),
				},
				CharacterSetName:         new("something"),
				CustomIamInstanceProfile: new("arn:aws:service:region:account:type/id"), // link?
				DBInstanceAutomatedBackupsReplications: []types.DBInstanceAutomatedBackupsReplication{
					{
						DBInstanceAutomatedBackupsArn: new("arn:aws:service:region:account:type/id"), // link
					},
				},
				DBName:                       new("name"),
				DBSystemId:                   new("id"),
				EnabledCloudwatchLogsExports: []string{},
				Iops:                         new(int32(10)),
				LatestRestorableTime:         new(time.Now()),
				ListenerEndpoint: &types.Endpoint{
					Address:      new("foo.bar.com"), // link
					HostedZoneId: new("id"),          // link
					Port:         new(int32(5432)),   // link
				},
				MasterUserSecret: &types.MasterUserSecret{
					KmsKeyId:     new("id"),                                     // link
					SecretArn:    new("arn:aws:service:region:account:type/id"), // link
					SecretStatus: new("okay"),
				},
				MaxAllocatedStorage:                   new(int32(10)),
				NcharCharacterSetName:                 new("english"),
				ProcessorFeatures:                     []types.ProcessorFeature{},
				ReadReplicaDBClusterIdentifiers:       []string{},
				ReadReplicaSourceDBInstanceIdentifier: new("id"),
				ReplicaMode:                           types.ReplicaModeMounted,
				ResumeFullAutomationModeTime:          new(time.Now()),
				SecondaryAvailabilityZone:             new("eu-west-1"), // link
				StatusInfos:                           []types.DBInstanceStatusInfo{},
				TdeCredentialArn:                      new("arn:aws:service:region:account:type/id"), // I don't have a good example for this so skipping for now. PR if required
				Timezone:                              new("GB"),
			},
		},
	}

	items, err := dBInstanceOutputMapper(context.Background(), mockRdsClient{}, "foo", nil, output)

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
		t.Errorf("got %v, expected %v", item.GetTags()["key"], "value")
	}

	tests := QueryTests{
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "database-1-instance-1.camcztjohmlj.eu-west-2.rds.amazonaws.com",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "route53-hosted-zone",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "Z1TTGA775OQIYO",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "ec2-security-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "sg-094e151c9fc5da181",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "rds-db-parameter-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default.aurora-mysql8.0",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "rds-db-subnet-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "default-vpc-0d7892e00e573e701",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "rds-db-cluster",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "database-1",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:kms:eu-west-2:052392120703:key/9653cbdd-1590-464a-8456-67389cef6933",
			ExpectedScope:  "052392120703.eu-west-2",
		},
		{
			ExpectedType:   "logs-log-stream",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:logs:eu-west-2:052392120703:log-group:RDSOSMetrics:log-stream:db-ET7CE5D5TQTK7MXNJGJNFQD52E",
			ExpectedScope:  "052392120703.eu-west-2",
		},
		{
			ExpectedType:   "iam-role",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:iam::052392120703:role/rds-monitoring-role",
			ExpectedScope:  "052392120703",
		},
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:kms:eu-west-2:052392120703:key/9653cbdd-1590-464a-8456-67389cef6933",
			ExpectedScope:  "052392120703.eu-west-2",
		},
		{
			ExpectedType:   "iam-role",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "kinesis-stream",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "aws-rds-das-db-AB1CDEFG23GHIJK4LMNOPQRST",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "backup-recovery-point",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "iam-instance-profile",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "rds-db-instance-automated-backup",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "foo.bar.com",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "route53-hosted-zone",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "secretsmanager-secret",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:service:region:account:type/id",
			ExpectedScope:  "account.region",
		},
	}

	tests.Execute(t, item)
}

func TestNewRDSDBInstanceAdapter(t *testing.T) {
	client, account, region := rdsGetAutoConfig(t)

	adapter := NewRDSDBInstanceAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
