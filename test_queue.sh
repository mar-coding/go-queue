#!/bin/bash

echo "Testing Distributed Queue System"
echo "-------------------------------"

# Base URL for API
BASE_URL="http://localhost:6001"

# Create a test queue
echo "Creating queue 'test-queue'..."
CREATE_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" -d '{"name": "test-queue2"}' $BASE_URL/createQueue)
echo "Create Response: $CREATE_RESPONSE"

# Extract queue ID using grep and cut
QUEUE_ID=$(echo $CREATE_RESPONSE | grep -o '"queueId":"[^"]*"' | cut -d'"' -f4)
echo "Queue created with ID: $QUEUE_ID"

if [ -z "$QUEUE_ID" ]; then
    echo "Failed to create queue or extract queue ID."
    exit 1
fi

# Append some messages
echo "Appending message 'Hello, world!' to queue $QUEUE_ID..."
APPEND_RESPONSE1=$(curl -s -X POST -H "Content-Type: application/json" -d "{\"queueId\": \"$QUEUE_ID\", \"clientId\": \"test-client\", \"data\": \"Hello, world!\"}" $BASE_URL/appendData)
echo "Append Response 1: $APPEND_RESPONSE1"

echo "Appending message 'This is a test message' to queue $QUEUE_ID..."
APPEND_RESPONSE2=$(curl -s -X POST -H "Content-Type: application/json" -d "{\"queueId\": \"$QUEUE_ID\", \"clientId\": \"test-client\", \"data\": \"This is a test message\"}" $BASE_URL/appendData)
echo "Append Response 2: $APPEND_RESPONSE2"

echo "Appending message 'Third message' to queue $QUEUE_ID..."
APPEND_RESPONSE3=$(curl -s -X POST -H "Content-Type: application/json" -d "{\"queueId\": \"$QUEUE_ID\", \"clientId\": \"test-client\", \"data\": \"Third message\"}" $BASE_URL/appendData)
echo "Append Response 3: $APPEND_RESPONSE3"

# Read the messages back
echo "Reading message 1 from queue $QUEUE_ID..."
READ_RESPONSE1=$(curl -s -X GET "$BASE_URL/readData?queueId=$QUEUE_ID&clientId=test-client")
echo "Read Response 1: $READ_RESPONSE1"

echo "Reading message 2 from queue $QUEUE_ID..."
READ_RESPONSE2=$(curl -s -X GET "$BASE_URL/readData?queueId=$QUEUE_ID&clientId=test-client")
echo "Read Response 2: $READ_RESPONSE2"

echo "Reading message 3 from queue $QUEUE_ID..."
READ_RESPONSE3=$(curl -s -X GET "$BASE_URL/readData?queueId=$QUEUE_ID&clientId=test-client")
echo "Read Response 3: $READ_RESPONSE3"

# This one should fail as we've read all messages
echo "Reading message 4 from queue $QUEUE_ID (should fail)..."
READ_RESPONSE4=$(curl -s -X GET "$BASE_URL/readData?queueId=$QUEUE_ID&clientId=test-client")
echo "Read Response 4: $READ_RESPONSE4"

echo "-------------------------------"
echo "Test completed"