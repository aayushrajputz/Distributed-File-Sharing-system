#!/bin/bash

# Setup MinIO bucket policy for file sharing
echo "Setting up MinIO bucket policy..."

# Wait for MinIO to be ready
echo "Waiting for MinIO to be ready..."
until curl -f http://localhost:9000/minio/health/live > /dev/null 2>&1; do
    echo "Waiting for MinIO..."
    sleep 2
done

echo "MinIO is ready!"

# Create bucket policy
cat > /tmp/bucket-policy.json << EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {"AWS": "*"},
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject"
      ],
      "Resource": ["arn:aws:s3:::file-sharing/*"]
    }
  ]
}
EOF

# Set the bucket policy using MinIO client
echo "Setting bucket policy..."
docker exec minio mc config host add myminio http://localhost:9000 minioadmin minioadmin
docker exec minio mc policy set-json /tmp/bucket-policy.json myminio/file-sharing

echo "Bucket policy set successfully!"
echo "MinIO setup complete!"
