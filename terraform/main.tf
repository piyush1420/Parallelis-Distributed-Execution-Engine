terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Data source for availability zones
data "aws_availability_zones" "available" {
  state = "available"
}

# Networking Module (VPC, Subnets, Security Groups)
module "networking" {
  source = "./modules/networking"

  project_name = var.project_name
  vpc_cidr     = var.vpc_cidr
  az_count     = var.az_count
  my_ip        = var.my_ip
}
# Database Module
module "database" {
  source = "./modules/database"

  project_name       = var.project_name
  db_username        = var.db_username
  db_password        = var.db_password
  db_instance_class  = var.db_instance_class

  vpc_id             = module.networking.vpc_id
  private_subnet_ids = module.networking.private_subnet_ids
  app_security_group_id = module.networking.app_security_group_id
  rds_security_group_id = module.networking.rds_security_group_id  # ← ADD THIS
}

# Cache Module
module "cache" {
  source = "./modules/cache"

  project_name          = var.project_name
  redis_node_type       = var.redis_node_type

  vpc_id                = module.networking.vpc_id
  private_subnet_ids    = module.networking.private_subnet_ids
  app_security_group_id = module.networking.app_security_group_id
  redis_security_group_id = module.networking.redis_security_group_id  # ← ADD THIS
}

# Messaging Module
module "messaging" {
  source = "./modules/messaging"

  project_name          = var.project_name
  kafka_version         = var.kafka_version
  kafka_instance_type   = var.kafka_instance_type
  number_of_brokers     = var.number_of_brokers

  vpc_id                = module.networking.vpc_id
  private_subnet_ids    = module.networking.private_subnet_ids
  app_security_group_id = module.networking.app_security_group_id
  msk_security_group_id = module.networking.msk_security_group_id  # ← ADD THIS
}

# Compute Module (EC2 Application Server)
module "compute" {
  source = "./modules/compute"

  project_name       = var.project_name
  instance_type      = var.instance_type
  key_pair_name      = var.key_pair_name

  vpc_id             = module.networking.vpc_id
  public_subnet_id   = module.networking.public_subnet_ids[0]
  security_group_id  = module.networking.app_security_group_id
}