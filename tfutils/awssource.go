// Code generated by "extractmaps aws-source"; DO NOT EDIT

package tfutils

import "github.com/overmindtech/sdp-go"

var AwssourceData = map[string][]TfMapData{
	"aws_alb_listener": {
		{
			Type:       "elbv2-listener",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_alb_listener_rule": {
		{
			Type:       "elbv2-rule",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_alb_target_group": {
		{
			Type:       "elbv2-target-group",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_ami": {
		{
			Type:       "ec2-image",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_autoscaling_group": {
		{
			Type:       "autoscaling-auto-scaling-group",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_cloudfront_Streamingdistribution": {
		{
			Type:       "cloudfront-streaming-distribution",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_cloudfront_cache_policy": {
		{
			Type:       "cloudfront-cache-policy",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_cloudfront_distribution": {
		{
			Type:       "cloudfront-distribution",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_cloudfront_function": {
		{
			Type:       "cloudfront-function",
			Method:     sdp.QueryMethod_GET,
			QueryField: "name",
			Scope:      "*",
		},
	},
	"aws_cloudfront_key_group": {
		{
			Type:       "cloudfront-key-group",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_cloudfront_origin_access_control": {
		{
			Type:       "cloudfront-origin-access-control",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_cloudfront_origin_request_policy": {
		{
			Type:       "cloudfront-origin-request-policy",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_cloudfront_realtime_log_config": {
		{
			Type:       "cloudfront-realtime-log-config",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_cloudfront_response_headers_policy": {
		{
			Type:       "cloudfront-response-headers-policy",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_cloudwatch_metric_alarm": {
		{
			Type:       "cloudwatch-alarm",
			Method:     sdp.QueryMethod_GET,
			QueryField: "alarm_name",
			Scope:      "*",
		},
	},
	"aws_db_instance": {
		{
			Type:       "rds-db-instance",
			Method:     sdp.QueryMethod_GET,
			QueryField: "identifier",
			Scope:      "*",
		},
	},
	"aws_db_instance_role_association": {
		{
			Type:       "rds-db-instance",
			Method:     sdp.QueryMethod_GET,
			QueryField: "db_instance_identifier",
			Scope:      "*",
		},
	},
	"aws_db_option_group": {
		{
			Type:       "rds-option-group",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_db_parameter_group": {
		{
			Type:       "rds-db-parameter-group",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_db_subnet_group": {
		{
			Type:       "rds-db-subnet-group",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_default_route_table": {
		{
			Type:       "ec2-route-table",
			Method:     sdp.QueryMethod_GET,
			QueryField: "default_route_table_id",
			Scope:      "*",
		},
	},
	"aws_dx_connection": {
		{
			Type:       "directconnect-connection",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_dx_gateway": {
		{
			Type:       "directconnect-direct-connect-gateway",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_dx_gateway_association": {
		{
			Type:       "directconnect-direct-connect-gateway-association",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_dx_gateway_association_proposal": {
		{
			Type:       "directconnect-direct-connect-gateway-association-proposal",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_dx_hosted_connection": {
		{
			Type:       "directconnect-hosted-connection",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_dx_lag": {
		{
			Type:       "directconnect-lag",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_dx_location": {
		{
			Type:       "directconnect-location",
			Method:     sdp.QueryMethod_GET,
			QueryField: "location_code",
			Scope:      "*",
		},
	},
	"aws_dx_private_virtual_interface": {
		{
			Type:       "directconnect-virtual-interface",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_dx_public_virtual_interface": {
		{
			Type:       "directconnect-virtual-interface",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_dx_router_configuration": {
		{
			Type:       "directconnect-router-configuration",
			Method:     sdp.QueryMethod_GET,
			QueryField: "virtual_interface_id",
			Scope:      "*",
		},
	},
	"aws_dx_transit_virtual_interface": {
		{
			Type:       "directconnect-virtual-interface",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_dynamodb_table": {
		{
			Type:       "dynamodb-table",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_ebs_volume": {
		{
			Type:       "ec2-volume",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_ec2_capacity_reservation": {
		{
			Type:       "ec2-capacity-reservation",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_ecs_capacity_provider": {
		{
			Type:       "ecs-capacity-provider",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_ecs_cluster": {
		{
			Type:       "ecs-cluster",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_ecs_service": {
		{
			Type:       "ecs-service",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "cluster_name",
			Scope:      "*",
		},
	},
	"aws_ecs_task_definition": {
		{
			Type:       "ecs-task-definition",
			Method:     sdp.QueryMethod_GET,
			QueryField: "family",
			Scope:      "*",
		},
	},
	"aws_efs_access_point": {
		{
			Type:       "efs-access-point",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_efs_backup_policy": {
		{
			Type:       "efs-backup-policy",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_efs_file_system": {
		{
			Type:       "efs-file-system",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_efs_mount_target": {
		{
			Type:       "efs-mount-target",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_efs_replication_configuration": {
		{
			Type:       "efs-replication-configuration",
			Method:     sdp.QueryMethod_GET,
			QueryField: "source_file_system_id",
			Scope:      "*",
		},
	},
	"aws_eip": {
		{
			Type:       "ec2-address",
			Method:     sdp.QueryMethod_GET,
			QueryField: "public_ip",
			Scope:      "*",
		},
	},
	"aws_eip_association": {
		{
			Type:       "ec2-address",
			Method:     sdp.QueryMethod_GET,
			QueryField: "public_ip",
			Scope:      "*",
		},
	},
	"aws_eks_addon": {
		{
			Type:       "eks-addon",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_eks_cluster": {
		{
			Type:       "eks-cluster",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_eks_fargate_profile": {
		{
			Type:       "eks-fargate-profile",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_eks_node_group": {
		{
			Type:       "eks-nodegroup",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_elb": {
		{
			Type:       "elb-load-balancer",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_iam_group": {
		{
			Type:       "iam-group",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_iam_instance_profile": {
		{
			Type:       "iam-instance-profile",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_iam_policy": {
		{
			Type:       "iam-policy",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_iam_role": {
		{
			Type:       "iam-role",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_iam_role_policy_attachment": {
		{
			Type:       "iam-policy",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "policy_arn",
			Scope:      "*",
		},
	},
	"aws_iam_user": {
		{
			Type:       "iam-user",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_iam_user_policy_attachment": {
		{
			Type:       "iam-policy",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "policy_arn",
			Scope:      "*",
		},
	},
	"aws_instance": {
		{
			Type:       "ec2-instance",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_internet_gateway": {
		{
			Type:       "ec2-internet-gateway",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_key_pair": {
		{
			Type:       "ec2-key-pair",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_lambda_function": {
		{
			Type:       "lambda-function",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_lambda_function_event_invoke_config": {
		{
			Type:       "lambda-function",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_lambda_function_url": {
		{
			Type:       "lambda-function",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "function_arn",
			Scope:      "*",
		},
	},
	"aws_lambda_layer_version": {
		{
			Type:       "lambda-layer-version",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_launch_template": {
		{
			Type:       "ec2-launch-template",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_lb": {
		{
			Type:       "elbv2-load-balancer",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_lb_listener": {
		{
			Type:       "elbv2-listener",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_lb_listener_rule": {
		{
			Type:       "elbv2-rule",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_lb_target_group": {
		{
			Type:       "elbv2-target-group",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_nat_gateway": {
		{
			Type:       "ec2-nat-gateway",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_network_acl": {
		{
			Type:       "ec2-network-acl",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_network_interface": {
		{
			Type:       "ec2-network-interface",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_networkfirewall_firewall": {
		{
			Type:       "network-firewall-firewall",
			Method:     sdp.QueryMethod_GET,
			QueryField: "name",
			Scope:      "*",
		},
	},
	"aws_networkfirewall_firewall_policy": {
		{
			Type:       "network-firewall-firewall-policy",
			Method:     sdp.QueryMethod_GET,
			QueryField: "name",
			Scope:      "*",
		},
	},
	"aws_networkfirewall_rule_group": {
		{
			Type:       "network-firewall-rule-group",
			Method:     sdp.QueryMethod_GET,
			QueryField: "name",
			Scope:      "*",
		},
	},
	"aws_networkmanager_connect_peer": {
		{
			Type:       "networkmanager-connect-peer",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_networkmanager_connection": {
		{
			Type:       "networkmanager-connection",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_networkmanager_core_network": {
		{
			Type:       "networkmanager-connect-attachment",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
		{
			Type:       "networkmanager-core-network",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_networkmanager_core_network_policy": {
		{
			Type:       "networkmanager-core-network-policy",
			Method:     sdp.QueryMethod_GET,
			QueryField: "core_network_id",
			Scope:      "*",
		},
	},
	"aws_networkmanager_device": {
		{
			Type:       "networkmanager-device",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_networkmanager_global_network": {
		{
			Type:       "networkmanager-global-network",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_networkmanager_link": {
		{
			Type:       "networkmanager-link",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_networkmanager_site": {
		{
			Type:       "networkmanager-site",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_networkmanager_site_to_site_vpn_attachment": {
		{
			Type:       "networkmanager-site-to-site-vpn-attachment",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_networkmanager_transit_gateway_peering": {
		{
			Type:       "networkmanager-transit-gateway-peering",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_networkmanager_transit_gateway_route_table_attachment": {
		{
			Type:       "networkmanager-transit-gateway-route-table-attachment",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_networkmanager_vpc_attachment": {
		{
			Type:       "networkmanager-vpc-attachment",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_placement_group": {
		{
			Type:       "ec2-placement-group",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_rds_cluster": {
		{
			Type:       "rds-db-cluster",
			Method:     sdp.QueryMethod_GET,
			QueryField: "cluster_identifier",
			Scope:      "*",
		},
	},
	"aws_rds_cluster_parameter_group": {
		{
			Type:       "rds-db-cluster-parameter-group",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_route": {
		{
			Type:       "ec2-route-table",
			Method:     sdp.QueryMethod_GET,
			QueryField: "route_table_id",
			Scope:      "*",
		},
	},
	"aws_route53_health_check": {
		{
			Type:       "route53-health-check",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_route53_hosted_zone_dnssec": {
		{
			Type:       "route53-hosted-zone",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_route53_record": {
		{
			Type:       "route53-resource-record-set",
			Method:     sdp.QueryMethod_SEARCH,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_route53_zone": {
		{
			Type:       "route53-hosted-zone",
			Method:     sdp.QueryMethod_GET,
			QueryField: "zone_id",
			Scope:      "*",
		},
	},
	"aws_route53_zone_association": {
		{
			Type:       "route53-hosted-zone",
			Method:     sdp.QueryMethod_GET,
			QueryField: "zone_id",
			Scope:      "*",
		},
	},
	"aws_route_table": {
		{
			Type:       "ec2-route-table",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_route_table_association": {
		{
			Type:       "ec2-route-table",
			Method:     sdp.QueryMethod_GET,
			QueryField: "route_table_id",
			Scope:      "*",
		},
		{
			Type:       "ec2-subnet",
			Method:     sdp.QueryMethod_GET,
			QueryField: "subnet_id",
			Scope:      "*",
		},
	},
	"aws_s3_bucket": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_acl": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_analytics_configuration": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_cors_configuration": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_intelligent_tiering_configuration": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_inventory": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_lifecycle_configuration": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_logging": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_metric": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_notification": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_object": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_object_lock_configuration": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_ownership_controls": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_policy": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_public_access_block": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_replication_configuration": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_request_payment_configuration": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_server_side_encryption_configuration": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_versioning": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_bucket_website_configuration": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_object": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_s3_object_copy": {
		{
			Type:       "s3-bucket",
			Method:     sdp.QueryMethod_GET,
			QueryField: "bucket",
			Scope:      "*",
		},
	},
	"aws_security_group": {
		{
			Type:       "ec2-security-group",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_security_group_rule": {
		{
			Type:       "ec2-security-group-rule",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
		{
			Type:       "ec2-security-group-rule",
			Method:     sdp.QueryMethod_GET,
			QueryField: "security_group_rule_id",
			Scope:      "*",
		},
		{
			Type:       "ec2-security-group",
			Method:     sdp.QueryMethod_GET,
			QueryField: "security_group_id",
			Scope:      "*",
		},
	},
	"aws_sns_platform_application": {
		{
			Type:       "sns-platform-application",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_sns_topic": {
		{
			Type:       "sns-topic",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_sns_topic_data_protection_policy": {
		{
			Type:       "sns-data-protection-policy",
			Method:     sdp.QueryMethod_GET,
			QueryField: "arn",
			Scope:      "*",
		},
	},
	"aws_sns_topic_subscription": {
		{
			Type:       "sns-subscription",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_sqs_queue": {
		{
			Type:       "sqs-queue",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_subnet": {
		{
			Type:       "ec2-subnet",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_vpc": {
		{
			Type:       "ec2-vpc",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_vpc_endpoint": {
		{
			Type:       "ec2-vpc-endpoint",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_vpc_peering_connection": {
		{
			Type:       "ec2-vpc-peering-connection",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_vpc_peering_connection_accepter": {
		{
			Type:       "ec2-vpc-peering-connection",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
	"aws_vpc_peering_connection_options": {
		{
			Type:       "ec2-vpc-peering-connection",
			Method:     sdp.QueryMethod_GET,
			QueryField: "vpc_peering_connection_id",
			Scope:      "*",
		},
	},
	"egress_only_internet_gateway": {
		{
			Type:       "ec2-egress-only-internet-gateway",
			Method:     sdp.QueryMethod_GET,
			QueryField: "id",
			Scope:      "*",
		},
	},
}