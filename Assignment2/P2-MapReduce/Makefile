PROTO_DIR      = protofiles
PROTO_FILES    = $(PROTO_DIR)/mapreduce.proto
PROTO_OUT_DIR  = .

GO_FLAGS       = --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
                 --go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative

INPUT_DIR      = dataset
NUM_FILES      = $(shell ls -1 $(INPUT_DIR) | wc -l)
NUM_REDUCERS   ?= 2 # Default, can be overridden by CLI
MODE           ?= word_count # Default mode

.PHONY: proto master worker client clean

proto:
	@protoc $(GO_FLAGS) $(PROTO_FILES)

client:
	@go run client/main.go $(NUM_REDUCERS) $(MODE)

clean:
#	@find . -name "*.pb.go" -delete
	@rm -f mr-*.txt out-*.txt
	@rm -f $(INPUT_DIR)/*.txt 2>/dev/null || true