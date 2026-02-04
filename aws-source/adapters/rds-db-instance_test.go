package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

func TestDBInstanceOutputMapper(t *testing.T) {
	output := &rds.DescribeDBInstancesOutput{
		DBInstances: []types.DBInstance{
			{
				DBInstanceIdentifier: PtrString("database-1-instance-1"),
				DBInstanceClass:      PtrString("db.r6g.large"),
				Engine:               PtrString("aurora-mysql"),
				DBInstanceStatus:     PtrString("available"),
				MasterUsername:       PtrString("admin"),
				Endpoint: &types.Endpoint{
					Address:      PtrString("database-1-instance-1.camcztjohmlj.eu-west-2.rds.amazonaws.com"), // link
					Port:         PtrInt32(3306),                                                              // link
					HostedZoneId: PtrString("Z1TTGA775OQIYO"),                                                 // link
				},
				AllocatedStorage:      PtrInt32(1),
				InstanceCreateTime:    PtrTime(time.Now()),
				PreferredBackupWindow: PtrString("00:05-00:35"),
				BackupRetentionPeriod: PtrInt32(1),
				DBSecurityGroups: []types.DBSecurityGroupMembership{
					{
						DBSecurityGroupName: PtrString("name"), // This is EC2Classic only so we're skipping this
					},
				},
				VpcSecurityGroups: []types.VpcSecurityGroupMembership{
					{
						VpcSecurityGroupId: PtrString("sg-094e151c9fc5da181"), // link
						Status:             PtrString("active"),
					},
				},
				DBParameterGroups: []types.DBParameterGroupStatus{
					{
						DBParameterGroupName: PtrString("default.aurora-mysql8.0"), // link
						ParameterApplyStatus: PtrString("in-sync"),
					},
				},
				AvailabilityZone: PtrString("eu-west-2a"), // link
				DBSubnetGroup: &types.DBSubnetGroup{
					DBSubnetGroupName:        PtrString("default-vpc-0d7892e00e573e701"), // link
					DBSubnetGroupDescription: PtrString("Created from the RDS Management Console"),
					VpcId:                    PtrString("vpc-0d7892e00e573e701"), // link
					SubnetGroupStatus:        PtrString("Complete"),
					Subnets: []types.Subnet{
						{
							SubnetIdentifier: PtrString("subnet-0d8ae4b4e07647efa"), // lnk
							SubnetAvailabilityZone: &types.AvailabilityZone{
								Name: PtrString("eu-west-2b"),
							},
							SubnetOutpost: &types.Outpost{
								Arn: PtrString("arn:aws:service:region:account:type/id"), // link
							},
							SubnetStatus: PtrString("Active"),
						},
					},
				},
				PreferredMaintenanceWindow: PtrString("fri:04:49-fri:05:19"),
				PendingModifiedValues:      &types.PendingModifiedValues{},
				MultiAZ:                    PtrBool(false),
				EngineVersion:              PtrString("8.0.mysql_aurora.3.02.0"),
				AutoMinorVersionUpgrade:    PtrBool(true),
				ReadReplicaDBInstanceIdentifiers: []string{
					"read",
				},
				LicenseModel: PtrString("general-public-license"),
				OptionGroupMemberships: []types.OptionGroupMembership{
					{
						OptionGroupName: PtrString("default:aurora-mysql-8-0"),
						Status:          PtrString("in-sync"),
					},
				},
				PubliclyAccessible:      PtrBool(false),
				StorageType:             PtrString("aurora"),
				DbInstancePort:          PtrInt32(0),
				DBClusterIdentifier:     PtrString("database-1"), // link
				StorageEncrypted:        PtrBool(true),
				KmsKeyId:                PtrString("arn:aws:kms:eu-west-2:052392120703:key/9653cbdd-1590-464a-8456-67389cef6933"), // link
				DbiResourceId:           PtrString("db-ET7CE5D5TQTK7MXNJGJNFQD52E"),
				CACertificateIdentifier: PtrString("rds-ca-2019"),
				DomainMemberships: []types.DomainMembership{
					{
						Domain:      PtrString("domain"),
						FQDN:        PtrString("fqdn"),
						IAMRoleName: PtrString("role"),
						Status:      PtrString("enrolled"),
					},
				},
				CopyTagsToSnapshot:                 PtrBool(false),
				MonitoringInterval:                 PtrInt32(60),
				EnhancedMonitoringResourceArn:      PtrString("arn:aws:logs:eu-west-2:052392120703:log-group:RDSOSMetrics:log-stream:db-ET7CE5D5TQTK7MXNJGJNFQD52E"), // link
				MonitoringRoleArn:                  PtrString("arn:aws:iam::052392120703:role/rds-monitoring-role"),                                                  // link
				PromotionTier:                      PtrInt32(1),
				DBInstanceArn:                      PtrString("arn:aws:rds:eu-west-2:052392120703:db:database-1-instance-1"),
				IAMDatabaseAuthenticationEnabled:   PtrBool(false),
				PerformanceInsightsEnabled:         PtrBool(true),
				PerformanceInsightsKMSKeyId:        PtrString("arn:aws:kms:eu-west-2:052392120703:key/9653cbdd-1590-464a-8456-67389cef6933"), // link
				PerformanceInsightsRetentionPeriod: PtrInt32(7),
				DeletionProtection:                 PtrBool(false),
				AssociatedRoles: []types.DBInstanceRole{
					{
						FeatureName: PtrString("something"),
						RoleArn:     PtrString("arn:aws:service:region:account:type/id"), // link
						Status:      PtrString("associated"),
					},
				},
				TagList:                []types.Tag{},
				CustomerOwnedIpEnabled: PtrBool(false),
				BackupTarget:           PtrString("region"),
				NetworkType:            PtrString("IPV4"),
				StorageThroughput:      PtrInt32(0),
				ActivityStreamEngineNativeAuditFieldsIncluded: PtrBool(true),
				ActivityStreamKinesisStreamName:               PtrString("aws-rds-das-db-AB1CDEFG23GHIJK4LMNOPQRST"), // link
				ActivityStreamKmsKeyId:                        PtrString("ab12345e-1111-2bc3-12a3-ab1cd12345e"),      // Not linking at the moment because there are too many possible formats. If you want to change this, submit a PR
				ActivityStreamMode:                            types.ActivityStreamModeAsync,
				ActivityStreamPolicyStatus:                    types.ActivityStreamPolicyStatusLocked,
				ActivityStreamStatus:                          types.ActivityStreamStatusStarted,
				AutomaticRestartTime:                          PtrTime(time.Now()),
				AutomationMode:                                types.AutomationModeAllPaused,
				AwsBackupRecoveryPointArn:                     PtrString("arn:aws:service:region:account:type/id"), // link
				CertificateDetails: &types.CertificateDetails{
					CAIdentifier: PtrString("id"),
					ValidTill:    PtrTime(time.Now()),
				},
				CharacterSetName:         PtrString("something"),
				CustomIamInstanceProfile: PtrString("arn:aws:service:region:account:type/id"), // link?
				DBInstanceAutomatedBackupsReplications: []types.DBInstanceAutomatedBackupsReplication{
					{
						DBInstanceAutomatedBackupsArn: PtrString("arn:aws:service:region:account:type/id"), // link
					},
				},
				DBName:                       PtrString("name"),
				DBSystemId:                   PtrString("id"),
				EnabledCloudwatchLogsExports: []string{},
				Iops:                         PtrInt32(10),
				LatestRestorableTime:         PtrTime(time.Now()),
				ListenerEndpoint: &types.Endpoint{
					Address:      PtrString("foo.bar.com"), // link
					HostedZoneId: PtrString("id"),          // link
					Port:         PtrInt32(5432),           // link
				},
				MasterUserSecret: &types.MasterUserSecret{
					KmsKeyId:     PtrString("id"),                                     // link
					SecretArn:    PtrString("arn:aws:service:region:account:type/id"), // link
					SecretStatus: PtrString("okay"),
				},
				MaxAllocatedStorage:                   PtrInt32(10),
				NcharCharacterSetName:                 PtrString("english"),
				ProcessorFeatures:                     []types.ProcessorFeature{},
				ReadReplicaDBClusterIdentifiers:       []string{},
				ReadReplicaSourceDBInstanceIdentifier: PtrString("id"),
				ReplicaMode:                           types.ReplicaModeMounted,
				ResumeFullAutomationModeTime:          PtrTime(time.Now()),
				SecondaryAvailabilityZone:             PtrString("eu-west-1"), // link
				StatusInfos:                           []types.DBInstanceStatusInfo{},
				TdeCredentialArn:                      PtrString("arn:aws:service:region:account:type/id"), // I don't have a good example for this so skipping for now. PR if required
				Timezone:                              PtrString("GB"),
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
