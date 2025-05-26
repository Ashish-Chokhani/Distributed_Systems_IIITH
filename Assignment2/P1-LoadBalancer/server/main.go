package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
)

// global counter for concurrent tasks handled by this server instance.
// var concurrentTasks int32

// threshold for maximum concurrent tasks before marking the server as unavailable.
const maxConcurrentTasks = 5

// fibonacci computes the n-th Fibonacci number recursively.
// Note: This implementation is intentionally inefficient to simulate CPU load.
func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

type backendServer struct {
    pb.UnimplementedBackendServiceServer
    serverAddr string
    concurrentTasks int32  // Move concurrentTasks inside the struct
}

// Compute processes the task request.
// If the task starts with "fibonacci:", it parses the number and computes the Fibonacci value.
// Otherwise, it returns a default message.
// It also updates the concurrent task counter.
func (s *backendServer) Compute(ctx context.Context, req *pb.TaskRequest) (*pb.TaskResponse, error) {
    // Increment concurrent tasks counter.
    atomic.AddInt32(&s.concurrentTasks, 1)
	defer atomic.AddInt32(&s.concurrentTasks, -1)

    task := req.Task
    parts := strings.Split(  task, ":")
    if len(parts) > 1 {
        task = strings.TrimSpace(strings.Join(parts[1:], ":"))
    }

    var result string

    // Check if the task is a Fibonacci computation.
    if strings.HasPrefix(task, "fibonacci:") {
        taskParts := strings.Split(task, ":")
        if len(taskParts) == 2 {
            n, err := strconv.Atoi(strings.TrimSpace(taskParts[1]))
            if err != nil {
                result = "Error: invalid number for fibonacci"
            } else {
                fib := fibonacci(n)
                result = fmt.Sprintf("Fibonacci(%d) = %d", n, fib)
            }
        } else {
            result = "Error: invalid task format"
        }
    } else {
        result = "Error: unknown task"
    }

    return &pb.TaskResponse{Result: result}, nil
}


// simulateBackendServer starts one backend server on the given port, registers it with the LB server,
// and periodically reports its actual load and availability (based on concurrent task count).
func simulateBackendServer(port int, lbAddress string, wg *sync.WaitGroup) {
	defer wg.Done()
	serverAddr := fmt.Sprintf("127.0.0.1:%d", port)

	// Connect to LB server for registration.
	conn, err := grpc.Dial(lbAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("Server %s: failed to connect to LB: %v", serverAddr, err)
		return
	}
	lbClient := pb.NewLoadBalancerClient(conn)
	_, err = lbClient.RegisterServer(context.Background(), &pb.ServerInfo{Address: serverAddr})
	if err != nil {
		log.Printf("Server %s: registration error: %v", serverAddr, err)
	}
	conn.Close()

	// Periodically report the true load and availability.
	serverInstance := &backendServer{serverAddr: serverAddr}
	
	go func(s *backendServer) {
		for {
			currentLoad := int(atomic.LoadInt32(&s.concurrentTasks)) // Use server-specific counter
			if currentLoad != 0 {
				log.Printf("%s has Current Load %d:", s.serverAddr, currentLoad)
			}
			available := currentLoad < maxConcurrentTasks
	
			conn, err := grpc.Dial(lbAddress, grpc.WithInsecure())
			if err != nil {
				log.Printf("Server %s: failed to reconnect to LB: %v", s.serverAddr, err)
				time.Sleep(5 * time.Second)
				continue
			}
			lbClient := pb.NewLoadBalancerClient(conn)
			_, err = lbClient.ReportLoad(context.Background(), &pb.ServerLoad{
				Address:   s.serverAddr,
				Load:      int32(currentLoad),
				Available: available,
			})
			if err != nil {
				log.Printf("Server %s: report load error: %v", s.serverAddr, err)
			}
			conn.Close()
			time.Sleep(5 * time.Second)
		}
	}(serverInstance)

	// Start backend gRPC server.
	lis, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Printf("Server %s: listen error: %v", serverAddr, err)
		return
	}
	grpcServer := grpc.NewServer()
	pb.RegisterBackendServiceServer(grpcServer, serverInstance)
	log.Printf("Backend server running on %s", serverAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("Server %s: serve error: %v", serverAddr, err)
	}
}

func main() {
	// Seed the random number generator (if needed elsewhere).
	// rand.Seed(time.Now().UnixNano())
	numServers := flag.Int("servers", 10, "Number of backend servers to spawn")
	startPort := flag.Int("startport", 50051, "Starting port for backend servers")
	lbAddress := flag.String("lb", "127.0.0.1:50050", "Load Balancer address")
	flag.Parse()

	var wg sync.WaitGroup
	for i := 0; i < *numServers; i++ {
		wg.Add(1)
		go simulateBackendServer(*startPort+i, *lbAddress, &wg)
		// Small delay between server spawns.
		time.Sleep(100 * time.Millisecond)
	}

	// Wait forever (or you can wait on wg if you plan to stop servers gracefully).
	wg.Wait()
}