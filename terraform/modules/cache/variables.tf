variable "project_name" {
  description = "Project name"
  type        = string
}

variable "redis_node_type" {
  description = "ElastiCache node type"
  type        = string
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
variable "redis_security_group_id" {
  description = "Redis security group ID from networking module"
  type        = string
}