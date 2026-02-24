# General
aws_region   = "us-west-2"
project_name = "job-scheduler"

# Networking
vpc_cidr = "10.0.0.0/16"
az_count = 2
my_ip    = "73.158.198.26/32"

# Database
db_username       = "dbadmin"
db_password       = "Password123!"
db_instance_class = "db.t3.micro"

# Cache
redis_node_type = "cache.t3.micro"

# Messaging
kafka_version       = "3.5.1"
kafka_instance_type = "kafka.t3.small"
number_of_brokers   = 2

# Compute
instance_type  = "t3.small"
key_pair_name  = "job-scheduler-key"