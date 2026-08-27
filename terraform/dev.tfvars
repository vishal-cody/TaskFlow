# =============================================================================
# Dev Environment Variable Overrides
# =============================================================================
# Usage: terraform plan -var-file=dev.tfvars

aws_region  = "us-east-1"
environment = "dev"

# EKS
eks_cluster_version     = "1.30"
eks_node_instance_types = ["t3.medium"]
eks_node_desired_count  = 2
eks_node_min_count      = 1
eks_node_max_count      = 4

# RDS
db_instance_class = "db.t3.micro"
db_name           = "jobplatform"
db_username       = "postgres"
db_password       = "change-me-in-production"

# ElastiCache
redis_node_type = "cache.t3.micro"

# Amazon MQ
mq_instance_type = "mq.t3.micro"
mq_username      = "admin"
mq_password      = "change-me-in-production"
