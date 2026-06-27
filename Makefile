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
	brew install protobuf hivemind
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go get modernc.org/sqlite github.com/lib/pq github.com/spf13/cobra
	cd sdk/ts && npm install
	cd web && npm install

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
TS_SRCS := $(shell find sdk/ts/src -name '*.ts')

$(COVERAGE_OUT): $(GO_SRCS)
	go test -coverprofile=$(COVERAGE_OUT) ./...
	cd sdk/go && go test ./...
	cd sdk/ts && npm test

.PHONY: tidy
tidy:
	go mod tidy
	cd sdk/go && go mod tidy

# Build the master service binary
.PHONY: build-master
build-master:
	go build -o bin/master ./cmd/master

# Run the master service (gRPC on :9000, HTTP webhook on :8080)
.PHONY: master
master:
	go run ./cmd/master

# Run the Vite dev server for the web UI (proxies gRPC-web to :9000)
.PHONY: web-dev
web-dev:
	cd web && npm install && npm run dev

# Run master + web UI concurrently (requires: brew install hivemind or foreman)
.PHONY: dev
dev:
	@command -v hivemind >/dev/null 2>&1 || { echo "install hivemind: brew install hivemind"; exit 1; }
	hivemind Procfile

Procfile:
	@printf 'master: go run ./cmd/master\nweb: cd web && npm run dev\n' > Procfile
