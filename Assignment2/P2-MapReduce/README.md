# 2. MapReduce

## 1. Introduction
This problem details the implementation of a MapReduce system using gRPC for distributed processing. The system consists of a master node that coordinates multiple worker nodes to process data in parallel. The implementation supports two processing modes: word count and inverted index generation.

## 2. System Architecture

### 2.1 Components
The system follows a master-worker architecture with the following components:

- **Master Server**: Acts as the coordinator, launching worker processes and assigning map and reduce tasks. It maintains the state of the computation and handles communication with workers.
- **Worker Servers**: Execute the map and reduce tasks assigned by the master. Each worker processes a portion of the input data and produces intermediate or final output.
- **Client**: Initiates the MapReduce job by connecting to the master and specifying the job parameters.

### 2.2 Communication Flow
- **Worker Initialization**: The master spawns worker processes on specific ports when assigning tasks.
- **Map Phase**: Workers process input files and generate intermediate files based on the selected processing mode.
- **Reduce Phase**: Workers process intermediate files to produce the final output.

### 2.3 Service Discovery
The system uses direct port assignments for communication:
- Master server listens on port 50051
- Map workers start at port 56052 and increment
- Reduce workers follow map workers in port assignment

## 3. gRPC Service Definitions
The system defines two main services:

### 3.1 Master Service

```go
service Master {
  rpc AssignMapTask(MapRequest) returns (TaskResponse);
  rpc AssignReduceTask(ReduceRequest) returns (TaskResponse);
}
```

### 3.2 Worker Service
```go
service Worker {
  rpc Map(MapRequest) returns (TaskResponse);
  rpc Reduce(ReduceRequest) returns (TaskResponse);
}
```

### 3.3 Message Types
```go
message MapRequest {
  int32 mapTaskId = 1;
  string filename = 2;
  int32 numReducers = 3;
  string mode = 4;
}

message ReduceRequest {
  int32 reduceTaskId = 1;
  string mode = 2;
}

message TaskResponse {
  bool success = 1;
  string message = 2;
}
```

## 4. Processing Modes

### 4.1 Word Count
Counts the occurrences of each word across all input files:

- **Map Phase**: Each mapper processes an input file, tokenizes it into words, and emits (word, 1) pairs to intermediate files.
- **Reduce Phase**: Each reducer processes intermediate files for its assigned partition, aggregates counts for each word, and writes the final counts to output files.

### 4.2 Inverted Index
Creates an index mapping words to the files they appear in:

- **Map Phase**: Each mapper processes an input file, tokenizes it into words, and emits (word, filename) pairs to intermediate files.
- **Reduce Phase**: Each reducer processes intermediate files for its assigned partition, creates a mapping of words to the list of files they appear in, and writes the inverted index to output files.

## 5. Implementation Details

### 5.1 Master Server
The master server coordinates the MapReduce job:

- Launches worker processes for map and reduce tasks
- Assigns map tasks to process specific input files
- Assigns reduce tasks to process partitions of intermediate data
- Maintains job state and handles worker communication

Key features:
- Dynamic worker process management
- Sequential coordination of map and reduce phases
- Support for different processing modes

### 5.2 Worker Servers
Workers execute the computational tasks:

- Process input files in the map phase
- Aggregate intermediate results in the reduce phase
- Implement the MapReduce logic for both word count and inverted index modes

The map function:
```go
func (w *Worker) Map(ctx context.Context, req *pb.MapRequest) (*pb.TaskResponse, error) {
    // Read input file
    // Process data according to mode
    // Write intermediate files
    return &pb.TaskResponse{Success: true, Message: "Map task completed"}, nil
}
```

The reduce function:
```go
func (w *Worker) Reduce(ctx context.Context, req *pb.ReduceRequest) (*pb.TaskResponse, error) {
    if req.Mode == "inverted_index" {
        return w.reduceInvertedIndex(req)
    } else if req.Mode == "word_count" {
        return w.reduceWordCount(req)
    }
    return &pb.TaskResponse{Success: false, Message: "Invalid mode"}, nil
}
```

### 5.3 Data Flow
- Input data is stored in the "dataset" directory
- Intermediate data is stored in the "intermediate" directory
- Final output is stored in the "output" directory

### 5.4 Concurrency Control
- WaitGroups synchronize the completion of map and reduce tasks
- Sequential launching of workers with delays ensures proper initialization
- Graceful shutdown of worker servers after task completion

## 6. Conclusion
The implemented MapReduce system successfully distributes data processing tasks across multiple workers using gRPC for communication. The architecture supports different processing modes and efficiently handles the coordination of distributed computation.

The system demonstrates key MapReduce principles:
- Parallel processing of input data
- Distributed aggregation of intermediate results
- Fault tolerance through task isolation
- Scalability through dynamic worker allocation

This implementation provides a foundation for more complex distributed processing systems and can be extended with additional features such as fault recovery, dynamic load balancing, and support for more processing modes.

## 7. Running the System

### 7.1 Prerequisites
- Go 1.18 or later
- `protoc` compiler
- `protoc-gen-go` and `protoc-gen-go-grpc` plugins (install via `go install`)

### 7.2 Setup
```bash
# Initialize Go module (only needed once)
go mod init github.com/example

# Download dependencies
go mod tidy

# Generate Go files from proto definitions
make proto
```

### 7.3 Directory Structure
```
.
├── client/
│   └── main.go
├── server/
│   └── main.go
├── master/
│   └── main.go
├── protofiles/
│   └── mapreduce.proto
├── dataset/
│   └── file1.txt, file2.txt, ...
├── intermediate/
├── output/
└── Makefile
```

### 7.4 Running
```bash

# In one terminal: start a worker (port 50050 used here)
go run server/main.go 50050

# In a second terminal: run the client with optional args or modify in `Makefile`
make client NUM_REDUCERS=2 MODE=word_count
# or
make client NUM_REDUCERS=2 MODE=inverted_index
```

Output will be written to the `output/` directory.