#include <mpi.h>
#include <bits/stdc++.h>

using namespace std;

const int INF = numeric_limits<int>::max();


void distributedBFS(int rank, int size, int N, int source,
                    const vector<vector<int>>& graph, vector<int>& dist) {
    vector<int> Frontier;               // Current frontier
    vector<int> NextFrontier;           // Next frontier
    dist.assign(N, INF);
    int level = 0;

    // Initialize the source node
    if (rank == source % size) {
        Frontier.push_back(source);
        dist[source] = level;
    }

    // Broadcast the initial `dist[source]` to all processes
    int sourceDist = dist[source]; // Only the owner of `source` will have the correct value
    MPI_Bcast(&sourceDist, 1, MPI_INT, source % size, MPI_COMM_WORLD);
    dist[source] = sourceDist;

    bool hasWork = !Frontier.empty();
    while (true) {
        vector<int> SendBuf;
        // Process the current frontier and prepare the send buffer
        for (int u : Frontier) for (int v : graph[u]) {
            if (dist[v] == INF) {
                int owner = v % size;
                if (owner == rank) {
                    if (dist[v] == INF) {
                        dist[v] = level + 1;
                        NextFrontier.push_back(v);
                    }
                } 
                else SendBuf.push_back(v);
            }
        }

        // Gather and exchange the data
        vector<int> RecvBuf;
        int sendSize = SendBuf.size();
        vector<int> recvSizes(size);

        MPI_Allgather(&sendSize, 1, MPI_INT, recvSizes.data(), 1, MPI_INT, MPI_COMM_WORLD);

        int totalRecv = accumulate(recvSizes.begin(), recvSizes.end(), 0);
        RecvBuf.resize(totalRecv);

        vector<int> displs(size, 0);
        for (int i = 1; i < size; ++i) displs[i] = displs[i - 1] + recvSizes[i - 1];
        
        MPI_Allgatherv(SendBuf.data(), sendSize, MPI_INT, RecvBuf.data(),recvSizes.data(), displs.data(), MPI_INT, MPI_COMM_WORLD);

        // Process the received vertices
        for (int v : RecvBuf) {
            int owner = v % size;
            if (dist[v] == INF) {
                dist[v] = level + 1;
                if (owner == rank) {
                    NextFrontier.push_back(v);  // Add to local frontier
                }
            }
        }

        // Synchronize dist[] across all processes
        vector<int> globalDist(N, INF);
        MPI_Allreduce(dist.data(), globalDist.data(), N, MPI_INT, MPI_MIN, MPI_COMM_WORLD);
        dist = move(globalDist);
        
        // Synchronize across all processes
        Frontier = move(NextFrontier);
        NextFrontier.clear();
        
        // Check if there's any work left
        hasWork = !Frontier.empty() ? 1 : 0;
        int globalHasWork = 0;
        MPI_Allreduce(&hasWork, &globalHasWork, 1, MPI_INT, MPI_LOR, MPI_COMM_WORLD);

        if (!globalHasWork) break;
        ++level;
    }
}

int main(int argc, char** argv) {
    MPI_Init(&argc, &argv);

    int rank, size;
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &size);

    if (argc != 3) {
        if (rank == 0) {
            cerr << "Usage: mpirun -np <num_processes> <executable> <input_file> <output_file>" << endl;
        }
        MPI_Finalize();
        return 1;
    }

    string inputFile = argv[1];
    string outputFile = argv[2];

    int N, M, R, L, K;
    vector<vector<int>> graph;
    vector<int> explorers;

    if (rank == 0) {
        ifstream infile(inputFile);
        if (!infile) {
            cerr << "Error: Unable to open input file." << endl;
            MPI_Abort(MPI_COMM_WORLD, 1);
        }

        infile >> N >> M;
        graph.resize(N);
        for (int i = 0; i < M; ++i) {
            int u, v, d;
            infile >> u >> v >> d;
            if (d == 1) {
                graph[u].push_back(v);
                graph[v].push_back(u);
            } else {
                graph[v].push_back(u);
            }
        }
        infile >> K;
        explorers.resize(K);
        for (int i = 0; i < K; ++i) infile >> explorers[i];
        infile >> R >> L;
        vector<int> blocked(L);
        for (int i = 0; i < L; ++i) infile >> blocked[i];
        for (int blockedNode : blocked) graph[blockedNode].clear();
        infile.close();
    }

    MPI_Bcast(&N, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&M, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&R, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&K, 1, MPI_INT, 0, MPI_COMM_WORLD);

    if (rank != 0) {
        graph.resize(N);
        explorers.resize(K);
    }

    for (int i = 0; i < N; ++i) {
        int edgesSize = (rank == 0) ? graph[i].size() : 0;
        MPI_Bcast(&edgesSize, 1, MPI_INT, 0, MPI_COMM_WORLD);
        if (rank != 0) graph[i].resize(edgesSize);
        MPI_Bcast(graph[i].data(), edgesSize, MPI_INT, 0, MPI_COMM_WORLD);
    }

    MPI_Bcast(explorers.data(), K, MPI_INT, 0, MPI_COMM_WORLD);

    vector<int> dist(N, INF);
    distributedBFS(rank, size, N, R, graph, dist);

    if (rank == 0) {
        ofstream outfile(outputFile);
        if (!outfile) {
            cerr << "Error: Unable to open output file." << endl;
            MPI_Abort(MPI_COMM_WORLD, 1);
        }
        for (int u : explorers) {
            outfile << (dist[u] == INF ? -1 : dist[u]) << " ";
        }
        outfile << endl;
        outfile.close();
    }

    MPI_Finalize();
    return 0;
}