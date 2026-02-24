"""
Experiment 6: Partition Scaling Test
====================================
Tests how Kafka partition count affects system throughput with fixed worker count.

Setup:
- Fixed worker count: 4 Kafka consumers
- Variable partition counts: 4, 8, 16, 32
- Measures: throughput, latency, error rate

Run with:
    locust -f experiment6_partitions.py --host=http://35.90.44.88:8080

Then open browser: http://localhost:8089
Recommended settings: 20 users, spawn rate 5, run time 3 minutes
"""

from locust import HttpUser, task, between, events
import json
from datetime import datetime, timedelta
import time
import csv
import os

# =============================================================================
# Global Metrics Tracking
# =============================================================================
class Metrics:
    def __init__(self):
        self.total_jobs_submitted = 0
        self.successful_submissions = 0
        self.failed_submissions = 0
        self.rate_limited = 0
        self.start_time = None
        self.partition_count = 16  
        self.worker_count = 4    
        self.job_ids = []
        
    def reset(self):
        self.total_jobs_submitted = 0
        self.successful_submissions = 0
        self.failed_submissions = 0
        self.rate_limited = 0
        self.start_time = time.time()
        self.job_ids = []
    
    def get_throughput(self):
        if self.start_time is None:
            return 0
        elapsed = time.time() - self.start_time
        return self.successful_submissions / elapsed if elapsed > 0 else 0
    
    def get_error_rate(self):
        total = self.successful_submissions + self.failed_submissions
        return (self.failed_submissions / total * 100) if total > 0 else 0

metrics = Metrics()


# =============================================================================
# Locust User Class
# =============================================================================
class PartitionScalingUser(HttpUser):
    """
    User that submits jobs to test partition scaling.
    
    Each user submits jobs at a moderate rate to simulate realistic load.
    Jobs are scheduled 10 seconds in the future to allow time for processing.
    """
    
    # Wait 100-300ms between requests per user
    wait_time = between(0.1, 0.3)
    
    def on_start(self):
        """Called when a simulated user starts"""
        if metrics.start_time is None:
            metrics.start_time = time.time()
        
        # Unique client ID for this user
        self.client_id = f"partition-test-user-{id(self) % 10}"
        self.jobs_submitted = 0
    
    @task
    def submit_job(self):
        """
        Submit a job to the scheduler.
        
        Creates a PAYMENT_PROCESS job scheduled 10 seconds in the future.
        Tracks success/failure for metrics.
        """
        
        # Create job payload
        payload = {
            "type": "PAYMENT_PROCESS",
            "payload": json.dumps({
                "orderId": f"order-{metrics.total_jobs_submitted}",
                "amount": 100.0,
                "timestamp": datetime.now().isoformat(),
                "testRun": "partition-scaling"
            }),
            "scheduledAt": (datetime.now() + timedelta(seconds=10)).isoformat()
        }
        
        # HTTP headers
        headers = {
            "Content-Type": "application/json",
            "X-Client-Id": self.client_id
        }
        
        # Submit job with error handling
        with self.client.post(
            "/api/jobs",
            json=payload,
            headers=headers,
            catch_response=True,
            name="Submit Job"
        ) as response:
            
            # Track the request
            metrics.total_jobs_submitted += 1
            
            # Handle different response codes
            if response.status_code == 202:  # Accepted
                metrics.successful_submissions += 1
                self.jobs_submitted += 1
                
                # Try to extract job ID
                try:
                    job_data = response.json()
                    job_id = job_data.get('id') or job_data.get('jobId')
                    if job_id:
                        metrics.job_ids.append(job_id)
                except:
                    pass
                
                response.success()
                
            elif response.status_code in [200, 201]:  # Also acceptable
                metrics.successful_submissions += 1
                self.jobs_submitted += 1
                response.success()
                
            elif response.status_code == 429:  # Rate limited
                metrics.rate_limited += 1
                response.success()  # Expected behavior, not a failure
                
            elif response.status_code >= 500:  # Server error
                metrics.failed_submissions += 1
                response.failure(f"Server error {response.status_code}")
                
            elif response.status_code >= 400:  # Client error
                metrics.failed_submissions += 1
                response.failure(f"Client error {response.status_code}")
                
            else:
                metrics.failed_submissions += 1
                response.failure(f"Unexpected status {response.status_code}")


# =============================================================================
# Event Handlers for Metrics and Reporting
# =============================================================================

@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    """Called when test starts - initialize metrics"""
    print(f"\n{'='*80}")
    print(f"  EXPERIMENT 6: PARTITION SCALING TEST")
    print(f"{'='*80}")
    print(f"  Configuration:")
    print(f"    Partition Count:  {metrics.partition_count}")
    print(f"    Worker Count:     {metrics.worker_count}")
    print(f"    Target Host:      {environment.host}")
    print(f"  ")
    print(f"  Test Parameters:")
    print(f"    Users:            {environment.parsed_options.num_users if hasattr(environment.parsed_options, 'num_users') else 'N/A'}")
    print(f"    Spawn Rate:       {environment.parsed_options.spawn_rate if hasattr(environment.parsed_options, 'spawn_rate') else 'N/A'}")
    print(f"{'='*80}\n")
    
    metrics.reset()


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    """Called when test stops - calculate and save results"""
    
    if metrics.start_time is None:
        print("No test data collected")
        return
    
    # Calculate metrics
    duration = time.time() - metrics.start_time
    throughput = metrics.get_throughput()
    error_rate = metrics.get_error_rate()
    
    # Print summary
    print(f"\n{'='*80}")
    print(f"  TEST RESULTS - {metrics.partition_count} PARTITIONS")
    print(f"{'='*80}")
    print(f"  Jobs Submitted:       {metrics.total_jobs_submitted:,}")
    print(f"  Successful:           {metrics.successful_submissions:,}")
    print(f"  Failed:               {metrics.failed_submissions:,}")
    print(f"  Rate Limited:         {metrics.rate_limited:,}")
    print(f"  Error Rate:           {error_rate:.2f}%")
    print(f"  ")
    print(f"  Duration:             {duration:.2f} seconds")
    print(f"  Throughput:           {throughput:.2f} jobs/sec")
    print(f"  ")
    print(f"  Configuration:")
    print(f"    Partition Count:    {metrics.partition_count}")
    print(f"    Worker Count:       {metrics.worker_count}")
    print(f"{'='*80}\n")
    
    # Save to CSV
    csv_file = 'experiment6_partition_scaling_results.csv'
    file_exists = os.path.isfile(csv_file)
    
    with open(csv_file, 'a', newline='') as f:
        writer = csv.writer(f)
        
        # Write header if new file
        if not file_exists:
            writer.writerow([
                'timestamp',
                'partition_count',
                'worker_count',
                'total_submitted',
                'successful',
                'failed',
                'rate_limited',
                'error_rate_%',
                'duration_sec',
                'throughput_jobs_per_sec',
                'avg_response_time_ms'
            ])
        
        # Get average response time from Locust stats
        avg_response_time = 0
        if environment.stats.total.num_requests > 0:
            avg_response_time = environment.stats.total.avg_response_time
        
        # Write data
        writer.writerow([
            datetime.now().isoformat(),
            metrics.partition_count,
            metrics.worker_count,
            metrics.total_jobs_submitted,
            metrics.successful_submissions,
            metrics.failed_submissions,
            metrics.rate_limited,
            round(error_rate, 2),
            round(duration, 2),
            round(throughput, 2),
            round(avg_response_time, 2)
        ])
    
    print(f"✅ Results saved to {csv_file}\n")


@events.request.add_listener
def on_request(request_type, name, response_time, response_length, exception, **kwargs):
    """Called on each request - allows real-time monitoring"""
    # This enables Locust's built-in statistics tracking
    pass


# =============================================================================
# Main execution info
# =============================================================================
if __name__ == "__main__":
    print("\n" + "="*80)
    print("  PARTITION SCALING EXPERIMENT")
    print("="*80)
    print("\n  This test measures how Kafka partition count affects throughput")
    print("  with a fixed worker count.\n")
    print("  BEFORE RUNNING:")
    print("    1. Update 'partition_count' variable in this file (line 26)")
    print("    2. Ensure Kafka topic has correct partition count (via Terraform)")
    print("    3. Clean database: DELETE FROM jobs;")
    print("    4. Restart app with new consumer group ID")
    print("    5. Verify worker count on EC2\n")
    print("  TO RUN:")
    print("    locust -f experiment6_partitions.py --host=http://35.90.44.88:8080\n")
    print("  RECOMMENDED SETTINGS:")
    print("    Users: 20")
    print("    Spawn rate: 5")
    print("    Run time: 3 minutes")
    print("="*80 + "\n")