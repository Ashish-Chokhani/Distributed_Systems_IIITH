PROTO_DIR    = protofiles
CLIENT_DIR   = client
SERVER_DIR   = server
LOADTEST_DIR = loadtest
ETCD_PORT = 2379

# List all proto files that need code generation.
PROTO_FILES  = $(PROTO_DIR)/lb.proto $(PROTO_DIR)/service.proto
PROTO_OUT_DIR = .

GO_FLAGS = --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
           --go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative

.PHONY: proto server client loadtest clean

# Generate Go code from all proto files.
proto:
	protoc $(GO_FLAGS) $(PROTO_FILES)

etcd:
	@echo "Checking if etcd is running on port $(ETCD_PORT)..."
	@if ! nc -z 127.0.0.1 $(ETCD_PORT); then \
		echo "Starting fresh etcd instance..."; \
		rm -rf /tmp/etcd-$(ETCD_PORT); \
		nohup etcd --advertise-client-urls http://127.0.0.1:$(ETCD_PORT) \
		--listen-client-urls http://127.0.0.1:$(ETCD_PORT) \
		--data-dir=/tmp/etcd-$(ETCD_PORT) > etcd.log 2>&1 & \
		sleep 2; \
	fi

# Run the backend servers.
server: etcd
	go run $(SERVER_DIR)/main.go -servers=3 -startport=50051 -lb=localhost:50050

# Run the clients.
client:
	go run $(CLIENT_DIR)/main.go -clients=4 -duration=30 -strategy=round_robin -lb=localhost:50050

# Clean up generated proto files.
clean:
	find $(PROTO_DIR) -name "*.pb.go" -delete