output "endpoint" {
  description = "RDS endpoint"
  value       = aws_db_instance.postgres.endpoint
}

output "connection_string" {
  description = "JDBC connection string"
  value       = "jdbc:postgresql://${aws_db_instance.postgres.endpoint}/jobscheduler"
}

output "database_name" {
  description = "Database name"
  value       = aws_db_instance.postgres.db_name
}

output "port" {
  description = "Database port"
  value       = aws_db_instance.postgres.port
}