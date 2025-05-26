#include <bits/stdc++.h>
#include <mpi.h>
using namespace std;

// L R U D
const int L = 0, R = 1, U = 2, D = 3;
char what[4] = {'L', 'R', 'U', 'D'};

int rotateBy90(int d) {
    return (d == L) ? U : (d == R) ? D : (d == U) ? R : L;
}

int reverseDirection(int d) {
    return (d == L) ? R : (d == R) ? L : (d == U) ? D : U;
}

array<int, 2> findNextPos(int x, int y, int d, int N, int M) {
    if (d == R) return {x, (y + 1) % M};
    if (d == L) return {x, (y - 1 + M) % M};
    if (d == D) return {(x + 1) % N, y};
    return {(x - 1 + N) % N, y};
}

void findPos(int N, int M, int T, vector<array<int, 3>>& pos, vector<array<int, 3>>& ans, int rank, int size) {
    for (int t = 0; t < T; t++) {
        vector<array<int, 3>> next_pos;
        map<array<int, 3>, int> local_collisions, global_collisions;

        for (auto& [x, y, d] : pos) local_collisions[{x, y, d}]++;

        vector<int> send_data;
        for (auto& [key, value] : local_collisions) {
            send_data.push_back(key[0]); send_data.push_back(key[1]); send_data.push_back(key[2]); send_data.push_back(value);
        }
        int send_size = send_data.size();
        vector<int> recv_sizes(size), recv_displs(size);

        MPI_Allgather(&send_size, 1, MPI_INT, recv_sizes.data(), 1, MPI_INT, MPI_COMM_WORLD);

        recv_displs[0] = 0;
        for (int i = 1; i < size; i++) recv_displs[i] = recv_displs[i - 1] + recv_sizes[i - 1];

        int total_size = accumulate(recv_sizes.begin(), recv_sizes.end(), 0);
        vector<int> recv_data(total_size);
        
        MPI_Allgatherv(send_data.data(), send_size, MPI_INT,
                       recv_data.data(), recv_sizes.data(), recv_displs.data(), MPI_INT, MPI_COMM_WORLD);
        
        for (int i = 0; i < total_size; i += 4) {
            array<int, 3> key = {recv_data[i], recv_data[i + 1], recv_data[i + 2]};
            global_collisions[key] += recv_data[i + 3];
        }
        
        for (auto& [x, y, d] : pos) {
            auto [X, Y] = findNextPos(x, y, d, N, M);

            int cnt_collisions = 0;
            cnt_collisions += global_collisions[{(X - 1 + N) % N, Y, D}];
            cnt_collisions += global_collisions[{(X + 1) % N, Y, U}];
            cnt_collisions += global_collisions[{X, (Y - 1 + M) % M, R}];
            cnt_collisions += global_collisions[{X, (Y + 1) % M, L}];

            if (cnt_collisions == 2) next_pos.push_back({X, Y, rotateBy90(d)});
            else if (cnt_collisions == 4) next_pos.push_back({X, Y, reverseDirection(d)});
            else next_pos.push_back({X, Y, d});
        }
        pos = next_pos;
    }
    ans = pos;
}

int main(int argc, char** argv) {
    if (argc < 3) {
        cerr << "Usage: " << argv[0] << " <input_file> <output_file>" << endl;
        return EXIT_FAILURE;
    }

    string input_file = argv[1];
    string output_file = argv[2];

    MPI_Init(&argc, &argv);

    int rank, size;
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &size);

    int N, M, K, T;
    vector<array<int, 3>> pos;

    if (rank == 0) {
        ifstream input(input_file);
        if (!input.is_open()) {
            cerr << "Failed to open input file: " << input_file << endl;
            MPI_Abort(MPI_COMM_WORLD, EXIT_FAILURE);
        }

        input >> N >> M >> K >> T;
        pos.resize(K);
        for (int i = 0; i < K; i++) {
            input >> pos[i][0] >> pos[i][1];
            char dir;
            input >> dir;
            if (dir == 'L') pos[i][2] = L;
            else if (dir == 'R') pos[i][2] = R;
            else if (dir == 'U') pos[i][2] = U;
            else pos[i][2] = D;
        }

        int chunk_size = (K + size - 1) / size;
        for (int i = 1; i < size; i++) {
            int start = i * chunk_size;
            int end = min((i + 1) * chunk_size, K);
            int count = end - start;

            if (count > 0) {
                vector<int> flat_data(count * 3);
                for (int j = 0; j < count; j++) {
                    flat_data[j * 3] = pos[start + j][0];
                    flat_data[j * 3 + 1] = pos[start + j][1];
                    flat_data[j * 3 + 2] = pos[start + j][2];
                }

                MPI_Send(&count, 1, MPI_INT, i, 0, MPI_COMM_WORLD);
                MPI_Send(flat_data.data(), count * 3, MPI_INT, i, 1, MPI_COMM_WORLD);
            } else {
                int zero = 0;
                MPI_Send(&zero, 1, MPI_INT, i, 0, MPI_COMM_WORLD);
            }
        }

        pos.resize(chunk_size);
    } else {
        int count;
        MPI_Recv(&count, 1, MPI_INT, 0, 0, MPI_COMM_WORLD, MPI_STATUS_IGNORE);

        if (count > 0) {
            vector<int> flat_data(count * 3);
            MPI_Recv(flat_data.data(), count * 3, MPI_INT, 0, 1, MPI_COMM_WORLD, MPI_STATUS_IGNORE);
            pos.resize(count);
            for (int i = 0; i < count; i++) {
                pos[i][0] = flat_data[i * 3];
                pos[i][1] = flat_data[i * 3 + 1];
                pos[i][2] = flat_data[i * 3 + 2];
            }
        }
    }

    MPI_Bcast(&N, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&M, 1, MPI_INT, 0, MPI_COMM_WORLD);
    MPI_Bcast(&T, 1, MPI_INT, 0, MPI_COMM_WORLD);

    vector<array<int, 3>> ans;
    findPos(N, M, T, pos, ans, rank, size);

    vector<int> local_flat_data;
    for (auto& [x, y, d] : ans) {
        local_flat_data.push_back(x);
        local_flat_data.push_back(y);
        local_flat_data.push_back(d);
    }

    int local_count = local_flat_data.size();
    vector<int> recv_counts(size), displs(size);

    MPI_Gather(&local_count, 1, MPI_INT, recv_counts.data(), 1, MPI_INT, 0, MPI_COMM_WORLD);

    if (rank == 0) {
        displs[0] = 0;
        for (int i = 1; i < size; i++) {
            displs[i] = displs[i - 1] + recv_counts[i - 1];
        }
    }

    int total_count = accumulate(recv_counts.begin(), recv_counts.end(), 0);
    vector<int> global_flat_data(total_count);

    MPI_Gatherv(local_flat_data.data(), local_count, MPI_INT, global_flat_data.data(), recv_counts.data(), displs.data(),
                MPI_INT, 0, MPI_COMM_WORLD);

    if (rank == 0) {
        ofstream output(output_file);
        if (!output.is_open()) {
            cerr << "Failed to open output file: " << output_file << endl;
            MPI_Abort(MPI_COMM_WORLD, EXIT_FAILURE);
        }

        for (int i = 0; i < total_count; i += 3) {
            output << global_flat_data[i] << " " << global_flat_data[i + 1] << " " << what[global_flat_data[i + 2]] << endl;
        }
    }

    MPI_Finalize();
    return EXIT_SUCCESS;
}



