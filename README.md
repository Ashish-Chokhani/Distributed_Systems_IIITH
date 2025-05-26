# Distributed Systems Assignments – IIITH

This repository contains all the assignments for the Distributed Systems course at IIIT Hyderabad. Each assignment explores a different concept in distributed systems using technologies such as MPI and gRPC, covering topics like parallel processing, distributed file systems, load balancing, MapReduce, distributed transactions, and Byzantine fault tolerance.

---

## 📦 Assignment 1

### 🧭 P1: Escape the Collapsing Maze
**Concepts**: MPI, Distributed Breadth-First Search  
**Description**:  
Implements a distributed solution using MPI to help agents escape a dynamically collapsing maze. Each agent explores the maze using a parallelized BFS algorithm across processes.  
📁 Directory: `P1-MazeEscape`

---

### 🧶 P2: Bob and the Bouncing Balls
**Concepts**: MPI, Agent Simulation, Synchronization  
**Description**:  
Simulates 2D physics of bouncing balls with moving agents using MPI. Handles collisions and movement updates in a distributed manner, optimizing communication and computation balance.  
📁 Directory: `P2-BouncingBalls`

---

### 📁 P3: Avengers Distributed File System Quest
**Concepts**: Fault-Tolerant File System, Chunking, Replication, MPI  
**Description**:  
Implements a distributed file system with chunk-level replication, supporting file upload, retrieval, and fault tolerance using MPI-based server-client architecture.  
📁 Directory: `P3-DistributedFS`

---

## 🛰️ Assignment 2

### ⚖️ P1: Balance Load?
**Concepts**: gRPC, Load Balancing, Service Discovery, etcd  
**Description**:  
Implements a gRPC-based load balancer using etcd for dynamic service registration and load monitoring. Supports multiple strategies: Pick First, Round Robin, and Least Load.  
📁 Directory: `P1-LoadBalancer`

---

### 🗺️ P2: MapReduce – Word Count & Inverted Index
**Concepts**: gRPC, Master-Worker Architecture, MapReduce  
**Description**:  
Implements a simplified MapReduce system using gRPC. Supports two modes: word count and inverted index. Master distributes tasks to workers discovered dynamically via port info.  
📁 Directory: `P2-MapReduce`

---

### 💳 P3: Strife – Distributed Payment System
**Concepts**: gRPC, Two-Phase Commit, Idempotency, TLS, Offline Processing  
**Description**:  
Simulates a distributed payment processing system with a Gateway and Bank Servers using gRPC. Implements Two-Phase Commit for atomic transactions and ensures fault-tolerant behavior with support for retries and offline messages.  
📁 Directory: `P3-Strife`

---

### 🛡️ P4: Byzantine Failure Resolution
**Concepts**: Byzantine Fault Tolerance, Message Passing, Majority Voting, gRPC  
**Description**:  
Implements a simulation of the Byzantine Generals Problem. The solution adheres to the condition `n > 3t` (where `n` is total nodes and `t` the number of faulty nodes), using `t+1` rounds of communication and majority voting for decision-making. The implementation detects consensus violations, handles message tampering by traitors, and simulates attack scenarios such as traitor commanders.

📁 Directory: `P4-ByzantineConsensus`

#### Highlights:
- Multi-round message exchange and majority voting
- Traitor simulation and message tampering
- Detection of cycles and round-based validation
- O(n² × t) message and memory complexity
- Evaluation of performance, real-world applicability, and scalability

---

## 🚀 How to Run

Each assignment directory includes its own `README.md` with detailed setup and execution instructions.

### Requirements
- **Python 3.10+**
- **gRPC & Protobuf**
- **MPI Tools** (e.g., `OpenMPI`)
- **etcd** (used in P1: Balance Load for service discovery)


### 📚 Acknowledgments
These projects were developed as part of the Distributed Systems course at IIIT Hyderabad. All solutions were individually implemented, strictly adhering to academic integrity policies.
