# =============================================================================
# Outputs — Connection strings and cluster info for CI/CD and kubectl
# =============================================================================

output "vpc_id" {
  description = "VPC ID."
  value       = aws_vpc.main.id
}

output "eks_cluster_name" {
  description = "EKS cluster name."
  value       = aws_eks_cluster.main.name
}

output "eks_cluster_endpoint" {
  description = "EKS cluster API endpoint."
  value       = aws_eks_cluster.main.endpoint
}

output "eks_cluster_ca_certificate" {
  description = "EKS cluster CA certificate (base64)."
  value       = aws_eks_cluster.main.certificate_authority[0].data
  sensitive   = true
}

output "rds_endpoint" {
  description = "RDS PostgreSQL endpoint."
  value       = aws_db_instance.main.endpoint
}

output "rds_connection_string" {
  description = "Full PostgreSQL connection string."
  value       = "postgres://${var.db_username}:${var.db_password}@${aws_db_instance.main.endpoint}/${var.db_name}?sslmode=require"
  sensitive   = true
}

output "redis_endpoint" {
  description = "ElastiCache Redis endpoint."
  value       = aws_elasticache_cluster.main.cache_nodes[0].address
}

output "rabbitmq_endpoint" {
  description = "Amazon MQ RabbitMQ AMQPS endpoint."
  value       = aws_mq_broker.main.instances[0].endpoints[0]
}

output "ecr_repository_urls" {
  description = "ECR repository URLs for Docker push."
  value       = { for k, v in aws_ecr_repository.main : k => v.repository_url }
}

# Helper command to configure kubectl.
output "kubectl_config_command" {
  description = "Run this command to configure kubectl for the cluster."
  value       = "aws eks update-kubeconfig --name ${aws_eks_cluster.main.name} --region ${var.aws_region}"
}
