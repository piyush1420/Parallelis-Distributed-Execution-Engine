# Networking Outputs
output "vpc_id" {
  description = "VPC ID"
  value       = module.networking.vpc_id
}

# Database Outputs
output "database_endpoint" {
  description = "RDS PostgreSQL endpoint"
  value       = module.database.endpoint
}

output "database_connection_string" {
  description = "JDBC connection string"
  value       = module.database.connection_string
  sensitive   = true
}

# Cache Outputs
output "redis_endpoint" {
  description = "ElastiCache Redis endpoint"
  value       = module.cache.endpoint
}

# Messaging Outputs
output "kafka_bootstrap_servers" {
  description = "MSK Kafka bootstrap servers"
  value       = module.messaging.bootstrap_brokers
}

# Compute Outputs
output "app_server_public_ip" {
  description = "Public IP of application server"
  value       = module.compute.public_ip
}

output "ssh_command" {
  description = "SSH command to connect to application server"
  value       = "ssh -i ${var.key_pair_name}.pem ec2-user@${module.compute.public_ip}"
}

# Complete Application Configuration
output "application_properties" {
  description = "Spring Boot application.properties for AWS"
  value = <<-EOT
    # Database Configuration
    spring.datasource.url=${module.database.connection_string}
    spring.datasource.username=${var.db_username}
    spring.datasource.password=${var.db_password}

    # Redis Configuration
    spring.data.redis.host=${split(":", module.cache.endpoint)[0]}
    spring.data.redis.port=${split(":", module.cache.endpoint)[1]}

    # Kafka Configuration
    spring.kafka.bootstrap-servers=${module.messaging.bootstrap_brokers}
  EOT
  sensitive = true
}