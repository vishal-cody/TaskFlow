# =============================================================================
# Distributed Job Processing Platform — Terraform
# =============================================================================
#
# Provisions the full production infrastructure on AWS:
#
#   ┌──────────────────────────────────────────────────────┐
#   │  VPC (3 AZs, public + private subnets)               │
#   │                                                      │
#   │  ┌───────────┐  ┌───────────┐  ┌──────────────────┐ │
#   │  │  EKS      │  │  RDS      │  │  ElastiCache     │ │
#   │  │  Cluster  │  │  Postgres │  │  Redis            │ │
#   │  └───────────┘  └───────────┘  └──────────────────┘ │
#   │                                                      │
#   │  ┌──────────────────┐  ┌──────────────────────────┐ │
#   │  │  Amazon MQ       │  │  ECR Repositories        │ │
#   │  │  RabbitMQ        │  │  (api, worker, frontend)  │ │
#   │  └──────────────────┘  └──────────────────────────┘ │
#   └──────────────────────────────────────────────────────┘
#
# Usage:
#   cd terraform/
#   terraform init
#   terraform plan -var-file=dev.tfvars
#   terraform apply -var-file=dev.tfvars
#
# =============================================================================

terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Remote backend for team collaboration (uncomment and configure for prod).
  # backend "s3" {
  #   bucket         = "jobplatform-terraform-state"
  #   key            = "infra/terraform.tfstate"
  #   region         = "us-east-1"
  #   dynamodb_table = "terraform-locks"
  #   encrypt        = true
  # }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "jobplatform"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}
