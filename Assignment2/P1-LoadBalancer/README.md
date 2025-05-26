# 1. Balance Load?

## 1. Introduction
This problem details the implementation of a look-aside load balancing system using gRPC. The system consists of three main components: a load balancer server, backend servers, and clients. The primary goal is to efficiently distribute computational load across multiple backend servers using different load balancing strategies.

## 2. System Architecture

### 2.1 Components
The system follows a distributed architecture with the following components:

- **Load Balancer (LB) Server**: Acts as the central coordination point, maintaining a registry of available backend servers and their current load status. It implements three different load balancing policies and responds to client queries with the most suitable backend server based on the chosen policy.
- **Backend Servers**: Register with the LB server, periodically report their load status, and handle computational tasks requested by clients. Each server has a threshold for maximum concurrent tasks after which it becomes unavailable for new requests.
- **Clients**: Query the LB server to determine which backend server to use for their computational requests. They then connect directly to the selected backend server to execute their tasks.

### 2.2 Communication Flow

1. **Server Registration**: Each backend server registers with the LB server when it starts.
2. **Load Reporting**: Backend servers periodically report their load status to the LB server.
3. **Server Selection**: Clients request the best server from the LB server based on a specified load balancing strategy.
4. **Task Execution**: Clients connect to the selected backend server and submit their computational tasks.

### 2.3 Service Discovery
The system uses etcd, a distributed key-value store, for service discovery:

- Backend servers register their addresses and status in etcd
- The LB server maintains its own entry in etcd with a time-to-live (TTL) lease
- Clients can discover the LB server through etcd
- The LB server discovers backend servers by querying etcd

## 3. gRPC Service Definitions
The system uses gRPC for all inter-component communication. The following services and RPCs are defined:

### 3.1 LoadBalancer Service

```go
service LoadBalancer {
 rpc RegisterServer(ServerInfo) returns (RegisterResponse);
 rpc ReportLoad(ServerLoad) returns (LoadResponse);
 rpc GetBestServer(BalanceRequest) returns (ServerInfo);
}
```

### 3.2 BackendService
service BackendService {
 rpc Compute(TaskRequest) returns (TaskResponse);
}

### 3.3 Message Types

```go
// Load balancing strategies enum
enum LoadBalanceStrategy {
 PICK_FIRST = 0;
 ROUND_ROBIN = 1;
 LEAST_LOAD = 2;
}

message ServerInfo {
 string address = 1;
}

message ServerLoad {
 string address = 1;
 int32 load = 2;
 bool available = 3;
}

message BalanceRequest {
 LoadBalanceStrategy strategy = 1;
}

message TaskRequest {
 string task = 1;
}

message TaskResponse {
 string result = 1;
}

message RegisterResponse {
 string message = 1;
}

message LoadResponse {
 string message = 1;
}

```

## 4. Load Balancing Policies
Three load balancing strategies are implemented in the system:

### 4.1 Pick First
The simplest strategy, Pick First selects the first available server from the list. The implementation retrieves all available servers from etcd and returns the first one in the list. This approach is best suited for situations where backend servers have similar performance characteristics and load conditions.

```go
selected := availableServers[0]
log.Printf("[PICK_FIRST] Selected server: %s with load: %d", selected.Address, selected.Load)
return &pb.ServerInfo{Address: selected.Address}, nil
```

### 4.2 Round Robin
The Round Robin strategy distributes requests cyclically among available servers, ensuring that each server receives an equal share of requests over time. A mutex-protected counter (rrIndex) is maintained to track the current position in the server list.

```go
lb.mu.Lock()
selected := availableServers[lb.rrIndex%len(availableServers)]
lb.rrIndex++
lb.mu.Unlock()
log.Printf("[ROUND_ROBIN] Selected server: %s with load: %d", selected.Address, selected.Load)
return &pb.ServerInfo{Address: selected.Address}, nil
```

### 4.3 Least Load
The Least Load strategy routes requests to the server with the lowest current load. This approach is particularly effective for workloads with varying computational requirements, as it dynamically adapts to the changing load conditions of the backend servers.

```go
best := availableServers[0]
for _, s := range availableServers {
   if s.Load < best.Load {
       best = s
   }
}
return &pb.ServerInfo{Address: best.Address}, nil
```

## 5. Implementation Details

### 5.1 Load Balancer Server
The LB server maintains a list of available servers and their load status in etcd. It implements the three load balancing policies and responds to client queries with the best available server. The server itself is registered in etcd with a TTL lease, which is kept alive to ensure continuous availability.

Key features:

- Server registration and status tracking
- Load reporting and availability monitoring
- Strategy-based server selection
- Service discovery via etcd

### 5.2 Backend Servers
Each backend server registers with the LB server upon startup and periodically reports its load status. The load is measured by the number of concurrent tasks being handled. When this number exceeds a threshold (maxConcurrentTasks), the server marks itself as unavailable for new requests.

The backend servers implement a computationally intensive task (Fibonacci calculation with recursive implementation) to simulate varying CPU loads:

```go
func fibonacci(n int) int {
   if n <= 1 {
       return n
   }
   return fibonacci(n-1) + fibonacci(n-2)
}
```

### 5.3 Clients
Clients query the LB server for the best backend server to use based on the specified load balancing strategy. They then connect directly to the selected server to execute their computational tasks. After completing a task, they close the connection and obtain a new server recommendation from the LB server for their next task.

### 5.4 Concurrency Control

- Atomic operations are used to safely update and read the concurrent task counters
- Mutex locks protect shared data in the LB server (round-robin index)
- WaitGroups coordinate the concurrent execution of clients and servers

## 6. Performance Analysis

### 6.1 Test Methodology
The system was tested using multiple concurrent clients sending compute-intensive Fibonacci calculations to the backend servers. The client program accepts parameters for:

- Number of concurrent clients
- Test duration
- Load balancing strategy
- LB server address

### 6.2 Metrics
The following metrics were measured:

- Total number of requests processed
- Average latency (in milliseconds)
- Throughput (requests per second)
- Server utilization (concurrent tasks per server)

### 6.3 Results Comparison
Each load balancing strategy was tested with 100 concurrent clients and 25 servers over a 30-second period:

Strategy | Total Requests | Avg Latency (ms) | Throughput (req/sec)
---------|---------------|------------------|--------------------
Least Load | 300 | 12583.33 | 10
Round Robin | 289 | 13110.56 | 9.63
Pick First | 299 | 12603.24 | 9.97

### 6.4 Analysis

1. **Least Load** shows remarkably consistent performance, maintaining the same throughput of 10 req/sec despite the increased client load. It now demonstrates the lowest latency among all strategies (12583.33 ms), suggesting it handles higher concurrency efficiently in this particular environment.
2. **Pick First** remains highly competitive, with throughput (9.97 req/sec) nearly matching Pick First and latency (12603.24 ms) only marginally higher. The strategy continues to demonstrate its ability to effectively distribute load based on server capacity.

3. **Round Robin** shows improved relative performance with 100 clients compared to the 50-client test, processing 289 requests with a throughput of 9.63 req/sec. However, it still exhibits the highest latency (13110.56 ms) among the three strategies.

## 7. Conclusion
The look-aside load balancing system successfully distributes computational load across multiple backend servers using different load balancing strategies. The Least Load strategy proved to be the most effective for the tested workload, providing the best balance between throughput and latency.


## 8. How to Run

### Step 1: Install Requirements

Install `protoc`:

```bash
# Ubuntu
sudo apt install protobuf-compiler

# Mac
brew install protobuf
```

Install gRPC Go plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

# Add to PATH if needed
export PATH="$PATH:$(go env GOPATH)/bin"
```

---

### Step 2: Clone and Initialize Project

```bash
cd P1-LoadBalancer

# (if starting from scratch)
go mod init github.com/yourusername/loadbalancer
go mod tidy
```

---

### Step 3: Generate gRPC Code

```bash
make proto
```

---

### Step 4: Start the Load Balancer Server

In one terminal:

```bash
go run lb_server/main.go
```

This:
- Starts the Load Balancer server
- Connects to etcd and begins watching registered backend servers

---

### Step 5: Start Backend Servers

In a separate terminal:

```bash
make server
```

This:
- Starts multiple backend servers (default: 3)
- Registers them with the LB server via etcd

---

### Step 6: Run Clients

In another terminal:

```bash
make client
```

You can also manually specify parameters or edit in `Makefile`:

```bash
go run client/main.go -clients=10 -duration=30 -strategy=least_load -lb=localhost:50050
```

Where:
- `-clients`: Number of concurrent client goroutines
- `-duration`: Duration (in seconds) to send requests
- `-strategy`: `pick_first`, `round_robin`, or `least_load`
- `-lb`: Address of the Load Balancer

---

### Optional: Clean Proto Files

```bash
make clean
```