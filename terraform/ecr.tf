# =============================================================================
# ECR — Container image repositories
# =============================================================================

locals {
  ecr_repositories = ["api", "worker", "frontend"]
}

resource "aws_ecr_repository" "main" {
  for_each = toset(local.ecr_repositories)

  name                 = "${local.name}-${each.key}"
  image_tag_mutability = "MUTABLE"
  force_delete         = var.environment != "prod"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = { Name = "${local.name}-${each.key}" }
}

# Lifecycle policy — keep only the last 10 untagged images.
resource "aws_ecr_lifecycle_policy" "main" {
  for_each   = aws_ecr_repository.main
  repository = each.value.name

  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Keep last 10 untagged images"
      selection = {
        tagStatus   = "untagged"
        countType   = "imageCountMoreThan"
        countNumber = 10
      }
      action = { type = "expire" }
    }]
  })
}
