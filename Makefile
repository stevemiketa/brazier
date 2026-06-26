GOPATH    := $(shell go env GOPATH)
export PATH := $(PATH):$(GOPATH)/bin

PROTO_DIR    := proto
GEN_DIR      := proto/gen
COVERAGE_OUT := coverage.out

GO_SRCS := $(shell find . -name '*.go' -not -path './vendor/*')

PROTO_SRCS := $(PROTO_DIR)/pipeline.proto $(PROTO_DIR)/runner.proto $(PROTO_DIR)/api.proto
PROTO_GENS := $(GEN_DIR)/pipeline.pb.go $(GEN_DIR)/runner.pb.go $(GEN_DIR)/api.pb.go \
              $(GEN_DIR)/api_grpc.pb.go



# Install all build/codegen tools
.PHONY: setup
setup:
	brew install protobuf
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

.PHONY: proto
proto: $(PROTO_GENS)
$(PROTO_GENS): $(PROTO_SRCS)
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GEN_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_SRCS)

.PHONY: test
test: $(COVERAGE_OUT)
$(COVERAGE_OUT): $(GO_SRCS)
	go test -coverprofile=$(COVERAGE_OUT) ./...
	cd sdk/go && go test ./...

.PHONY: tidy
tidy:
	go mod tidy
	cd sdk/go && go mod tidy
