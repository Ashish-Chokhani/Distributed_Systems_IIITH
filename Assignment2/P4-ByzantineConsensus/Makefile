# Directories
PROTO_DIR    = protofiles
CLIENT_DIR   = client
SERVER_DIR   = server

# Proto files
PROTO_FILE   = $(PROTO_DIR)/bft.proto
PROTO_OUT_DIR = .

GO_FLAGS = --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
           --go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative

# Default port for server
SERVER_PORT  = 50051

.PHONY: proto server client clean run_simulation

# Generate Go code from proto file
proto:
	protoc $(GO_FLAGS) $(PROTO_FILE)

# Run the Byzantine server
server:
	go run $(SERVER_DIR)/main.go --port=$(SERVER_PORT)

# Run the client with default settings
client:
	go run $(CLIENT_DIR)/main.go --server=localhost:$(SERVER_PORT)

# Run a simulation with specific parameters
# Example: make run_simulation n=7 t=2 decision=ATTACK commander-traitor=true
run_simulation:
	go run $(CLIENT_DIR)/main.go --server=localhost:$(SERVER_PORT) \
		--n=$(n) --t=$(t) --decision=$(decision) --commander-traitor=$(commander-traitor)

# Build the server binary
build_server:
	go build -o bft_server $(SERVER_DIR)/main.go

# Build the client binary
build_client:
	go build -o bft_client $(CLIENT_DIR)/main.go

# Clean up generated files
clean:
	@rm -f bft_server bft_client
	@find $(PROTO_DIR) -name "*.pb.go" -delete