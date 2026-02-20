output "role_arn" {
  description = "ARN of the created IAM role."
  value       = aws_iam_role.overmind.arn
}

output "source_id" {
  description = "UUID of the Overmind source."
  value       = overmind_aws_source.this.id
}

output "external_id" {
  description = "AWS STS external ID used in the trust policy."
  value       = data.overmind_aws_external_id.this.external_id
}
