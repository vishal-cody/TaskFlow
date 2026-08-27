# =============================================================================
# Amazon MQ — RabbitMQ Broker
# =============================================================================

resource "aws_security_group" "mq" {
  name_prefix = "${local.name}-mq-"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 5671
    to_port     = 5671
    protocol    = "tcp"
    cidr_blocks = local.private_subnets
    description = "AMQPS from private subnets"
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = local.private_subnets
    description = "HTTPS management console"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${local.name}-mq-sg" }
}

resource "aws_mq_broker" "main" {
  broker_name = "${local.name}-rabbitmq"

  engine_type        = "RabbitMQ"
  engine_version     = "3.13"
  host_instance_type = var.mq_instance_type
  deployment_mode    = var.environment == "prod" ? "CLUSTER_MULTI_AZ" : "SINGLE_INSTANCE"

  publicly_accessible = false
  subnet_ids          = var.environment == "prod" ? aws_subnet.private[*].id : [aws_subnet.private[0].id]
  security_groups     = [aws_security_group.mq.id]

  user {
    username = var.mq_username
    password = var.mq_password
  }

  tags = { Name = "${local.name}-rabbitmq" }
}
