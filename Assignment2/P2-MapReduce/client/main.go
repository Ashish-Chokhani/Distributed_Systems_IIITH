package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
)

// Master struct
type Master struct {
	pb.UnimplementedMasterServer
	numMappers  int
	numReducers int
	Mode        string // Add this
}

// AssignMapTask starts a mapper and assigns a map task
func (m *Master) AssignMapTask(ctx context.Context, req *pb.MapRequest) (*pb.TaskResponse, error) {
	fmt.Printf("Assigning map task %d for file %s\n", req.MapTaskId, req.Filename)

	port := 56052 + int(req.MapTaskId)
	m.startWorker(port)

	conn, err := grpc.Dial(fmt.Sprintf(":%d", port), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pb.NewWorkerClient(conn)
	return client.Map(ctx, req)
}

// AssignReduceTask starts a reducer and assigns a reduce task
func (m *Master) AssignReduceTask(ctx context.Context, req *pb.ReduceRequest) (*pb.TaskResponse, error) {
	fmt.Printf("Assigning reduce task %d\n", req.ReduceTaskId)

	port := 56052 + m.numMappers + int(req.ReduceTaskId)
	m.startWorker(port)

	conn, err := grpc.Dial(fmt.Sprintf(":%d", port), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pb.NewWorkerClient(conn)
	return client.Reduce(ctx, req)
}

// startWorker launches a worker on the given port
func (m *Master) startWorker(port int) {
	cmd := exec.Command("go", "run", "server/main.go", fmt.Sprintf("%d", port))
	cmd.Stdout = nil
	cmd.Stderr = nil

	err := cmd.Start()
	if err != nil {
		fmt.Printf("Failed to start worker on port %d: %v\n", port, err)
	}

	time.Sleep(1 * time.Second)
}

// startMaster starts the master server
func startMaster(numMappers, numReducers int, mode string) {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}

	grpcServer := grpc.NewServer()
	pb.RegisterMasterServer(grpcServer, &Master{numMappers: numMappers, numReducers: numReducers, Mode: mode})

	fmt.Println("Master server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run client/main.go <numReducers> <mode>")
		return
	}

	numReducers, _ := strconv.Atoi(os.Args[1])
	mode := os.Args[2]
	inputDir := "dataset"
	files, err := filepath.Glob(filepath.Join(inputDir, "*.txt"))
	if err != nil {
		fmt.Printf("Failed to read input directory: %v\n", err)
		return
	}

	numMappers := len(files)
	if numMappers == 0 {
		fmt.Println("No input files found")
		return
	}

	go startMaster(numMappers, numReducers, mode)

	time.Sleep(2 * time.Second) // Let master start up

	var wg sync.WaitGroup

	// Start mappers
	for i, file := range files {
		wg.Add(1)
		go func(i int, file string) {
			defer wg.Done()
			conn, err := grpc.Dial(":50051", grpc.WithInsecure())
			if err != nil {
				fmt.Printf("Failed to connect to master: %v\n", err)
				return
			}
			defer conn.Close()

			client := pb.NewMasterClient(conn)
			req := &pb.MapRequest{MapTaskId: int32(i), Filename: file, NumReducers: int32(numReducers), Mode: mode}
			res, err := client.AssignMapTask(context.Background(), req)
			if err != nil {
				fmt.Printf("Map task failed: %v\n", err)
				return
			}
			fmt.Println(res.Message)
		}(i, file)
		time.Sleep(1 * time.Second)
	}

	wg.Wait()

	// Start reducers
	for i := 0; i < numReducers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			conn, err := grpc.Dial(":50051", grpc.WithInsecure())
			if err != nil {
				fmt.Printf("Failed to connect to master: %v\n", err)
				return
			}
			defer conn.Close()

			client := pb.NewMasterClient(conn)
			req := &pb.ReduceRequest{ReduceTaskId: int32(i), Mode: mode}
			res, err := client.AssignReduceTask(context.Background(), req)
			if err != nil {
				fmt.Printf("Reduce task failed: %v\n", err)
				return
			}
			fmt.Println(res.Message)
		}(i)
		time.Sleep(1 * time.Second)
	}

	wg.Wait()
}