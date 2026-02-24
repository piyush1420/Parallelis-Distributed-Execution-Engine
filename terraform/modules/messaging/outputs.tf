output "bootstrap_brokers" {
  description = "MSK bootstrap brokers (plaintext)"
  value       = aws_msk_cluster.main.bootstrap_brokers
}

output "bootstrap_brokers_tls" {
  description = "MSK bootstrap brokers (TLS)"
  value       = aws_msk_cluster.main.bootstrap_brokers_tls
}

output "cluster_arn" {
  description = "MSK cluster ARN"
  value       = aws_msk_cluster.main.arn
}

output "zookeeper_connect_string" {
  description = "Zookeeper connection string"
  value       = aws_msk_cluster.main.zookeeper_connect_string
}