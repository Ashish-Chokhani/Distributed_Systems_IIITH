package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	pb "github.com/example/protofiles"
	"google.golang.org/grpc"
)

const (
	etcdServersPrefix = "/lb/servers/"
	etcdLBKey         = "/lb/lbserver"
)

type ServerStatus struct {
	Address   string `json:"address"`
	Load      int    `json:"load"`
	Available bool   `json:"available"`
}

type LoadBalancer struct {
	pb.UnimplementedLoadBalancerServer
	rrIndex    int          // for round-robin selection
	etcdClient *clientv3.Client
	mu         sync.Mutex // protects rrIndex
}

// RegisterServer writes the server's JSON status into etcd.
func (lb *LoadBalancer) RegisterServer(ctx context.Context, req *pb.ServerInfo) (*pb.RegisterResponse, error) {
	status := ServerStatus{
		Address:   req.Address,
		Load:      0,
		Available: true,
	}
	data, err := json.Marshal(status)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status: %v", err)
	}
	_, err = lb.etcdClient.Put(ctx, etcdServersPrefix+req.Address, string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to put key in etcd: %v", err)
	}
	log.Printf("Registered server: %s\n", req.Address)
	return &pb.RegisterResponse{Message: "Registered successfully"}, nil
}

// ReportLoad updates a server’s load and availability in etcd.
func (lb *LoadBalancer) ReportLoad(ctx context.Context, req *pb.ServerLoad) (*pb.LoadResponse, error) {
	status := ServerStatus{
		Address:   req.Address,
		Load:      int(req.Load),
		Available: req.Available,
	}
	data, err := json.Marshal(status)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status: %v", err)
	}
	_, err = lb.etcdClient.Put(ctx, etcdServersPrefix+req.Address, string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to update key in etcd: %v", err)
	}
	log.Printf("Updated server %s: load=%d, available=%v\n", req.Address, req.Load, req.Available)
	return &pb.LoadResponse{Message: "Load updated"}, nil
}

// GetBestServer queries etcd for all backend servers and selects one based on the requested strategy.
func (lb *LoadBalancer) GetBestServer(ctx context.Context, req *pb.BalanceRequest) (*pb.ServerInfo, error) {
	// Query etcd for all keys with the servers prefix.
	resp, err := lb.etcdClient.Get(ctx, etcdServersPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to query etcd: %v", err)
	}
	var availableServers []ServerStatus
	for _, kv := range resp.Kvs {
		var status ServerStatus
		if err := json.Unmarshal(kv.Value, &status); err != nil {
			log.Printf("failed to unmarshal key %s: %v", string(kv.Key), err)
			continue
		}
		if status.Available {
			availableServers = append(availableServers, status)
		}
	}
	if len(availableServers) == 0 {
		return nil, fmt.Errorf("no available servers")
	}

	switch req.Strategy {
	case pb.LoadBalanceStrategy_PICK_FIRST:
		selected := availableServers[0]
		log.Printf("[PICK_FIRST] Selected server: %s with load: %d", selected.Address,selected.Load)
		return &pb.ServerInfo{Address: selected.Address}, nil

	case pb.LoadBalanceStrategy_ROUND_ROBIN:
		lb.mu.Lock()
		if len(availableServers) == 0 {
			return nil, fmt.Errorf("no available servers")
		}
		
		if lb.rrIndex >= len(availableServers) {
			lb.rrIndex = 0
		}

		selected := availableServers[lb.rrIndex%len(availableServers)]
		lb.rrIndex++
		lb.mu.Unlock()
		log.Printf("[ROUND_ROBIN] Selected server: %s with load: %d", selected.Address, selected.Load)
		return &pb.ServerInfo{Address: selected.Address}, nil

	case pb.LoadBalanceStrategy_LEAST_LOAD:
		// Log the available servers and their loads.
		best := availableServers[0]
		for _, s := range availableServers {
			if s.Load < best.Load {
				best = s
			}
		}
		// log.Printf("[LEAST_LOAD] Selected server: %s with load: %d", best.Address, best.Load)
		return &pb.ServerInfo{Address: best.Address}, nil

	default:
		return nil, fmt.Errorf("unknown strategy")
	}
}

// registerLBServer registers the LB server itself in etcd with a TTL lease.
func registerLBServer(etcdClient *clientv3.Client, addr string, leaseTTL int64) (clientv3.LeaseID, error) {
	ctx := context.Background()
	leaseResp, err := etcdClient.Grant(ctx, leaseTTL)
	if err != nil {
		return 0, err
	}
	_, err = etcdClient.Put(ctx, etcdLBKey, addr, clientv3.WithLease(leaseResp.ID))
	if err != nil {
		return 0, err
	}
	// Keep the lease alive indefinitely.
	ch, err := etcdClient.KeepAlive(ctx, leaseResp.ID)
	if err != nil {
		return 0, err
	}
	go func() {
		for range ch {
			// Drain the keepalive channel.
		}
	}()
	return leaseResp.ID, nil
}

func main() {
	// Create an etcd client.
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}

	lb := &LoadBalancer{
		etcdClient: etcdClient,
	}

	// Register the LB server itself in etcd.
	lbAddress := "127.0.0.1:50050"
	_, err = registerLBServer(etcdClient, lbAddress, 10)
	if err != nil {
		log.Fatalf("Failed to register LB server in etcd: %v", err)
	}

	listener, err := net.Listen("tcp", ":50050")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLoadBalancerServer(grpcServer, lb)

	log.Println("Load Balancer running on port 50050")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}