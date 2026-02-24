#!/bin/bash
set -e

# Update system
yum update -y

# Install Java 17
yum install java-17-amazon-corretto-headless -y

# Install CloudWatch agent (optional, for monitoring)
wget https://s3.amazonaws.com/amazoncloudwatch-agent/amazon_linux/amd64/latest/amazon-cloudwatch-agent.rpm
rpm -U ./amazon-cloudwatch-agent.rpm

# Create application directory
mkdir -p /opt/jobscheduler
chown ec2-user:ec2-user /opt/jobscheduler

# Setup systemd service (template, will be configured later)
cat > /etc/systemd/system/jobscheduler.service <<'EOF'
[Unit]
Description=Job Scheduler Application
After=network.target

[Service]
Type=simple
User=ec2-user
WorkingDirectory=/opt/jobscheduler
ExecStart=/usr/bin/java -jar /opt/jobscheduler/app.jar --spring.config.location=/opt/jobscheduler/application.properties
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload

echo "✅ EC2 instance setup complete"