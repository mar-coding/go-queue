# Reliable Queuing System

A distributed and fault-tolerant message queuing platform that enables multiple broker nodes to collaboratively offer reliable queuing services to clients.

## Table of Contents
- [System Overview](#system-overview)
- [Features](#features)
- [Architecture](#architecture)
- [Design Decisions and Trade-offs](#design-decisions-and-trade-offs)
- [Implementation Details](#implementation-details)
- [Getting Started](#getting-started)
- [API Reference](#api-reference)
- [Reliability Analysis](#reliability-analysis)
- [Future Improvements](#future-improvements)

## System Overview

This project implements a distributed and reliable queuing platform where multiple broker nodes collaborate to provide queuing services to clients. The system is designed to be fault-tolerant, ensuring that if a broker crashes, the overall system continues to operate without data loss.

Key characteristics of the system:
- Queues are persistent, append-only, FIFO data structures
- Multiple brokers replicate queue data to ensure fault tolerance
- Clients can create queues, append data, and read data from queues
- Each client has a unique read position per queue

## Features

- **Distributed Architecture**: Multiple broker nodes work together to provide a reliable service
- **Fault Tolerance**: The system continues to function even if some broker nodes fail
- **Data Replication**: Queue data is replicated across multiple nodes to prevent data loss
- **Persistence**: All queue data is stored persistently on disk
- **Client Tracking**: The system keeps track of each client's read position
- **REST API**: Simple HTTP API for client interactions
- **Health Monitoring**: Continuous monitoring of node health

## Architecture

The system follows a multi-node architecture with the following components:

### Components

1. **Broker Nodes**: Independent server instances that handle queue operations and replicate data
2. **Queues**: FIFO data structures that store messages
3. **Clients**: External systems that interact with brokers to use the queuing service
4. **Node Registry**: Keeps track of available broker nodes and their health status
5. **File Storage**: Persistent storage system for queues and client data

### Communication

- **REST API**: External client communication over HTTP
- **RPC Protocol**: Internal broker-to-broker communication for replication and coordination
- **Health Checks**: Periodic ping messages to check node status

## Design Decisions and Trade-offs

### 1. Consistency vs. Availability

**Decision**: The system prioritizes availability over strong consistency.

**Rationale**: 
- In a distributed queuing system, ensuring messages are always available for processing is critical
- The system allows operations to proceed even when some nodes are down
- Eventual consistency is achieved through background replication

**Trade-offs**:
- During network partitions, clients may see slightly different queue states across different brokers
- In rare cases, a message might be read twice if a client reconnects to a different broker that hasn't yet synchronized

### 2. Replication Strategy

**Decision**: Implemented an active replication strategy with configurable replication factor.

**Rationale**:
- Multiple copies of each queue ensure data durability
- Configurable replication factor allows for balancing between reliability and resource usage
- Replicas are dynamically assigned based on node availability

**Trade-offs**:
- Higher replication factor increases reliability but requires more storage and network resources
- Write operations need to be propagated to all replicas, which can increase latency

### 3. Storage Mechanism

**Decision**: File-based persistent storage instead of in-memory storage.

**Rationale**:
- Ensures data durability across node restarts
- Simpler to implement than a distributed database
- File system operations are relatively efficient for the expected workload

**Trade-offs**:
- Slower than pure in-memory solutions
- File I/O operations can become a bottleneck under high load
- Requires careful file handling to prevent corruption

### 4. Client Offset Management

**Decision**: Server-side tracking of client read positions.

**Rationale**:
- Simplifies client implementation
- Allows for seamless client reconnection to different brokers
- Enables at-least-once message delivery semantics

**Trade-offs**:
- Requires additional state management on the server
- Increases memory usage proportional to the number of clients
- Requires synchronization of client offsets across replicas

### 5. Communication Protocol

**Decision**: HTTP for client-to-broker communication and a custom RPC protocol for inter-broker communication.

**Rationale**:
- HTTP provides a simple, widely-supported interface for clients
- Custom RPC allows for more efficient broker-to-broker communication
- Both protocols are relatively simple to implement and debug

**Trade-offs**:
- HTTP has higher overhead than binary protocols
- Custom RPC implementation may not be as robust as established frameworks
- Two different protocols increase implementation complexity

## Implementation Details

### Technology Stack

- **Language**: Go 1.19
- **Concurrency**: Goroutines and channels for parallel processing
- **HTTP Framework**: Standard Go HTTP library
- **Persistence**: Custom file-based storage implementation
- **Serialization**: JSON for both REST API and inter-node communication

### Key Components

1. **Entity Layer**:
   - Queue: Represents a message queue with FIFO semantics
   - Message: Individual data elements stored in queues
   - Node: Represents a broker node in the cluster
   - Client: Tracks client information and read positions

2. **Repository Layer**:
   - QueueRepository: Manages persistent storage of queues and messages
   - FileStorage: Handles low-level file operations for persistence

3. **Service Layer**:
   - QueueService: Implements queue operations (create, append, read)
   - NodeService: Manages node health checking and replica selection

4. **Transport Layer**:
   - REST Handler: Processes client HTTP requests
   - RPC Client/Server: Handles inter-node communication

5. **Configuration**:
   - Node configuration via JSON files
   - Dynamic discovery of other nodes in the cluster

### Replication Process

1. When a queue is created:
   - The broker selects N-1 other nodes as replicas (where N is the replication factor)
   - The queue is created on all replica nodes

2. When a message is appended:
   - The message is stored locally
   - The message is asynchronously sent to all replica nodes
   - The operation returns success after local storage is confirmed

3. When a node fails:
   - Health checks detect the failure
   - For each affected queue, a new replica is selected
   - Queue data is synchronized to the new replica

## Getting Started

### Prerequisites

- Go 1.19 or later
- Linux/macOS/Windows operating system

### Running a Single Broker

```bash
# Start a broker node
./start.sh 1
```

### Running a Cluster

```bash
# Start multiple broker nodes
./start.sh 1  # In terminal 1
./start.sh 2  # In terminal 2
./start.sh 3  # In terminal 3
```

### Testing the System

```bash
# Run the test script
./test_queue.sh
```

## API Reference

### Create Queue

```
POST /createQueue
{
  "name": "my-queue"
}
```

Response:
```
{
  "queueId": "uuid-string",
  "message": "Queue created successfully"
}
```

### Append Data

```
POST /appendData
{
  "queueId": "uuid-string",
  "clientId": "client-uuid",
  "data": 123  // Integers for simplicity
}
```

Response:
```
{
  "messageId": "uuid-string",
  "message": "Data appended successfully"
}
```

### Read Data

```
GET /readData?queueId=uuid-string&clientId=client-uuid
```

Response:
```
{
  "messageId": "uuid-string",
  "data": "123"
}
```

### Node Status

```
GET /status
```

Response:
```
{
  "nodeId": "node1",
  "aliveNodes": 3,
  "totalNodes": 3,
  "deadNodes": 0,
  "uptimeHours": 2,
  "version": "1.0.0",
  "queueCount": 5
}
```

## Reliability Analysis

The system provides fault tolerance through the following mechanisms:

1. **Data Replication**:
   - Each queue is replicated across multiple nodes
   - If a node fails, its data is still available on other nodes

2. **Node Health Monitoring**:
   - Regular health checks detect node failures
   - Failed nodes are removed from the available node list

3. **Dynamic Replica Management**:
   - When a node fails, new replicas are automatically created
   - The system maintains the configured replication factor

4. **Graceful Degradation**:
   - The system continues to function with reduced capacity when nodes fail
   - As long as at least one replica of a queue is available, operations can proceed

5. **Recovery Process**:
   - When a failed node recovers, it can rejoin the cluster
   - Data synchronization ensures the recovered node has up-to-date information

### Reliability Guarantees

- **Message Persistence**: Once acknowledged, a message is guaranteed to be stored on disk
- **At-Least-Once Delivery**: Each message will be delivered to a client at least once
- **FIFO Ordering**: Messages are delivered in the same order they were appended
- **Durability**: Data survives node restarts and crashes

## Future Improvements

1. **Enhanced Consistency**:
   - Implement stronger consistency models for critical applications
   - Add quorum-based writes for improved reliability

2. **Performance Optimizations**:
   - Batch processing of messages
   - More efficient storage format
   - Cache frequently accessed data

3. **Advanced Features**:
   - Message TTL (Time-To-Live)
   - Topic-based publishing
   - Message filtering

4. **Monitoring and Management**:
   - Admin dashboard for system monitoring
   - Advanced metrics collection
   - Alerting system for node failures

5. **Security Enhancements**:
   - Authentication and authorization
   - Encrypted communication
   - Access control per queue