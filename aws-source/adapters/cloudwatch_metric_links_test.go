package adapters

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func TestSuggestedQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Name          string
		Namespace     string
		Dimensions    []types.Dimension
		ExpectedType  string
		ExpectedQuery string
	}{
		{
			Name:      "AWS/EC2 Instance",
			Namespace: "AWS/EC2",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("InstanceId"),
					Value: aws.String("i-1234567890abcdef0"),
				},
			},
			ExpectedType:  "ec2-instance",
			ExpectedQuery: "i-1234567890abcdef0",
		},
		{
			Name:      "AWS/EC2 AutoScalingGroup",
			Namespace: "AWS/EC2",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("AutoScalingGroupName"),
					Value: aws.String("my-asg"),
				},
			},
			ExpectedType:  "autoscaling-auto-scaling-group",
			ExpectedQuery: "my-asg",
		},
		{
			Name:      "AWS/EC2 Image",
			Namespace: "AWS/EC2",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("ImageId"),
					Value: aws.String("ami-1234567890abcdef0"),
				},
			},
			ExpectedType:  "ec2-image",
			ExpectedQuery: "ami-1234567890abcdef0",
		},
		{
			Name:      "AWS/ApplicationELB with multiple dimensions",
			Namespace: "AWS/ApplicationELB",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("TargetGroup"),
					Value: aws.String("targetgroup/k8s-default-smartloo-d63873991a/98720d5dcd06067a"),
				},
				{
					Name:  aws.String("LoadBalancer"),
					Value: aws.String("app/ingress/1bf10920c5bd199d"),
				},
			},
			ExpectedType:  "elbv2-target-group",
			ExpectedQuery: "k8s-default-smartloo-d63873991a",
		},
		{
			Name:      "AWS/ApplicationELB with one dimension",
			Namespace: "AWS/ApplicationELB",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("LoadBalancer"),
					Value: aws.String("app/ingress/1bf10920c5bd199d"),
				},
			},
			ExpectedType:  "elbv2-load-balancer",
			ExpectedQuery: "ingress",
		},
		{
			Name:      "Backup",
			Namespace: "AWS/Backup",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("BackupVaultName"),
					Value: aws.String("aws/efs/automatic-backup-vault"),
				},
			},
			ExpectedType:  "backup-backup-vault",
			ExpectedQuery: "aws/efs/automatic-backup-vault",
		},
		{
			Name:      "Certificate",
			Namespace: "AWS/CertificateManager",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("CertificateArn"),
					Value: aws.String("arn:aws:acm:eu-west-2:944651592624:certificate/3092dd18-f6cd-4ae7-b129-9023904bb7d0"),
				},
			},
			ExpectedType:  "acm-certificate",
			ExpectedQuery: "arn:aws:acm:eu-west-2:944651592624:certificate/3092dd18-f6cd-4ae7-b129-9023904bb7d0",
		},
		{
			Name:      "EBS Volume",
			Namespace: "AWS/EBS",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("VolumeId"),
					Value: aws.String("vol-1234567890abcdef0"),
				},
			},
			ExpectedType:  "ec2-volume",
			ExpectedQuery: "vol-1234567890abcdef0",
		},
		{
			Name:      "EBS Filesystem",
			Namespace: "AWS/EFS",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("FileSystemId"),
					Value: aws.String("fs-12345678"),
				},
			},
			ExpectedType:  "efs-file-system",
			ExpectedQuery: "fs-12345678",
		},
		{
			Name:      "RDS Cluster",
			Namespace: "AWS/RDS",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("DBClusterIdentifier"),
					Value: aws.String("my-cluster"),
				},
			},
			ExpectedType:  "rds-db-cluster",
			ExpectedQuery: "my-cluster",
		},
		{
			Name:      "RDS DB Instance",
			Namespace: "AWS/RDS",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("DBInstanceIdentifier"),
					Value: aws.String("my-instance"),
				},
			},
			ExpectedType:  "rds-db-instance",
			ExpectedQuery: "my-instance",
		},
		{
			Name:      "RDS with cluster and instance",
			Namespace: "AWS/RDS",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("DBClusterIdentifier"),
					Value: aws.String("my-cluster"),
				},
				{
					Name:  aws.String("DBInstanceIdentifier"),
					Value: aws.String("my-instance"),
				},
			},
			ExpectedType:  "rds-db-instance",
			ExpectedQuery: "my-instance",
		},
		{
			Name:      "S3 Bucket",
			Namespace: "AWS/S3",
			Dimensions: []types.Dimension{
				{
					Name:  aws.String("BucketName"),
					Value: aws.String("my-bucket"),
				},
			},
			ExpectedType:  "s3-bucket",
			ExpectedQuery: "my-bucket",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			scope := "123456789012.eu-west-2"
			query, err := SuggestedQuery(c.Namespace, scope, c.Dimensions)

			if err != nil {
				t.Fatal(err)
			}

			if query.GetQuery().GetType() != c.ExpectedType {
				t.Fatalf("expected type %q, got %q", c.ExpectedType, query.GetQuery().GetType())
			}

			if query.GetQuery().GetQuery() != c.ExpectedQuery {
				t.Fatalf("expected query %q, got %q", c.ExpectedQuery, query.GetQuery().GetQuery())
			}
		})
	}
}
