package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
)

func main() {
	// Parse command-line arguments
	numClients := flag.Int("clients", 50, "Number of concurrent clients")
	testDuration := flag.Int("duration", 30, "Test duration in seconds")
	strategyStr := flag.String("strategy", "least_load", "Load balancing strategy: pick_first, round_robin, least_load")
	lbAddress := flag.String("lb", "127.0.0.1:50050", "Load Balancer address")
	flag.Parse()

	// Map strategy string to proto enum
	var lbStrategy pb.LoadBalanceStrategy
	switch *strategyStr {
	case "pick_first":
		lbStrategy = pb.LoadBalanceStrategy_PICK_FIRST
	case "round_robin":
		lbStrategy = pb.LoadBalanceStrategy_ROUND_ROBIN
	case "least_load":
		lbStrategy = pb.LoadBalanceStrategy_LEAST_LOAD
	default:
		lbStrategy = pb.LoadBalanceStrategy_LEAST_LOAD
	}

	// Connect to the Load Balancer (LB)
	lbConn, err := grpc.Dial(*lbAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to Load Balancer: %v", err)
	}
	defer lbConn.Close()
	lbClient := pb.NewLoadBalancerClient(lbConn)

	var totalRequests int64
	var totalLatency int64
	var wg sync.WaitGroup
	stopTime := time.Now().Add(time.Duration(*testDuration) * time.Second)

	// Launch concurrent clients
	for i := 0; i < *numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			task := fmt.Sprintf("fibonacci:%d",40)
			for time.Now().Before(stopTime) {
				// Get the best backend server dynamically for each request
				serverInfo, err := lbClient.GetBestServer(context.Background(), &pb.BalanceRequest{Strategy: lbStrategy})
				// log.Printf("Server %s computing Fibonacci %d",serverInfo.Address, clientID)

				if err != nil {
					log.Printf("Client %d: error getting best server: %v", clientID, err)
					continue
				}
				// Connect to the selected backend server
				backendConn, err := grpc.Dial(serverInfo.Address, grpc.WithInsecure())
				if err != nil {
					log.Printf("Client %d: error connecting to backend %s: %v", clientID, serverInfo.Address, err)
					continue
				}
				backendClient := pb.NewBackendServiceClient(backendConn)

				start := time.Now()
				_, err = backendClient.Compute(context.Background(), &pb.TaskRequest{Task: fmt.Sprintf("Client %d: %s", clientID, task)})
				backendConn.Close() // Close the connection after each request
				if err != nil {
					log.Printf("Client %d: compute error: %v", clientID, err)
					continue
				}

				latency := time.Since(start)
				atomic.AddInt64(&totalRequests, 1)
				atomic.AddInt64(&totalLatency, latency.Milliseconds())

				// Add a small delay to avoid overwhelming the system
				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}
	wg.Wait()

	// Calculate & log test results
	avgLatency := float64(totalLatency) / float64(totalRequests)
	throughput := float64(totalRequests) / float64(*testDuration)
	log.Printf("Load Test Results: Total Requests: %d, Average Latency: %.2f ms, Throughput: %.2f req/sec",
		totalRequests, avgLatency, throughput)
}