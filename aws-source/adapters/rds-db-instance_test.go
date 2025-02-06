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

func TestDBInstanceOutputMapper(t *testing.T) {
	output := &rds.DescribeDBInstancesOutput{
		DBInstances: []types.DBInstance{
			{
				DBInstanceIdentifier: adapterhelpers.PtrString("database-1-instance-1"),
				DBInstanceClass:      adapterhelpers.PtrString("db.r6g.large"),
				Engine:               adapterhelpers.PtrString("aurora-mysql"),
				DBInstanceStatus:     adapterhelpers.PtrString("available"),
				MasterUsername:       adapterhelpers.PtrString("admin"),
				Endpoint: &types.Endpoint{
					Address:      adapterhelpers.PtrString("database-1-instance-1.camcztjohmlj.eu-west-2.rds.amazonaws.com"), // link
					Port:         adapterhelpers.PtrInt32(3306),                                                              // link
					HostedZoneId: adapterhelpers.PtrString("Z1TTGA775OQIYO"),                                                 // link
				},
				AllocatedStorage:      adapterhelpers.PtrInt32(1),
				InstanceCreateTime:    adapterhelpers.PtrTime(time.Now()),
				PreferredBackupWindow: adapterhelpers.PtrString("00:05-00:35"),
				BackupRetentionPeriod: adapterhelpers.PtrInt32(1),
				DBSecurityGroups: []types.DBSecurityGroupMembership{
					{
						DBSecurityGroupName: adapterhelpers.PtrString("name"), // This is EC2Classic only so we're skipping this
					},
				},
				VpcSecurityGroups: []types.VpcSecurityGroupMembership{
					{
						VpcSecurityGroupId: adapterhelpers.PtrString("sg-094e151c9fc5da181"), // link
						Status:             adapterhelpers.PtrString("active"),
					},
				},
				DBParameterGroups: []types.DBParameterGroupStatus{
					{
						DBParameterGroupName: adapterhelpers.PtrString("default.aurora-mysql8.0"), // link
						ParameterApplyStatus: adapterhelpers.PtrString("in-sync"),
					},
				},
				AvailabilityZone: adapterhelpers.PtrString("eu-west-2a"), // link
				DBSubnetGroup: &types.DBSubnetGroup{
					DBSubnetGroupName:        adapterhelpers.PtrString("default-vpc-0d7892e00e573e701"), // link
					DBSubnetGroupDescription: adapterhelpers.PtrString("Created from the RDS Management Console"),
					VpcId:                    adapterhelpers.PtrString("vpc-0d7892e00e573e701"), // link
					SubnetGroupStatus:        adapterhelpers.PtrString("Complete"),
					Subnets: []types.Subnet{
						{
							SubnetIdentifier: adapterhelpers.PtrString("subnet-0d8ae4b4e07647efa"), // lnk
							SubnetAvailabilityZone: &types.AvailabilityZone{
								Name: adapterhelpers.PtrString("eu-west-2b"),
							},
							SubnetOutpost: &types.Outpost{
								Arn: adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
							},
							SubnetStatus: adapterhelpers.PtrString("Active"),
						},
					},
				},
				PreferredMaintenanceWindow: adapterhelpers.PtrString("fri:04:49-fri:05:19"),
				PendingModifiedValues:      &types.PendingModifiedValues{},
				MultiAZ:                    adapterhelpers.PtrBool(false),
				EngineVersion:              adapterhelpers.PtrString("8.0.mysql_aurora.3.02.0"),
				AutoMinorVersionUpgrade:    adapterhelpers.PtrBool(true),
				ReadReplicaDBInstanceIdentifiers: []string{
					"read",
				},
				LicenseModel: adapterhelpers.PtrString("general-public-license"),
				OptionGroupMemberships: []types.OptionGroupMembership{
					{
						OptionGroupName: adapterhelpers.PtrString("default:aurora-mysql-8-0"),
						Status:          adapterhelpers.PtrString("in-sync"),
					},
				},
				PubliclyAccessible:      adapterhelpers.PtrBool(false),
				StorageType:             adapterhelpers.PtrString("aurora"),
				DbInstancePort:          adapterhelpers.PtrInt32(0),
				DBClusterIdentifier:     adapterhelpers.PtrString("database-1"), // link
				StorageEncrypted:        adapterhelpers.PtrBool(true),
				KmsKeyId:                adapterhelpers.PtrString("arn:aws:kms:eu-west-2:052392120703:key/9653cbdd-1590-464a-8456-67389cef6933"), // link
				DbiResourceId:           adapterhelpers.PtrString("db-ET7CE5D5TQTK7MXNJGJNFQD52E"),
				CACertificateIdentifier: adapterhelpers.PtrString("rds-ca-2019"),
				DomainMemberships: []types.DomainMembership{
					{
						Domain:      adapterhelpers.PtrString("domain"),
						FQDN:        adapterhelpers.PtrString("fqdn"),
						IAMRoleName: adapterhelpers.PtrString("role"),
						Status:      adapterhelpers.PtrString("enrolled"),
					},
				},
				CopyTagsToSnapshot:                 adapterhelpers.PtrBool(false),
				MonitoringInterval:                 adapterhelpers.PtrInt32(60),
				EnhancedMonitoringResourceArn:      adapterhelpers.PtrString("arn:aws:logs:eu-west-2:052392120703:log-group:RDSOSMetrics:log-stream:db-ET7CE5D5TQTK7MXNJGJNFQD52E"), // link
				MonitoringRoleArn:                  adapterhelpers.PtrString("arn:aws:iam::052392120703:role/rds-monitoring-role"),                                                  // link
				PromotionTier:                      adapterhelpers.PtrInt32(1),
				DBInstanceArn:                      adapterhelpers.PtrString("arn:aws:rds:eu-west-2:052392120703:db:database-1-instance-1"),
				IAMDatabaseAuthenticationEnabled:   adapterhelpers.PtrBool(false),
				PerformanceInsightsEnabled:         adapterhelpers.PtrBool(true),
				PerformanceInsightsKMSKeyId:        adapterhelpers.PtrString("arn:aws:kms:eu-west-2:052392120703:key/9653cbdd-1590-464a-8456-67389cef6933"), // link
				PerformanceInsightsRetentionPeriod: adapterhelpers.PtrInt32(7),
				DeletionProtection:                 adapterhelpers.PtrBool(false),
				AssociatedRoles: []types.DBInstanceRole{
					{
						FeatureName: adapterhelpers.PtrString("something"),
						RoleArn:     adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
						Status:      adapterhelpers.PtrString("associated"),
					},
				},
				TagList:                []types.Tag{},
				CustomerOwnedIpEnabled: adapterhelpers.PtrBool(false),
				BackupTarget:           adapterhelpers.PtrString("region"),
				NetworkType:            adapterhelpers.PtrString("IPV4"),
				StorageThroughput:      adapterhelpers.PtrInt32(0),
				ActivityStreamEngineNativeAuditFieldsIncluded: adapterhelpers.PtrBool(true),
				ActivityStreamKinesisStreamName:               adapterhelpers.PtrString("aws-rds-das-db-AB1CDEFG23GHIJK4LMNOPQRST"), // link
				ActivityStreamKmsKeyId:                        adapterhelpers.PtrString("ab12345e-1111-2bc3-12a3-ab1cd12345e"),      // Not linking at the moment because there are too many possible formats. If you want to change this, submit a PR
				ActivityStreamMode:                            types.ActivityStreamModeAsync,
				ActivityStreamPolicyStatus:                    types.ActivityStreamPolicyStatusLocked,
				ActivityStreamStatus:                          types.ActivityStreamStatusStarted,
				AutomaticRestartTime:                          adapterhelpers.PtrTime(time.Now()),
				AutomationMode:                                types.AutomationModeAllPaused,
				AwsBackupRecoveryPointArn:                     adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
				CertificateDetails: &types.CertificateDetails{
					CAIdentifier: adapterhelpers.PtrString("id"),
					ValidTill:    adapterhelpers.PtrTime(time.Now()),
				},
				CharacterSetName:         adapterhelpers.PtrString("something"),
				CustomIamInstanceProfile: adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link?
				DBInstanceAutomatedBackupsReplications: []types.DBInstanceAutomatedBackupsReplication{
					{
						DBInstanceAutomatedBackupsArn: adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
					},
				},
				DBName:                       adapterhelpers.PtrString("name"),
				DBSystemId:                   adapterhelpers.PtrString("id"),
				EnabledCloudwatchLogsExports: []string{},
				Iops:                         adapterhelpers.PtrInt32(10),
				LatestRestorableTime:         adapterhelpers.PtrTime(time.Now()),
				ListenerEndpoint: &types.Endpoint{
					Address:      adapterhelpers.PtrString("foo.bar.com"), // link
					HostedZoneId: adapterhelpers.PtrString("id"),          // link
					Port:         adapterhelpers.PtrInt32(5432),           // link
				},
				MasterUserSecret: &types.MasterUserSecret{
					KmsKeyId:     adapterhelpers.PtrString("id"),                                     // link
					SecretArn:    adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // link
					SecretStatus: adapterhelpers.PtrString("okay"),
				},
				MaxAllocatedStorage:                   adapterhelpers.PtrInt32(10),
				NcharCharacterSetName:                 adapterhelpers.PtrString("english"),
				ProcessorFeatures:                     []types.ProcessorFeature{},
				ReadReplicaDBClusterIdentifiers:       []string{},
				ReadReplicaSourceDBInstanceIdentifier: adapterhelpers.PtrString("id"),
				ReplicaMode:                           types.ReplicaModeMounted,
				ResumeFullAutomationModeTime:          adapterhelpers.PtrTime(time.Now()),
				SecondaryAvailabilityZone:             adapterhelpers.PtrString("eu-west-1"), // link
				StatusInfos:                           []types.DBInstanceStatusInfo{},
				TdeCredentialArn:                      adapterhelpers.PtrString("arn:aws:service:region:account:type/id"), // I don't have a good example for this so skipping for now. PR if required
				Timezone:                              adapterhelpers.PtrString("GB"),
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

	tests := adapterhelpers.QueryTests{
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

	adapter := NewRDSDBInstanceAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
