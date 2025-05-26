# 1. Escape the collapsing maze 

## Solution Approach

This program implements a **Distributed Breadth-First Search (BFS)** using MPI for efficient parallelization on large graphs. Below are the key steps in the solution approach:

1. **Graph Partitioning**:  
   - The graph is partitioned across processes by assigning nodes to processes based on their indices (`node % size`).
   - Each process is responsible for maintaining the distances of its assigned nodes.

2. **Frontier-Based BFS**:  
   - The BFS is executed level by level, maintaining a **current frontier** and a **next frontier**.
   - Each process iteratively computes the next frontier for its nodes and communicates updates to other processes.

3. **Communication**:  
   - Each process sends newly discovered nodes (next frontier) to their respective owners using **MPI_Allgatherv** for global communication.
   - A global reduction step using **MPI_Allreduce** l processes agree when no frontier nodes remain, signaling BFS completion.

4. **Broadcast and Synchronization**:  
   - The initial distances and graph structure are broadcast from the root process to all other processes.
   - Synchronization is achieved at each level of BFS to ensure consistency.

5. **Handling Special Constraints**:  
   - Blocked nodes are removed from the graph.
   - Nodes specified as explorers are evaluated for their distances from the source node.

---

## Complexity Analysis

### Sequential BFS
- **Time Complexity**: \(O(N + M)\), where \(N\) is the number of nodes, and \(M\) is the number of edges.
- **Space Complexity**: \(O(N + M)\) for storing the adjacency list and distances.

### Parallel BFS
- **Time Complexity**: \(O((N + M) / P + message overhead), where \(P\) is the number of processes
- **Space Complexity**: \(O((N + M) / P)\) per process, as each process stores only a partition of the graph.

### Message Complexity
- Each process sends the discovered nodes to their owners during every level of BFS.
- Total message complexity per level is proportional to the number of edges crossing process boundaries.

---

## Results

### Performance Across Processes

- The runtime decreases as the number of processes increases, but communication overhead becomes significant for very high process counts.
- Balanced graph partitioning significantly improves performance by reducing inter-process communication.

For N=10000 M=9999 K=100000 L=0, we obtain the following plot:

![Plot 1](./1/1.png)

---

## 🏃‍♂️ How to Run:

To compile and test the first problem — *Escape the Collapsing Maze* — using MPI, follow the steps below.

### Directory Structure

Ensure your project is structured as follows:

```
P1-MazeEscape/
├── 1.cpp
├── 1.png
scripts/
└── P1-MazeEscape/
    ├── 1_test.sh
    ├── marks.txt
    └── testcases/
        ├── 1.in
        ├── 1.out
        ├── 2.in
        ├── 2.out
        └── ... (more test cases)
```

### ✅ Prerequisites

- MPI installed (e.g., OpenMPI or MPICH)
- `mpic++` and `mpiexec` available in your shell
- Bash shell

### 🚀 Run Instructions

1. **Navigate to the test directory**:

   ```bash
   cd scripts/P1-MazeEscape
   ```

2. **Give execute permission to the test script** (if not already):

   ```bash
   chmod +x 1_test.sh
   ```

3. **Run the test script**:

   ```bash
   ./1_test.sh
   ```

### 🔍 What It Does

- Detects the presence of `*.cpp`, `*.c`, or `*.py` files in `../../P1-MazeEscape/`
- Compiles C++ or C code using `mpic++` or `mpicc`
- Runs the executable across 1 to 12 MPI processes for each test case
- Compares the output against expected `.out` files
- Calculates total score from `marks.txt`

### ✅ Expected Output (on success)

If your solution is correct, you should see an output like this:

```
Test case 1: PASSED
Test case 2: PASSED
Test case 3: PASSED
Test case 4: PASSED
Test case 5: PASSED
Test case 6: PASSED
Test case 7: PASSED
Test case 8: PASSED
Test case 9: PASSED
Test case 10: PASSED
Final Score: 15.00/15
```

### 🧹 Cleanup

The script automatically cleans up the compiled binary (`P1-MazeEscape`) and temporary result files after completion.


# 2. Bob and Bouncing Balls

## Solution Approach

This program simulates the movement of agents on a 2D grid based on a set of rules, using MPI (Message Passing Interface) to parallelize computations across multiple processes. The program reads the initial agent positions, grid dimensions, and simulation time from an input file and outputs the final agent positions after applying the rules iteratively for the specified time steps.

---


### Parallelization Strategy
1. **Data Partitioning:**
   - The initial positions of balls are divided among processes. Each process works on a subset of balls based on the rank of the process and the total number of processes.
   - The master process (rank 0) divides the data and sends chunks to other processes using `MPI_Send`. 

2. **Simulation:**
   - Each process computes the next positions for its assigned balls independently based on grid boundary conditions and collision rules.
   - To ensure global collision consistency, each process computes local collision data and exchanges it with all other processes using `MPI_Allgather` for synchronization.

3. **Global Collision Handling:**
   - After synchronizing collision data, all processes update their balls' positions using the global collision information.

4. **Gathering Results:**
   - At the end of the simulation, each process sends its final balls positions back to the master process using  `MPI_Gatherv`.

---

## Highlights of the Program

- **MPI Communication:** Efficient use of `MPI_Allgather` for synchronizing collision data and `MPI_Gatherv` for collecting results from all processes.
- **Boundary Handling:** Agents' movements wrap around the grid using modulo operations to handle boundaries seamlessly.
- **Dynamic Collision Handling:** The program dynamically adjusts the direction of agents based on the number of collisions detected at their destination.
- **Fault-Tolerant Input:** The master process handles input validation, and the program gracefully exits if the input file is invalid.

---

## Complexity Analysis

### Original (Serial) Algorithm
- **Time Complexity:**  
  \(O(T*K)\), where \(T\) is the number of time steps, and \(K\) is the number of agents.  
  For each time step, every ball's position is updated, and collision detection is performed.

- **Space Complexity:**  
  \(O(K)\) to store balls positions.

### Parallel Algorithm
- **Time Complexity:**  
  O((T*K)/P) + T * logP), where \(P\) is the number of processes.  
  - The first term corresponds to the time for each process to update its assigned agents.  
  - The second term accounts for communication overhead during global collision synchronization.

- **Space Complexity:**  
  O(K/P) per process for storing balls positions.

- **Message Complexity:**  
  - Each process exchanges local collision data with \(P - 1\) other processes in every time step, resulting in O(T*P^2) messages.

---

## Results and Performance Analysis

### Performance Across Processes

- **Speedup:** The parallel version shows near-linear speedup with an increasing number of processes until communication overhead becomes significant.

For N=981 M=704 K=175 T=862, we obtain following plot
![Plot 2](./2/2.png)

### Observations
1. **Communication Overhead:** As the number of processes increases, the communication overhead from `MPI_Allgather` and `MPI_Gatherv` becomes the limiting factor.
2. **Load Imbalance:** If the number of balls (\(K\)) is not evenly divisible by the number of processes (\(P\)), some processes may have slightly more work than others, causing load imbalance.

---

## How to Run: P2 - Bob and Bouncing Balls

This section explains how to compile, test, and evaluate the **Bob and Bouncing Balls** MPI program using the provided test script.

### Directory Structure

Ensure the files are organized as follows:

```
P2-BouncingBalls/
├── 2.cpp or 2.c or 2.py
├── 2.png
scripts/
└── P2-BouncingBalls/
    ├── 2_test.sh
    ├── marks.txt
    └── testcases/
        ├── 1.in
        ├── 1.out
        ├── 2.in
        ├── 2.out
        └── ... (more test cases)
```

### Prerequisites

- MPI installed (e.g., OpenMPI or MPICH)
- `mpic++`, `mpicc`, or `python3` accessible via command line
- Bash shell

### Run Instructions

1. **Navigate to the test directory**:

   ```bash
   cd scripts/P2-BouncingBalls
   ```

2. **Make the test script executable** (if not already):

   ```bash
   chmod +x 2_test.sh
   ```

3. **Run the test script**:

   ```bash
   ./2_test.sh
   ```

### What It Does

- Detects whether a `.cpp`, `.c`, or `.py` solution exists in `../../P2-BouncingBalls/`
- Compiles C++ or C code using `mpic++` or `mpicc`, or runs the Python file using `python3`
- Executes your program using 1 to 12 MPI processes for each test case
- Normalizes output formatting for fair comparison
- Compares your program's output with the expected output
- Awards marks for each passed test case based on `marks.txt`

### Expected Output (on success)

If your program is correct, you’ll see output like this:

```
Test case 1: PASSED
Test case 2: PASSED
Test case 3: PASSED
Test case 4: PASSED
Test case 5: PASSED
Test case 6: PASSED
Test case 7: PASSED
Test case 8: PASSED
Test case 9: PASSED
Test case 10: PASSED
Test case 11: PASSED
Test case 12: PASSED
Test case 13: PASSED
Test case 14: PASSED
Test case 15: PASSED
Final Score: 30.00/30
```

### 🧹 Cleanup

The script automatically deletes the compiled binary (`P2-BouncingBalls`) and `results/` directory after execution.

```bash
rm -rf 2 results/
```

If you want to inspect the intermediate results before cleanup, comment out or remove the `rm -rf` line at the end of `2_test.sh`.

# 3.  Avengers Distributed File System Quest 

This project implements a distributed file management system using MPI for parallelism and fault tolerance. The system provides functionalities such as uploading, retrieving, and searching files, along with heartbeat monitoring, failover, and recovery mechanisms.

---

## Solution Approach

### Parallelization Strategy

1. **Upload Functionality**:
   - The file is divided into chunks, and each chunk is distributed across worker nodes.
   - Chunks are replicated to ensure fault tolerance.
   - The master node coordinates the chunk distribution and metadata management.
   - Worker nodes handle the actual storage of chunks.

2. **Retrieve Functionality**:
   - The master node queries metadata to locate chunks for the requested file.
   - Worker nodes send the chunks back to the master node.
   - The master node reconstructs the file and provides it to the user.

3. **Search Functionality**:
   - The master node distributes the search request to all worker nodes holding chunks of the file.
   - Each worker node searches its local chunks for the requested word and reports results back to the master node.
   - Results are aggregated by the master and presented to the user.

4. **Heartbeat Monitoring**:
   - Worker nodes periodically send heartbeat signals to the master.
   - The master tracks the last heartbeat time for each worker to monitor their availability.

5. **Failover Mechanism**:
   - If a worker node fails (detected via missing heartbeats), the master initiates failover.
   - Replicated chunks are redistributed to available nodes to maintain redundancy.

6. **Recovery Mechanism**:
   - When a failed node recovers, the master updates the metadata and reintegrates the node into the system.

---

## Highlights of the Program

- **Distributed File Storage**:
  - Files are divided into chunks and distributed across multiple nodes.
  - Replication ensures high availability and fault tolerance.

- **Dynamic Load Balancing**:
  - The master node balances the workload across available nodes by tracking their loads.

- **Fault Tolerance**:
  - The system monitors node availability and performs failover to ensure service continuity.

- **Efficient Search**:
  - Parallel search allows fast retrieval of matching results across distributed data.

- **Scalability**:
  - The system supports adding nodes dynamically, improving scalability and performance.

---

## How Each Functionality is Parallelized

1. **Upload**:
   - The master node partitions the file and distributes chunks across worker nodes.
   - Each worker stores its assigned chunks and maintains replicas.

2. **Retrieve**:
   - The master queries metadata for chunk locations and requests retrieval from worker nodes.
   - Workers send their chunks to the master for reassembly.

3. **Search**:
   - The master broadcasts the search query to all workers holding chunks of the file.
   - Workers perform local searches and return results, which the master aggregates.

4. **Heartbeat Monitoring**:
   - Each worker sends periodic heartbeat signals to the master.
   - The master tracks heartbeats and marks nodes as failed if signals are not received.

5. **Failover**:
   - When a node fails, the master identifies replicated chunks stored on the failed node.
   
6. **Recovery**:
   - Recovered nodes register with the master, which updates the metadata to reintegrate them.

---

## Complexity Analysis

### Original Algorithm
- **Time Complexity**: (O(n)\) for single-threaded file processing, where \(n\) is the file size. The search as well as retrieve operation requires O(n) complexity as well
- **Space Complexity**: \(O(n)\) additional space for storing an entire file

### Parallel Algorithm
- **Time Complexity**: Search as well retrieve requires \(O(n / p)\), where \(p\) is the number of processes (worker nodes).
- **Space Complexity**: O(N*replication_factor), accounting for chunks processed concurrently, ensuring fault tolerance
- **Individual Complexity**:
  - Upload: \(O(p * REPLICATION\_FACTOR)\)
  - Retrieve: \(O(N)\), where \(N\) is the file size
  - Search: \(O(N/p)\), as all nodes are queried in parallel.

---

## How to Run

### 1. Compile the Code
```bash
mpic++ -std=c++17 ./3.cpp -o exec
```

### 2. Run the DFS System
```bash
mpirun -np <num_processors> ./exec
```

### 3. Testing with Provided Test Cases
Navigate to the relevant directory:
```bash
cd scripts
chmod +x ./3_test.sh
./3.sh
```
---

## Assumptions
1.	File Naming:
   - All file names in the system are unique. Duplicate file names are not allowed.
2.	File Content:
	- File content does not contain any punctuation marks.
3.	Word Size:
	- Words in the files are guaranteed to have a length of at least 1 character.
4.	System Configuration:
	- The number of worker nodes (p) is known and remains constant during system execution.


