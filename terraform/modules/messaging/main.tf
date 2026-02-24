# MSK Configuration
resource "aws_msk_configuration" "main" {
  name              = "${var.project_name}-msk-config"
  kafka_versions    = [var.kafka_version]

  server_properties = <<PROPERTIES
auto.create.topics.enable=true
default.replication.factor=2
min.insync.replicas=1
num.partitions=16
log.retention.hours=168
PROPERTIES
}

# MSK Cluster
resource "aws_msk_cluster" "main" {
  cluster_name           = "${var.project_name}-kafka"
  kafka_version          = var.kafka_version
  number_of_broker_nodes = var.number_of_brokers

  broker_node_group_info {
    instance_type   = var.kafka_instance_type
    client_subnets  = var.private_subnet_ids
    security_groups = [var.msk_security_group_id]

    storage_info {
      ebs_storage_info {
        volume_size = 100
      }
    }
  }

  configuration_info {
    arn      = aws_msk_configuration.main.arn
    revision = aws_msk_configuration.main.latest_revision
  }

  encryption_info {
    encryption_in_transit {
      client_broker = "PLAINTEXT"
      in_cluster    = true
    }
  }

  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = aws_cloudwatch_log_group.msk.name
      }
    }
  }

  tags = {
    Name = "${var.project_name}-kafka"
  }
}

# CloudWatch Log Group for MSK
resource "aws_cloudwatch_log_group" "msk" {
  name              = "/aws/msk/${var.project_name}"
  retention_in_days = 7

  tags = {
    Name = "${var.project_name}-msk-logs"
  }
}