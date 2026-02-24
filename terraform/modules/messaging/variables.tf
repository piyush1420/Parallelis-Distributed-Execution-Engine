variable "project_name" {
  description = "Project name"
  type        = string
}

variable "kafka_version" {
  description = "Kafka version"
  type        = string
}

variable "kafka_instance_type" {
  description = "MSK broker instance type"
  type        = string
}

variable "number_of_brokers" {
  description = "Number of Kafka brokers"
  type        = number
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "private_subnet_ids" {
  description = "List of private subnet IDs"
  type        = list(string)
}

variable "app_security_group_id" {
  description = "Application security group ID"
  type        = string
}

variable "msk_security_group_id" {
  description = "MSK security group ID from networking module"
  type        = string
}